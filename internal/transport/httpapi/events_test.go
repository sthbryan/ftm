package httpapi

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
	s.hub.addWS(c)
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
// WebSocket through the bus, in the shapes the web already parses.
func TestChangesFromAnySurfaceReachTheWebSocket(t *testing.T) {
	s := newTestServer(t, process.NewManager(), config.DefaultConfig())
	s.listenEvents()
	client := attachClient(s)

	tunnel, err := s.svc.CreateTunnel(service.TunnelInput{Name: "World", Provider: config.ProviderBore, LocalPort: 30000})
	if err != nil {
		t.Fatalf("CreateTunnel: %v", err)
	}
	if msg := readMessage(t, client, "tunnel_state"); msg["id"] != tunnel.ID || msg["name"] != "World" {
		t.Fatalf("tunnel_state = %v, want %s", msg, tunnel.ID)
	}

	if err := s.svc.DeleteTunnel(tunnel.ID); err != nil {
		t.Fatalf("DeleteTunnel: %v", err)
	}
	if msg := readMessage(t, client, "tunnel_deleted"); msg["id"] != tunnel.ID {
		t.Fatalf("tunnel_deleted = %v", msg)
	}

	s.bus.Publish(events.InstallProgress{Provider: "bore", Percent: 50})
	msg := readMessage(t, client, "install")
	if msg["percent"] != 50.0 || msg["install"].(map[string]any)["step"] != "Installing..." {
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

	s.bus.Publish(events.ProviderUpdatesAvailable{Providers: []events.ProviderBinary{{ID: "cloudflared", Latest: "2026.11.0", HasUpdate: true}}})
	msg = readMessage(t, client, "provider_update_available")
	if ps := msg["providers"].([]any); len(ps) != 1 || ps[0].(map[string]any)["latest"] != "2026.11.0" {
		t.Fatalf("provider_update_available = %v", msg)
	}

	s.bus.Publish(events.ProviderUpdated{Provider: "cloudflared", OK: true, Version: "2026.11.0", Binary: events.ProviderBinary{ID: "cloudflared", Source: "bundled"}})
	msg = readMessage(t, client, "provider_updated")
	if msg["ok"] != true || msg["binary"].(map[string]any)["source"] != "bundled" {
		t.Fatalf("provider_updated = %v", msg)
	}
	if _, ok := msg["restarted"].([]any); !ok {
		t.Fatalf("provider_updated.restarted = %v, want an array", msg["restarted"])
	}
}

// A WebSocket log subscription follows the tunnel's output from the bus.
func TestWebSocketLogSubscriptionFollowsTheBus(t *testing.T) {
	s := newTestServer(t, nil, config.DefaultConfig())
	client := attachClient(s)

	if err := s.subscribeLogs(client, "world"); err != nil {
		t.Fatalf("subscribeLogs: %v", err)
	}
	// The bus subscription is live once subscribeLogs returns.
	s.bus.Publish(events.LogLine{TunnelID: "other", Line: "not mine"})
	s.bus.Publish(events.LogLine{TunnelID: "world", Line: "hello"})

	if msg := readMessage(t, client, "log"); msg["id"] != "world" || msg["line"] != "hello" {
		t.Fatalf("log = %v", msg)
	}
}
