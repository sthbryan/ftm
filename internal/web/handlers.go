package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sthbryan/ftm/internal/autostart"
	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/process"
	"github.com/sthbryan/ftm/internal/store"
)

type Handlers struct {
	manager   *process.Manager
	store     *store.Store
	server    *Server
	autostart autostart.Manager
}

func NewHandlers(manager *process.Manager, st *store.Store, server *Server) *Handlers {
	return &Handlers{
		manager:   manager,
		store:     st,
		server:    server,
		autostart: autostart.New(),
	}
}

func (h *Handlers) Route(w http.ResponseWriter, r *http.Request) {
	if !h.setCORS(w, r) {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return
	}

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == "" {
		path = "/"
	}

	switch {
	case path == "/api/tunnels":
		h.handleTunnels(w, r)
	case strings.HasPrefix(path, "/api/logs/"):
		h.handleLogs(w, r)
	case path == "/api/status":
		h.handleStatus(w)
	case path == "/api/settings":
		h.handleSettings(w, r)
	case path == "/api/providers":
		h.handleProviders(w)
	case path == "/api/providers/installed":
		h.handleInstalledProviders(w, r)
	case strings.HasPrefix(path, "/api/providers/"):
		h.handleProviderActions(w, r, path)
	case path == "/api/i18n":
		h.handleI18n(w, r)
	case path == "/api/i18n/current":
		h.handleI18nCurrent(w)
	case path == "/api/update":
		h.handleUpdate(w, r)
	case path == "/api/detect-port":
		h.handleDetectPort(w)
	case path == "/api/pip":
		h.handlePiP(w, r)
	case strings.HasPrefix(path, "/api/tunnels/"):
		h.handleTunnelActions(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handlers) setCORS(w http.ResponseWriter, r *http.Request) bool {
	origin, ok := allowedOrigin(r)
	if !ok {
		return false
	}

	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Add("Vary", "Origin")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	return true
}

func (h *Handlers) writeTunnelJSON(w http.ResponseWriter, t config.TunnelConfig) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tunnelStateFor(h.manager, t))
}
