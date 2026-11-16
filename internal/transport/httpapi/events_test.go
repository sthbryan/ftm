package httpapi

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/process"
	"github.com/sthbryan/ftm/internal/service"
)

func attachClient(t *testing.T, s *Server) *sseClient {
	t.Helper()
	c, _, _ := s.hub.subscribe(0, false)
	t.Cleanup(func() { s.hub.unsubscribe(c) })
	return c
}

func readMessage(t *testing.T, c *sseClient, wantType string) map[string]any {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case f := <-c.ch:
			var msg map[string]any
			if err := json.Unmarshal(f.Data, &msg); err != nil {
				t.Fatalf("bad event %s: %v", f.Data, err)
			}
			if msg["type"] != f.Event {
				t.Fatalf("event %q carries type %v", f.Event, msg["type"])
			}
			if f.Event == wantType {
				return msg
			}
		case <-deadline:
			t.Fatalf("no %q event broadcast", wantType)
		}
	}
}

// A change made outside the web (the TUI, the desktop shell) reaches the
// event stream through the bus, in the shapes the web parses.
func TestChangesFromAnySurfaceReachTheEventStream(t *testing.T) {
	s := newTestServer(t, process.NewManager(), config.DefaultConfig())
	s.listenEvents()
	client := attachClient(t, s)

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

// Following a tunnel's logs yields only that tunnel's lines from the bus.
func TestFollowLogsYieldsOnlyThatTunnel(t *testing.T) {
	s := newTestServer(t, nil, config.DefaultConfig())

	lines, stop := s.followLogs("world")
	defer stop()
	// The bus subscription is live once followLogs returns.
	s.bus.Publish(events.LogLine{TunnelID: "other", Line: "not mine"})
	s.bus.Publish(events.LogLine{TunnelID: "world", Line: "hello"})

	select {
	case line := <-lines:
		if line != "hello" {
			t.Fatalf("line = %q, want hello", line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no log line")
	}
}

// The WebSocket is gone: /ws/events is no API route.
func TestWebSocketRouteIsGone(t *testing.T) {
	s := newTestServer(t, nil, config.DefaultConfig())
	rec := call(t, s, "GET", "/ws/events", "", "Upgrade", "websocket", "Connection", "Upgrade")
	if rec.Code == 101 || rec.Code == 400 {
		t.Fatalf("/ws/events answered %d, want the dashboard fallback", rec.Code)
	}
}
