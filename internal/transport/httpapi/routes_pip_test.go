package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sthbryan/ftm/internal/core"
)

func newPiPTestServer(t *testing.T) *Server {
	cfg := &core.Config{
		Tunnels: []core.TunnelConfig{
			{ID: "test", Name: "cloudflared", Provider: core.ProviderCloudflared, LocalPort: 3000},
		},
	}

	return newTestServer(t, nil, cfg)
}

func postPiP(t *testing.T, s *Server, body string) *httptest.ResponseRecorder {
	return call(t, s, http.MethodPost, "/api/pip", body)
}

func TestHandlePiPWithoutOpener(t *testing.T) {
	s := newPiPTestServer(t)

	rec := postPiP(t, s, `{"id":"test"}`)

	if rec.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotImplemented)
	}
}

func TestHandlePiPWithOpener(t *testing.T) {
	s := newPiPTestServer(t)

	var got string
	s.SetPiPOpener(func(tunnelID string) error {
		got = tunnelID
		return nil
	})

	rec := postPiP(t, s, `{"id":"test"}`)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got != "test" {
		t.Errorf("opener received %q, want %q", got, "test")
	}
}

func TestHandlePiPUnknownTunnel(t *testing.T) {
	s := newPiPTestServer(t)
	s.SetPiPOpener(func(string) error {
		t.Error("opener called for an unknown tunnel")
		return nil
	})

	rec := postPiP(t, s, `{"id":"nope"}`)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandlePiPMissingID(t *testing.T) {
	s := newPiPTestServer(t)

	rec := postPiP(t, s, `{}`)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlePiPRejectsGet(t *testing.T) {
	s := newPiPTestServer(t)

	rec := call(t, s, http.MethodGet, "/api/pip", "")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandlePiPOpenerError(t *testing.T) {
	s := newPiPTestServer(t)
	s.SetPiPOpener(func(string) error {
		return errors.New("boom")
	})

	rec := postPiP(t, s, `{"id":"test"}`)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestStatusReportsNativePiP(t *testing.T) {
	s := newPiPTestServer(t)

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

	rec := call(t, s, http.MethodGet, "/api/status", "")

	var payload struct {
		NativePiP bool `json:"nativePip"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decoding status: %v", err)
	}
	return payload.NativePiP
}
