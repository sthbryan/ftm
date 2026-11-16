package httpapi

import (
	"net/http"
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
	srv := newTestServer(t, manager, &config.Config{})

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
			w := call(t, srv, tt.method, tt.path, "")
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d (%s)", w.Code, tt.want, w.Body.String())
			}
		})
	}
}
