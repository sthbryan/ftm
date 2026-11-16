package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/store"
	"github.com/sthbryan/ftm/internal/supervisor"
)

func clearServer(t *testing.T, cfg *core.Config) *Server {
	t.Helper()

	return newTestServer(t, supervisor.NewManager(), cfg)
}

func decodeDeleted(t *testing.T, body []byte) int {
	t.Helper()

	var payload struct {
		Deleted int `json:"deleted"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("response does not decode: %v (%s)", err, body)
	}
	return payload.Deleted
}

func TestClearTunnelsRemovesEveryConnection(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.Language = "es"
	cfg.AddTunnel(core.TunnelConfig{ID: "a", Name: "First", LocalPort: 30000})
	cfg.AddTunnel(core.TunnelConfig{ID: "b", Name: "Second", LocalPort: 30001})

	s := clearServer(t, cfg)

	w := call(t, s, http.MethodDelete, "/api/tunnels", "")

	if got := decodeDeleted(t, w.Body.Bytes()); got != 2 {
		t.Errorf("deleted = %d, want 2", got)
	}
	got := s.svc.Config()
	if len(got.Tunnels) != 0 {
		t.Fatalf("Tunnels = %+v, want empty", got.Tunnels)
	}
	if !got.Onboarded {
		t.Error("Onboarded = false, want the empty home and not the welcome screen")
	}
	if got.Language != "es" {
		t.Errorf("Language = %q, want it untouched", got.Language)
	}

	saved, err := store.Peek(s.svc.Dirs())
	if err != nil {
		t.Fatalf("Peek() failed: %v", err)
	}
	if len(saved.Tunnels) != 0 {
		t.Errorf("persisted Tunnels = %+v, want empty", saved.Tunnels)
	}
}

func TestClearTunnelsOnAnEmptyConfig(t *testing.T) {
	cfg := core.DefaultConfig()
	s := clearServer(t, cfg)

	w := call(t, s, http.MethodDelete, "/api/tunnels", "")

	if got := decodeDeleted(t, w.Body.Bytes()); got != 0 {
		t.Errorf("deleted = %d, want 0", got)
	}
}

func TestListTunnelsAfterClearingReturnsAnArray(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.AddTunnel(core.TunnelConfig{ID: "a", Name: "First", LocalPort: 30000})

	s := clearServer(t, cfg)
	call(t, s, http.MethodDelete, "/api/tunnels", "")

	w := call(t, s, http.MethodGet, "/api/tunnels", "")

	if got := w.Body.String(); got != "[]\n" {
		t.Fatalf("body = %q, want an empty JSON array and not null", got)
	}
}
