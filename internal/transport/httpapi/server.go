// Package httpapi serves ftm's HTTP API, its live events and the web
// dashboard. It is a thin transport: every action goes to internal/service,
// and every change reaches the clients from the event bus, whichever surface
// made it.
//
// REST routes are declared with huma, which also serves the OpenAPI 3.1
// document at /openapi.json. Live updates go out as Server-Sent Events on
// /api/events and /api/tunnels/{id}/logs?follow=1.
package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/platform/autostart"
	"github.com/sthbryan/ftm/internal/service"
)

const (
	listenHost = "127.0.0.1"
	portMin    = 40500
	portMax    = 40550

	shutdownTimeout = 5 * time.Second
)

// ErrNoPiP is returned when no floating panel opener is registered.
var ErrNoPiP = errors.New("floating panel is not available on this build")

// Server serves the API for one service.
type Server struct {
	svc       *service.Service
	bus       *events.Bus
	hub       *hub
	autostart autostart.Manager

	httpServer *http.Server
	handler    http.Handler
	port       int

	ctx    context.Context
	cancel context.CancelFunc

	mu        sync.RWMutex
	pipOpener func(tunnelID string) error
	apiToken  string
}

// NewServer builds a server. bus may be nil, in which case nothing is
// broadcast.
func NewServer(svc *service.Service, bus *events.Bus) *Server {
	s := &Server{
		svc:       svc,
		bus:       bus,
		hub:       newHub(),
		autostart: autostart.New(),
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.handler = s.routes()
	return s
}

// SetToken sets the runtime token that clients without a browser Origin must
// send. An empty token turns the check off.
func (s *Server) SetToken(token string) {
	s.mu.Lock()
	s.apiToken = token
	s.mu.Unlock()
}

func (s *Server) token() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.apiToken
}

// Handler is the full HTTP handler, guard included.
func (s *Server) Handler() http.Handler { return s.handler }

func (s *Server) listen() (net.Listener, error) {
	if preferred := s.svc.Config().WebPort; preferred > 0 {
		ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", listenHost, preferred))
		if err == nil {
			return ln, nil
		}
		log.Printf("httpapi: port %d unavailable (%v), scanning %d-%d", preferred, err, portMin, portMax)
	}

	for port := portMin; port <= portMax; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", listenHost, port))
		if err == nil {
			return ln, nil
		}
	}

	return nil, fmt.Errorf("no available port in range %d-%d", portMin, portMax)
}

// Start listens and serves in the background. It returns once the port is
// bound; nothing it does waits on the network beyond that.
func (s *Server) Start() error {
	ln, err := s.listen()
	if err != nil {
		return err
	}

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()
		return fmt.Errorf("unexpected listener address %T", ln.Addr())
	}

	s.port = addr.Port
	if err := s.svc.SetWebPort(s.port); err != nil {
		log.Printf("httpapi: could not persist the web port: %v", err)
	}

	s.httpServer = &http.Server{
		Handler:           s.handler,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return s.ctx },
	}

	s.listenEvents()

	go func() {
		if err := s.httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("httpapi: server stopped: %v", err)
		}
	}()

	return nil
}

// listenEvents turns bus events into broadcasts until the server stops.
func (s *Server) listenEvents() {
	if s.bus == nil {
		return
	}
	sub, cancel := s.bus.Subscribe()
	go func() {
		<-s.ctx.Done()
		cancel()
	}()
	go func() {
		for e := range sub {
			s.handleEvent(e)
		}
	}()
}

// Stop ends the live streams and shuts the server down, waiting a few
// seconds for requests in flight.
func (s *Server) Stop() error {
	s.cancel()
	if s.httpServer == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// Port is the port the server listens on.
func (s *Server) Port() int { return s.port }

// URL is the dashboard address.
func (s *Server) URL() string { return fmt.Sprintf("http://localhost:%d", s.port) }

// SetPiPOpener registers the desktop shell's floating panel.
func (s *Server) SetPiPOpener(fn func(tunnelID string) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pipOpener = fn
}

// HasPiP reports whether a floating panel opener is registered.
func (s *Server) HasPiP() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pipOpener != nil
}

// OpenPiP opens the floating panel for a tunnel.
func (s *Server) OpenPiP(tunnelID string) error {
	s.mu.RLock()
	opener := s.pipOpener
	s.mu.RUnlock()

	if opener == nil {
		return ErrNoPiP
	}
	return opener(tunnelID)
}

// ClientCount is how many dashboards are listening on /api/events.
func (s *Server) ClientCount() int { return s.hub.clients() }

func (s *Server) handleEvent(e events.Event) {
	switch e := e.(type) {
	case events.TunnelStatus:
		s.broadcastStatus(e.Status)
	case events.TunnelSaved:
		s.hub.publish(eventTunnelState, tunnelStateFromView(s.svc.View(e.Tunnel)))
	case events.TunnelDeleted:
		s.hub.publish(eventTunnelDeleted, tunnelDeletedMessage{Type: eventTunnelDeleted, ID: e.ID})
	case events.TunnelStats:
		if s.ClientCount() == 0 {
			return
		}
		s.hub.publish(eventTunnelStats, tunnelStatsMessage{
			Type:           eventTunnelStats,
			ID:             e.ID,
			Visitors:       e.Visitors,
			ActiveSessions: e.ActiveSessions,
		})
	case events.InstallProgress:
		s.hub.publish(eventInstall, installMessageFrom(e))
	case events.UpdateAvailable:
		s.hub.publish(eventUpdateAvailable, updateAvailableMessage{
			Type:       eventUpdateAvailable,
			Current:    e.Current,
			Latest:     e.Latest,
			Tag:        e.Tag,
			AssetName:  e.AssetName,
			ReleaseURL: e.ReleaseURL,
			Method:     e.Method,
		})
	case events.ProviderUpdatesAvailable:
		log.Printf("httpapi: provider updates available for %d provider(s)", len(e.Providers))
		s.hub.publish(eventProviderUpdateAvailable, providerUpdateAvailableFrom(e))
	case events.ProviderUpdated:
		s.hub.publish(eventProviderUpdated, providerUpdatedFrom(e))
	}
}

func (s *Server) broadcastStatus(status core.TunnelStatus) {
	if status.State == core.TunnelStateInstall {
		s.hub.publish(eventTunnelState, tunnelStatePatch{
			Type:     eventTunnelState,
			ID:       status.ID,
			State:    string(core.TunnelStateInstall),
			Provider: string(status.Provider),
		})
		s.notify("Installing", "Installing tunnel for "+string(status.Provider)+"...", "info")
		return
	}

	s.hub.publish(eventTunnelState, tunnelStateFromStatus(status))

	switch status.State {
	case core.TunnelStateOnline:
		if status.PublicURL != "" {
			s.notify("Tunnel Active", status.Name+" - "+status.PublicURL, "success")
		}
	case core.TunnelStateError:
		s.notify("Tunnel Error", status.Name+": "+status.ErrorMessage, "error")
	case core.TunnelStateTimeout:
		s.notify("Tunnel Timeout", status.Name+" could not connect", "error")
	case core.TunnelStateStopped:
		s.notify("Tunnel Stopped", status.Name+" has been stopped", "info")
	}
}

func (s *Server) notify(title, body, kind string) {
	s.notifyWithSound(title, body, kind, s.svc.Config().NotificationSound)
}

// BroadcastNewVisitor tells the dashboards a player joined a tunnel.
func (s *Server) BroadcastNewVisitor(status core.TunnelStatus) {
	s.notifyWithSound(
		i18n.T("notify_visitor_title"),
		i18n.TF("notify_visitor_body", status.Name, status.Visitors),
		"info",
		false,
	)
}

func (s *Server) notifyWithSound(title, body, kind string, sound bool) {
	channel := "toast"
	if s.svc.Config().NotificationsStatus == core.NotificationGranted {
		channel = "os"
	}

	s.hub.publish(eventNotification, notificationMessage{
		Type: eventNotification,
		Notification: notificationBody{
			Channel:      channel,
			Title:        title,
			Body:         body,
			ToastType:    kind,
			SoundType:    kind,
			SoundEnabled: sound,
		},
	})
}

// followLogs streams the lines a tunnel prints from now on, whether or not it
// is running yet. ch is nil when there is no event bus.
func (s *Server) followLogs(tunnelID string) (ch <-chan string, cancel func()) {
	if s.bus == nil {
		return nil, func() {}
	}

	sub, stop := s.bus.Subscribe(events.Only(events.KindLogLine))
	out := make(chan string, 100)
	go func() {
		defer close(out)
		for e := range sub {
			line, ok := e.(events.LogLine)
			if !ok || line.TunnelID != tunnelID {
				continue
			}
			select {
			case out <- line.Line:
			default:
				// A reader this far behind gets the snapshot on reconnect.
			}
		}
	}()
	return out, stop
}
