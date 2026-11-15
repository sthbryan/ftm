package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/process"
)

func TestProviderUpdateRejectsWhatFTMDoesNotManage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))

	manager := process.NewManager()
	srv := NewServer(manager, &config.Config{})

	tests := []struct {
		method, path string
		want         int
	}{
		{http.MethodPost, "/api/providers/serveo/update", http.StatusBadRequest},
		{http.MethodPost, "/api/providers/localhostrun/update", http.StatusBadRequest},
		{http.MethodPost, "/api/providers/nope/update", http.StatusNotFound},
		{http.MethodGet, "/api/providers/cloudflared/update", http.StatusMethodNotAllowed},
		{http.MethodPost, "/api/providers/cloudflared/delete", http.StatusNotFound},
		{http.MethodPost, "/api/providers/installed", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Host = "localhost"
			w := httptest.NewRecorder()
			srv.handlers.Route(w, req)
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d (%s)", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

func TestProviderUpdateNoticeIsSentOncePerVersion(t *testing.T) {
	var sent []providerUpdateAvailableMessage
	svc := newProviderUpdateService(nil, func(v any) {
		sent = append(sent, v.(providerUpdateAvailableMessage))
	})

	report := []process.ProviderBinary{
		{ID: config.ProviderCloudflared, Version: "2024.6.1", Latest: "2026.11.0", HasUpdate: true, Updatable: true},
		{ID: config.ProviderBore, Version: "0.6.0", Latest: "0.6.0"},
	}

	svc.announce(report)
	svc.announce(report)
	if len(sent) != 1 || len(sent[0].Providers) != 1 || sent[0].Providers[0].ID != "cloudflared" {
		t.Fatalf("sent = %+v, want one notice for cloudflared", sent)
	}

	report[0].Latest = "2026.12.0"
	svc.announce(report)
	if len(sent) != 2 {
		t.Fatalf("a newer release was not announced: %+v", sent)
	}
}
