package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"foundry-tunnel/internal/clipboard"
	"foundry-tunnel/internal/config"
	"foundry-tunnel/internal/process"

	"github.com/gorilla/websocket"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	manager    *process.Manager
	config     *config.Config
	httpServer *http.Server
	port       int
	mu         sync.RWMutex
	clients    map[*Client]bool
	broadcast  chan []byte
}

type Client struct {
	server *Server
	conn   *websocket.Conn
	send   chan []byte
}

type TunnelResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	Port      int    `json:"port"`
	Running   bool   `json:"running"`
	Starting  bool   `json:"starting"`
	PublicURL string `json:"publicUrl"`
	Error     string `json:"error,omitempty"`
}

type CreateTunnelRequest struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Port     int    `json:"port"`
}

func NewServer(manager *process.Manager, cfg *config.Config) *Server {
	s := &Server{
		manager:   manager,
		config:    cfg,
		clients:   make(map[*Client]bool),
		broadcast: make(chan []byte, 100),
	}
	return s
}

func (s *Server) findPort() int {
	if s.config.WebPort > 0 {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.WebPort))
		if err == nil {
			ln.Close()
			return s.config.WebPort
		}
	}
	for port := 8080; port <= 8090; port++ {
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

	mux := http.NewServeMux()

	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	mux.HandleFunc("/api/tunnels", s.handleTunnels)
	mux.HandleFunc("/api/tunnels/", s.handleTunnelActions)
	mux.HandleFunc("/api/copy-url", s.handleCopyURL)
	mux.HandleFunc("/api/logs/", s.handleLogs)
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/api/status", s.handleStatus)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go s.broadcastLoop()
	go s.statusWatcher()

	go func() {
		s.httpServer.ListenAndServe()
	}()

	return nil
}

func (s *Server) Stop() error {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

func (s *Server) handleTunnels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listTunnels(w, r)
	case http.MethodPost:
		s.createTunnel(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) listTunnels(w http.ResponseWriter, r *http.Request) {
	tunnels := make([]TunnelResponse, 0, len(s.config.Tunnels))
	for _, t := range s.config.Tunnels {
		resp := TunnelResponse{
			ID:       t.ID,
			Name:     t.Name,
			Provider: string(t.Provider),
			Port:     t.LocalPort,
		}
		if status, ok := s.manager.GetStatus(t.ID); ok {
			resp.Running = status.Running
			resp.Starting = status.Starting
			resp.PublicURL = status.PublicURL
			resp.Error = status.Error
		}
		tunnels = append(tunnels, resp)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tunnels)
}

func (s *Server) createTunnel(w http.ResponseWriter, r *http.Request) {
	var req CreateTunnelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tunnel := config.TunnelConfig{
		ID:        fmt.Sprintf("tunnel-%d", time.Now().Unix()),
		Name:      req.Name,
		Provider:  config.Provider(req.Provider),
		LocalPort: req.Port,
		AutoStart: false,
	}

	s.config.Tunnels = append(s.config.Tunnels, tunnel)
	s.config.Save()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TunnelResponse{
		ID:       tunnel.ID,
		Name:     tunnel.Name,
		Provider: string(tunnel.Provider),
		Port:     tunnel.LocalPort,
	})
}

func (s *Server) handleTunnelActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tunnels/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	id := parts[0]

	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch r.Method {
	case http.MethodPost:
		switch action {
		case "start":
			s.startTunnel(w, r, id)
		case "stop":
			s.stopTunnel(w, r, id)
		default:
			http.Error(w, "unknown action", http.StatusBadRequest)
		}
	case http.MethodDelete:
		s.deleteTunnel(w, r, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) startTunnel(w http.ResponseWriter, r *http.Request, id string) {
	var tunnel *config.TunnelConfig
	for i := range s.config.Tunnels {
		if s.config.Tunnels[i].ID == id {
			tunnel = &s.config.Tunnels[i]
			break
		}
	}
	if tunnel == nil {
		http.Error(w, "tunnel not found", http.StatusNotFound)
		return
	}

	if needsInstall, canInstall := s.manager.CheckInstallation(tunnel.Provider); needsInstall && canInstall {
		go s.manager.InstallProvider(tunnel.Provider)
		http.Error(w, "installing provider", http.StatusAccepted)
		return
	}

	err := s.manager.Start(*tunnel, func(status config.TunnelStatus) {
		s.broadcastUpdate(id, status)
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func (s *Server) stopTunnel(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.manager.Stop(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func (s *Server) deleteTunnel(w http.ResponseWriter, r *http.Request, id string) {
	s.manager.Stop(id)

	for i, t := range s.config.Tunnels {
		if t.ID == id {
			s.config.Tunnels = append(s.config.Tunnels[:i], s.config.Tunnels[i+1:]...)
			break
		}
	}
	s.config.Save()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleCopyURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	clipboard.Write(req.URL)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "copied"})
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/logs/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	id := parts[0]

	logs := s.manager.GetLogs(id)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(strings.Join(logs, "\n")))
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"port":    s.port,
		"version": "0.2.0",
	})
}

func (s *Server) broadcastUpdate(tunnelID string, status config.TunnelStatus) {
	msg := map[string]interface{}{
		"type":     "tunnel_update",
		"tunnelId": tunnelID,
		"status": map[string]interface{}{
			"running":   status.Running,
			"starting":  status.Starting,
			"publicUrl": status.PublicURL,
			"error":     status.Error,
		},
	}
	data, _ := json.Marshal(msg)
	s.broadcast <- data
}

func (s *Server) broadcastLoop() {
	for msg := range s.broadcast {
		s.mu.RLock()
		for client := range s.clients {
			select {
			case client.send <- msg:
			default:
				close(client.send)
				delete(s.clients, client)
			}
		}
		s.mu.RUnlock()
	}
}

func (s *Server) statusWatcher() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		for _, t := range s.config.Tunnels {
			if status, ok := s.manager.GetStatus(t.ID); ok {
				s.broadcastUpdate(t.ID, status)
			}
		}
	}
}
