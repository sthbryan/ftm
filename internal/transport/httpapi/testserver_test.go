package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/process"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/store"
)

// newTestServer wires a server the way the app does, with the config saved
// under a temp dir. sup may be nil for routes that never start anything.
func newTestServer(t *testing.T, sup service.Supervisor, cfg *config.Config) *Server {
	t.Helper()

	root := t.TempDir()
	st := store.New(store.Dirs{
		Config: filepath.Join(root, "config"),
		Legacy: filepath.Join(root, "legacy"),
	}, cfg)

	bus := events.New()
	t.Cleanup(bus.Close)

	if sup == nil {
		sup = nopSupervisor{}
	}
	if m, ok := sup.(*process.Manager); ok {
		m.SetEvents(bus)
	}
	s := NewServer(service.New(st, sup, bus), bus)
	t.Cleanup(func() { _ = s.Stop() })
	return s
}

// call sends one request through the full handler, guard included, the way
// the dashboard page does.
func call(t *testing.T, s *Server, method, path, body string, headers ...string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Host = "127.0.0.1:40500"
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(headers); i += 2 {
		if headers[i+1] == "" {
			req.Header.Del(headers[i])
			continue
		}
		req.Header.Set(headers[i], headers[i+1])
	}

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d (%s)", rec.Code, want, rec.Body.String())
	}
}

type nopSupervisor struct{}

func (nopSupervisor) Start(config.TunnelConfig) error                { return nil }
func (nopSupervisor) Stop(string) error                              { return nil }
func (nopSupervisor) StopAll()                                       {}
func (nopSupervisor) CheckInstallation(config.Provider) (bool, bool) { return false, false }
func (nopSupervisor) InstallProvider(config.Provider) error          { return nil }

var _ = http.MethodGet
