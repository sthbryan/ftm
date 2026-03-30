package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/process"
)

type Handlers struct {
	manager *process.Manager
	config  *config.Config
	server  *Server
}

func NewHandlers(manager *process.Manager, cfg *config.Config, server *Server) *Handlers {
	return &Handlers{
		manager: manager,
		config:  cfg,
		server:  server,
	}
}

func (h *Handlers) Route(w http.ResponseWriter, r *http.Request) {
	h.setCORS(w)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch {
	case r.URL.Path == "/api/tunnels":
		h.handleTunnels(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/logs/"):
		h.handleLogs(w, r)
	case r.URL.Path == "/api/status":
		h.handleStatus(w)
	case r.URL.Path == "/api/notifications":
		h.handleNotifications(w, r)
	case r.URL.Path == "/api/settings":
		h.handleSettings(w, r)
	case r.URL.Path == "/api/providers":
		h.handleProviders(w)
	case r.URL.Path == "/api/detect-port":
		h.handleDetectPort(w)
	case strings.HasPrefix(r.URL.Path, "/api/tunnels/"):
		h.handleTunnelActions(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handlers) setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func (h *Handlers) tunnelToMap(t config.TunnelConfig) map[string]interface{} {
	item := map[string]interface{}{
		"id":       t.ID,
		"name":     t.Name,
		"provider": string(t.Provider),
		"port":     t.LocalPort,
		"state":    "stopped",
	}

	if status, ok := h.manager.GetStatus(t.ID); ok {
		item["publicUrl"] = status.PublicURL
		item["errorMessage"] = status.ErrorMessage
		item["state"] = string(status.State)
	}


	if item["state"] == "stopped" {
		if needsInstall, canInstall := h.manager.CheckInstallation(t.Provider); needsInstall && canInstall {
			item["state"] = "need_installing"
		}
	}

	return item
}

func (h *Handlers) writeTunnelJSON(w http.ResponseWriter, t config.TunnelConfig) {
	state := "stopped"
	var publicURL, errorMessage string

	if tunnelStatus, ok := h.manager.GetStatus(t.ID); ok {
		publicURL = tunnelStatus.PublicURL
		errorMessage = tunnelStatus.ErrorMessage
		state = string(tunnelStatus.State)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           t.ID,
		"name":         t.Name,
		"provider":     string(t.Provider),
		"port":         t.LocalPort,
		"state":        state,
		"publicUrl":    publicURL,
		"errorMessage": errorMessage,
	})
}

func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
