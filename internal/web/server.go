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

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/process"
	"github.com/sthbryan/ftm/internal/service"
)

const (
	listenHost = "127.0.0.1"
	portMin    = 40500
	portMax    = 40550
)

//go:embed static/*
var staticFiles embed.FS

// Server is a thin transport: actions go to the service, and every change
// reaches the browser from the event bus, whichever surface made it.
type Server struct {
	manager      *process.Manager
	svc          *service.Service
	bus          *events.Bus
	httpServer   *http.Server
	port         int
	clients      map[*wsClient]struct{}
	clientsMu    sync.RWMutex
	handlers     *Handlers
	providerSvc  *providerUpdateService
	updateCtx    context.Context
	updateCancel context.CancelFunc
	pipOpener    func(tunnelID string) error
	pipMu        sync.RWMutex
}

func NewServer(manager *process.Manager, svc *service.Service, bus *events.Bus) *Server {
	s := &Server{
		manager: manager,
		svc:     svc,
		bus:     bus,
		clients: make(map[*wsClient]struct{}),
	}
	s.handlers = NewHandlers(manager, svc, s)
	s.updateCtx, s.updateCancel = context.WithCancel(context.Background())
	s.providerSvc = newProviderUpdateService(manager, s.broadcastJSON)
	return s
}

func (s *Server) listen() (net.Listener, error) {
	if preferred := s.svc.Config().WebPort; preferred > 0 {
		ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", listenHost, preferred))
		if err == nil {
			return ln, nil
		}
		log.Printf("web: port %d unavailable (%v), scanning %d-%d", preferred, err, portMin, portMax)
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
	if err := s.svc.SetWebPort(s.port); err != nil {
		log.Printf("web: could not persist the web port: %v", err)
	}

	s.httpServer = &http.Server{Handler: s.setupRoutes()}

	if s.bus != nil {
		sub, cancel := s.bus.Subscribe()
		go func() {
			<-s.updateCtx.Done()
			cancel()
		}()
		go s.eventLoop(sub)
	}
	s.providerSvc.Start(s.updateCtx)
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

	webDist := filepath.Join("web", "dist")
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

// eventLoop turns bus events into WebSocket messages until the subscription
// is cancelled.
func (s *Server) eventLoop(sub <-chan events.Event) {
	for e := range sub {
		s.handleEvent(e)
	}
}

func (s *Server) handleEvent(e events.Event) {
	switch e := e.(type) {
	case events.TunnelStatus:
		s.broadcastStatus(e.Status)
	case events.TunnelSaved:
		s.BroadcastTunnelUpdate(e.Tunnel)
	case events.TunnelDeleted:
		s.BroadcastTunnelDeleted(e.ID)
	case events.TunnelStats:
		if s.ClientCount() == 0 {
			return
		}
		s.broadcastJSON(map[string]interface{}{
			"type":           "tunnel_stats",
			"id":             e.ID,
			"visitors":       e.Visitors,
			"activeSessions": e.ActiveSessions,
		})
	case events.InstallProgress:
		s.broadcastInstall(e)
	case events.UpdateAvailable:
		s.broadcastJSON(map[string]interface{}{
			"type":       "update_available",
			"current":    e.Current,
			"latest":     e.Latest,
			"tag":        e.Tag,
			"assetName":  e.AssetName,
			"releaseUrl": e.ReleaseURL,
			"method":     e.Method,
		})
	}
}

func (s *Server) broadcastStatus(status config.TunnelStatus) {
	if status.State == config.TunnelStateInstall {
		s.broadcastJSON(tunnelStatePatch{
			Type:     "tunnel_state",
			ID:       status.ID,
			State:    string(config.TunnelStateInstall),
			Provider: string(status.Provider),
		})
		s.broadcastNotification("tunnel_installing", "Installing", "Installing tunnel for "+string(status.Provider)+"...", "info", "info")
		return
	}

	s.broadcastJSON(tunnelStateFromStatus(status))
	s.broadcastTunnelNotification(status)
}

func (s *Server) broadcastInstall(p events.InstallProgress) {
	if p.Err != "" {
		// The failure reaches the tunnel as an error state; the progress bar
		// only needs to know the download ended.
		p.Done = true
	}

	step := "Installing..."
	if p.Done {
		step = "Done"
	}

	s.broadcastJSON(map[string]interface{}{
		"type": "install",
		"install": map[string]interface{}{
			"provider": p.Provider,
			"percent":  p.Percent,
			"step":     step,
		},
		"percent": p.Percent,
		"current": p.Current,
		"total":   p.Total,
		"done":    p.Done,
	})
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

func (s *Server) getTunnel(id string) (config.TunnelConfig, bool) {
	return s.svc.Tunnel(id)
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

func (s *Server) broadcastNotification(_ string, title, body, toastType, soundType string) {
	s.broadcastNotificationWithSound(title, body, toastType, soundType, s.svc.Config().NotificationSound)
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
	if s.svc.Config().NotificationsStatus == config.NotificationGranted {
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
