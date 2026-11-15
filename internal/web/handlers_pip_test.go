package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/store"
)

func newPiPTestServer() *Server {
	cfg := &config.Config{
		Tunnels: []config.TunnelConfig{
			{ID: "test", Name: "cloudflared", Provider: config.ProviderCloudflared, LocalPort: 3000},
		},
	}

	st := store.New(store.Dirs{}, cfg)
	s := &Server{store: st}
	s.handlers = NewHandlers(nil, st, s)
	return s
}

func postPiP(s *Server, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/pip", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.handlers.Route(rec, req)
	return rec
}

func TestHandlePiPWithoutOpener(t *testing.T) {
	s := newPiPTestServer()

	rec := postPiP(s, `{"id":"test"}`)

	if rec.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotImplemented)
	}
}

func TestHandlePiPWithOpener(t *testing.T) {
	s := newPiPTestServer()

	var got string
	s.SetPiPOpener(func(tunnelID string) error {
		got = tunnelID
		return nil
	})

	rec := postPiP(s, `{"id":"test"}`)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got != "test" {
		t.Errorf("opener received %q, want %q", got, "test")
	}
}

func TestHandlePiPUnknownTunnel(t *testing.T) {
	s := newPiPTestServer()
	s.SetPiPOpener(func(string) error {
		t.Error("opener called for an unknown tunnel")
		return nil
	})

	rec := postPiP(s, `{"id":"nope"}`)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandlePiPMissingID(t *testing.T) {
	s := newPiPTestServer()

	rec := postPiP(s, `{}`)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlePiPRejectsGet(t *testing.T) {
	s := newPiPTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/pip", nil)
	rec := httptest.NewRecorder()
	s.handlers.Route(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandlePiPOpenerError(t *testing.T) {
	s := newPiPTestServer()
	s.SetPiPOpener(func(string) error {
		return errors.New("boom")
	})

	rec := postPiP(s, `{"id":"test"}`)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestStatusReportsNativePiP(t *testing.T) {
	s := newPiPTestServer()

	if nativePiPFromStatus(t, s) {
		t.Error("nativePip = true before an opener is registered")
	}

	s.SetPiPOpener(func(string) error { return nil })

	if !nativePiPFromStatus(t, s) {
		t.Error("nativePip = false after an opener is registered")
	}
}

func nativePiPFromStatus(t *testing.T, s *Server) bool {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	s.handlers.Route(rec, req)

	var payload struct {
		NativePiP bool `json:"nativePip"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decoding status: %v", err)
	}
	return payload.NativePiP
}
