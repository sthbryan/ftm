package web

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/process"
	"github.com/sthbryan/ftm/internal/service"
)

func attachClient(s *Server) *wsClient {
	c := &wsClient{
		send:             make(chan []byte, 64),
		done:             make(chan struct{}),
		logSubscriptions: map[string]context.CancelFunc{},
	}
	s.clientsMu.Lock()
	s.clients[c] = struct{}{}
	s.clientsMu.Unlock()
	return c
}

func readMessage(t *testing.T, c *wsClient, wantType string) map[string]any {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case raw := <-c.send:
			var msg map[string]any
			if err := json.Unmarshal(raw, &msg); err != nil {
				t.Fatalf("bad message %s: %v", raw, err)
			}
			if msg["type"] == wantType {
				return msg
			}
		case <-deadline:
			t.Fatalf("no %q message broadcast", wantType)
		}
	}
}

// A change made outside the web (the TUI, the desktop shell) reaches the
// browser through the bus.
func TestChangesFromAnySurfaceAreBroadcast(t *testing.T) {
	s := newTestServer(t, process.NewManager(), config.DefaultConfig())
	sub, cancel := s.bus.Subscribe()
	defer cancel()
	go s.eventLoop(sub)
	client := attachClient(s)

	tunnel, err := s.svc.CreateTunnel(service.TunnelInput{Name: "World", Provider: config.ProviderBore, LocalPort: 30000})
	if err != nil {
		t.Fatalf("CreateTunnel: %v", err)
	}
	if msg := readMessage(t, client, "tunnel_state"); msg["id"] != tunnel.ID {
		t.Fatalf("tunnel_state = %v, want %s", msg, tunnel.ID)
	}

	if err := s.svc.DeleteTunnel(tunnel.ID); err != nil {
		t.Fatalf("DeleteTunnel: %v", err)
	}
	if msg := readMessage(t, client, "tunnel_deleted"); msg["id"] != tunnel.ID {
		t.Fatalf("tunnel_deleted = %v", msg)
	}

	s.bus.Publish(events.InstallProgress{Provider: "bore", Percent: 50})
	if msg := readMessage(t, client, "install"); msg["percent"] != 50.0 {
		t.Fatalf("install = %v", msg)
	}

	s.bus.Publish(events.TunnelStats{ID: tunnel.ID, Visitors: 3, ActiveSessions: 1})
	if msg := readMessage(t, client, "tunnel_stats"); msg["visitors"] != 3.0 {
		t.Fatalf("tunnel_stats = %v", msg)
	}

	s.bus.Publish(events.UpdateAvailable{Latest: "9.9.9"})
	if msg := readMessage(t, client, "update_available"); msg["latest"] != "9.9.9" {
		t.Fatalf("update_available = %v", msg)
	}
}
