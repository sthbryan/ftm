package web

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/process"
	"github.com/sthbryan/ftm/internal/store"
)

func clearHandlers(t *testing.T, cfg *config.Config) *Handlers {
	t.Helper()

	root := t.TempDir()
	dirs := store.Dirs{Config: filepath.Join(root, "config"), Legacy: filepath.Join(root, "legacy")}

	return NewServer(process.NewManager(), store.New(dirs, cfg)).handlers
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
	cfg := config.DefaultConfig()
	cfg.Language = "es"
	cfg.AddTunnel(config.TunnelConfig{ID: "a", Name: "First", LocalPort: 30000})
	cfg.AddTunnel(config.TunnelConfig{ID: "b", Name: "Second", LocalPort: 30001})

	h := clearHandlers(t, cfg)

	w := httptest.NewRecorder()
	h.clearTunnels(w)

	if got := decodeDeleted(t, w.Body.Bytes()); got != 2 {
		t.Errorf("deleted = %d, want 2", got)
	}
	got := h.store.Snapshot()
	if len(got.Tunnels) != 0 {
		t.Fatalf("Tunnels = %+v, want empty", got.Tunnels)
	}
	if !got.Onboarded {
		t.Error("Onboarded = false, want the empty home and not the welcome screen")
	}
	if got.Language != "es" {
		t.Errorf("Language = %q, want it untouched", got.Language)
	}

	saved, err := store.Peek(h.store.Dirs())
	if err != nil {
		t.Fatalf("Peek() failed: %v", err)
	}
	if len(saved.Tunnels) != 0 {
		t.Errorf("persisted Tunnels = %+v, want empty", saved.Tunnels)
	}
}

func TestClearTunnelsOnAnEmptyConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	h := clearHandlers(t, cfg)

	w := httptest.NewRecorder()
	h.clearTunnels(w)

	if got := decodeDeleted(t, w.Body.Bytes()); got != 0 {
		t.Errorf("deleted = %d, want 0", got)
	}
}

func TestListTunnelsAfterClearingReturnsAnArray(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.AddTunnel(config.TunnelConfig{ID: "a", Name: "First", LocalPort: 30000})

	h := clearHandlers(t, cfg)
	h.clearTunnels(httptest.NewRecorder())

	w := httptest.NewRecorder()
	h.listTunnels(w)

	if got := w.Body.String(); got != "[]\n" {
		t.Fatalf("body = %q, want an empty JSON array and not null", got)
	}
}
