package web

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/process"
)

const (
	listenHost = "127.0.0.1"
	portMin    = 40500
	portMax    = 40550
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	manager       *process.Manager
	config        *config.Config
	httpServer    *http.Server
	port          int
	clients       map[*wsClient]struct{}
	clientsMu     sync.RWMutex
	handlers      *Handlers
	StatusChannel chan config.TunnelStatus
	updateSvc     *updateService
	providerSvc   *providerUpdateService
	updateCtx     context.Context
	updateCancel  context.CancelFunc
	pipOpener     func(tunnelID string) error
	pipMu         sync.RWMutex
}

func NewServer(manager *process.Manager, cfg *config.Config) *Server {
	s := &Server{
		manager:       manager,
		config:        cfg,
		clients:       make(map[*wsClient]struct{}),
		StatusChannel: make(chan config.TunnelStatus, 64),
	}
	s.handlers = NewHandlers(manager, cfg, s)
	s.updateCtx, s.updateCancel = context.WithCancel(context.Background())
	s.updateSvc = newUpdateService(s.broadcast)
	s.providerSvc = newProviderUpdateService(manager, s.broadcastJSON)
	return s
}

func (s *Server) listen() (net.Listener, error) {
	if s.config.WebPort > 0 {
		ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", listenHost, s.config.WebPort))
		if err == nil {
			return ln, nil
		}
		log.Printf("web: port %d unavailable (%v), scanning %d-%d", s.config.WebPort, err, portMin, portMax)
	}

	for port := portMin; port <= portMax; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", listenHost, port))
		if err == nil {
			return ln, nil
		}
	}

	return nil, fmt.Errorf("no available port in range %d-%d", portMin, portMax)
}

func (s *Server) Start() error {
	ln, err := s.listen()
	if err != nil {
		return err
	}

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		ln.Close()
		return fmt.Errorf("unexpected listener address %T", ln.Addr())
	}

	s.port = addr.Port
	s.config.WebPort = s.port
	if err := s.config.Save(); err != nil {
		log.Printf("web: could not persist the web port: %v", err)
	}

	s.httpServer = &http.Server{Handler: s.setupRoutes()}

	s.updateSvc.Start(s.updateCtx)
	s.providerSvc.Start(s.updateCtx)
	go s.installProgressLoop()
	go s.statusUpdateLoop()
	go s.statsLoop()
	go func() {
		if err := s.httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("web: server stopped: %v", err)
		}
	}()

	return nil
}

func (s *Server) setupRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/", s.handlers.Route)

	mux.HandleFunc("/ws/events", s.handleWebSocket)

	webDist := filepath.Join("web-svelte", "dist")
	var staticFS fs.FS
	if _, err := os.Stat(webDist); err == nil {
		staticFS, _ = fs.Sub(os.DirFS(webDist), ".")
	} else {
		staticFS, _ = fs.Sub(staticFiles, "static")
	}
	fileServer := http.FileServer(http.FS(staticFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws/") {
			return
		}
		path := r.URL.Path
		if path != "/" && !strings.Contains(path, ".") {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})

	return guardHost(mux)
}

func (s *Server) Stop() error {
	if s.updateCancel != nil {
		s.updateCancel()
	}
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

func (s *Server) SetPiPOpener(fn func(tunnelID string) error) {
	s.pipMu.Lock()
	defer s.pipMu.Unlock()
	s.pipOpener = fn
}

func (s *Server) HasPiP() bool {
	s.pipMu.RLock()
	defer s.pipMu.RUnlock()
	return s.pipOpener != nil
}

func (s *Server) OpenPiP(tunnelID string) error {
	s.pipMu.RLock()
	opener := s.pipOpener
	s.pipMu.RUnlock()

	if opener == nil {
		return ErrNoPiP
	}
	return opener(tunnelID)
}

func (s *Server) ClientCount() int {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	return len(s.clients)
}

func (s *Server) URL() string {
	return fmt.Sprintf("http://localhost:%d", s.port)
}

func (s *Server) installProgressLoop() {
	for progress := range s.manager.ProgressEvents() {
		step := "Installing..."
		if progress.Done {
			step = "Done"
		}

		s.broadcastJSON(map[string]interface{}{
			"type": "install",
			"install": map[string]interface{}{
				"provider": progress.Name,
				"percent":  progress.Percent,
				"step":     step,
			},
			"percent": progress.Percent,
			"current": progress.Current,
			"total":   progress.Total,
			"done":    progress.Done,
		})
	}
}

func (s *Server) statusUpdateLoop() {
	for status := range s.StatusChannel {
		s.broadcastJSON(tunnelStateFromStatus(status))
		s.broadcastTunnelNotification(status)
	}
}

const statsInterval = 5 * time.Second

func (s *Server) statsLoop() {
	ticker := time.NewTicker(statsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.updateCtx.Done():
			return
		case <-ticker.C:
			s.broadcastStats()
		}
	}
}

func (s *Server) broadcastStats() {
	if s.ClientCount() == 0 {
		return
	}

	for _, t := range s.config.Tunnels {
		status, ok := s.manager.GetStatus(t.ID)
		if !ok || status.State != config.TunnelStateOnline {
			continue
		}

		s.broadcastJSON(map[string]interface{}{
			"type":           "tunnel_stats",
			"id":             t.ID,
			"visitors":       status.Visitors,
			"activeSessions": status.ActiveSessions,
		})
	}
}

func (s *Server) BroadcastTunnelUpdate(t config.TunnelConfig) {
	s.broadcastJSON(tunnelStateFor(s.manager, t))
}

func (s *Server) BroadcastTunnelDeleted(id string) {
	s.broadcastJSON(map[string]interface{}{
		"type": "tunnel_deleted",
		"id":   id,
	})
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
	if err := s.config.Save(); err != nil {
		log.Printf("web: could not save the config: %v", err)
	}
}

func (s *Server) broadcastTunnelNotification(status config.TunnelStatus) {
	switch status.State {
	case config.TunnelStateOnline:
		if status.PublicURL == "" {
			return
		}
		s.broadcastNotification("tunnel_online", "Tunnel Active", status.Name+" - "+status.PublicURL, "success", "success")
	case config.TunnelStateError:
		s.broadcastNotification("tunnel_error", "Tunnel Error", status.Name+": "+status.ErrorMessage, "error", "error")
	case config.TunnelStateTimeout:
		s.broadcastNotification("tunnel_timeout", "Tunnel Timeout", status.Name+" could not connect", "error", "error")
	case config.TunnelStateStopped:
		s.broadcastNotification("tunnel_stopped", "Tunnel Stopped", status.Name+" has been stopped", "info", "info")
	}
}

func (s *Server) broadcastInstallingNotification(tunnel config.TunnelConfig) {
	s.broadcastNotification("tunnel_installing", "Installing", "Installing tunnel for "+string(tunnel.Provider)+"...", "info", "info")
}

func (s *Server) broadcastNotification(_ string, title, body, toastType, soundType string) {
	s.broadcastNotificationWithSound(title, body, toastType, soundType, s.config.NotificationSound)
}

func (s *Server) BroadcastNewVisitor(status config.TunnelStatus) {
	s.broadcastNotificationWithSound(
		i18n.T("notify_visitor_title"),
		i18n.TF("notify_visitor_body", status.Name, status.Visitors),
		"info",
		"info",
		false,
	)
}

func (s *Server) broadcastNotificationWithSound(title, body, toastType, soundType string, sound bool) {
	channel := "toast"
	if s.config.NotificationsStatus == config.NotificationGranted {
		channel = "os"
	}

	s.broadcastJSON(map[string]interface{}{
		"type": "notification",
		"notification": map[string]interface{}{
			"channel":      channel,
			"title":        title,
			"body":         body,
			"toastType":    toastType,
			"soundType":    soundType,
			"soundEnabled": sound,
		},
	})
}
