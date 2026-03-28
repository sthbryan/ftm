package web

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/process"
)

type Server struct {
	manager    *process.Manager
	config     *config.Config
	httpServer *http.Server
	port       int
	clients    map[chan string]bool
	clientsMu  sync.RWMutex
	handlers   *Handlers
}

func NewServer(manager *process.Manager, cfg *config.Config) *Server {
	s := &Server{
		manager:  manager,
		config:   cfg,
		clients:  make(map[chan string]bool),
	}
	s.handlers = NewHandlers(manager, cfg, s)
	return s
}

func (s *Server) findPort() int {
	if s.config.WebPort > 0 {
		return s.config.WebPort
	}
	for port := 40500; port <= 40550; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	return 0
}

func (s *Server) Start() error {
	port := s.findPort()
	if port == 0 {
		return fmt.Errorf("no available port found")
	}
	s.port = port
	s.config.WebPort = port
	s.config.Save()

	mux := s.setupRoutes()
	s.setupMiddleware(mux)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go s.broadcastLoop()
	go s.installProgressLoop()
	go s.httpServer.ListenAndServe()
	return nil
}

func (s *Server) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/", s.handlers.Route)

	webDist := filepath.Join("web-svelte", "dist")
	var staticFS fs.FS
	if _, err := os.Stat(webDist); err == nil {
		staticFS, _ = fs.Sub(os.DirFS(webDist), ".")
	} else {
		staticFS, _ = fs.Sub(staticFiles, "static")
	}
	fileServer := http.FileServer(http.FS(staticFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			return
		}
		path := r.URL.Path
		if path != "/" && !strings.Contains(path, ".") {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})

	return mux
}

func (s *Server) setupMiddleware(mux *http.ServeMux) {
}

func (s *Server) Stop() error {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*5e9)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) Port() int {
	return s.port
}

func (s *Server) URL() string {
	return fmt.Sprintf("http://localhost:%d", s.port)
}

func (s *Server) broadcastLoop() {
	ticker := NewTicker(1)
	defer ticker.Stop()

	for range ticker.C {
		s.clientsMu.RLock()
		if len(s.clients) == 0 {
			s.clientsMu.RUnlock()
			continue
		}
		s.clientsMu.RUnlock()

		for _, tunnel := range s.config.Tunnels {
			status, ok := s.manager.GetStatus(tunnel.ID)
			if !ok {
				update := map[string]interface{}{
					"id":        tunnel.ID,
					"state":     "stopped",
					"publicUrl": "",
				}
				data, _ := MarshalJSON(update)
				s.broadcast(string(data))
				continue
			}

			update := map[string]interface{}{
				"id":           tunnel.ID,
				"state":        string(status.State),
				"publicUrl":    status.PublicURL,
				"errorMessage": status.ErrorMessage,
			}
			data, _ := MarshalJSON(update)
			s.broadcast(string(data))
		}
	}
}

func (s *Server) installProgressLoop() {
	for progress := range s.manager.DownloadProgress {
		update := map[string]interface{}{
			"type":    "install",
			"percent": progress.Percent,
			"current": progress.Current,
			"total":   progress.Total,
			"done":    progress.Done,
		}
		data, _ := MarshalJSON(update)
		s.broadcast(string(data))
	}
}

func (s *Server) broadcast(msg string) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for ch := range s.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (s *Server) addClient(ch chan string) {
	s.clientsMu.Lock()
	s.clients[ch] = true
	s.clientsMu.Unlock()
}

func (s *Server) removeClient(ch chan string) {
	s.clientsMu.Lock()
	delete(s.clients, ch)
	s.clientsMu.Unlock()
}

func (s *Server) BroadcastTunnelUpdate(t config.TunnelConfig) {
	state := "stopped"
	var publicURL, errorMessage string

	if status, ok := s.manager.GetStatus(t.ID); ok {
		state = string(status.State)
		publicURL = status.PublicURL
		errorMessage = status.ErrorMessage
	}

	update := map[string]interface{}{
		"id":           t.ID,
		"name":         t.Name,
		"provider":     string(t.Provider),
		"port":         t.LocalPort,
		"state":        state,
		"publicUrl":    publicURL,
		"errorMessage": errorMessage,
	}
	data, _ := MarshalJSON(update)
	s.broadcast(string(data))
}

func (s *Server) getClientChan() chan string {
	return make(chan string, 10)
}

func (s *Server) getTunnel(id string) *config.TunnelConfig {
	for i := range s.config.Tunnels {
		if s.config.Tunnels[i].ID == id {
			return &s.config.Tunnels[i]
		}
	}
	return nil
}

func (s *Server) updateConfig() {
	s.config.Save()
}

//go:embed static/*
var staticFiles embed.FS
