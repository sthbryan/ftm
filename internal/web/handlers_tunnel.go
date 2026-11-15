package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/sthbryan/ftm/internal/config"
)

func (h *Handlers) handleTunnels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listTunnels(w)
	case http.MethodPost:
		h.createTunnel(w, r)
	case http.MethodDelete:
		h.clearTunnels(w)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) clearTunnels(w http.ResponseWriter) {
	ids := make([]string, 0, len(h.config.Tunnels))
	for _, t := range h.config.Tunnels {
		ids = append(ids, t.ID)
	}

	for _, id := range ids {
		h.manager.Stop(id)
	}

	deleted := h.config.ClearTunnels()
	h.server.updateConfig()

	for _, id := range ids {
		h.server.BroadcastTunnelDeleted(id)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"deleted": deleted})
}

func (h *Handlers) listTunnels(w http.ResponseWriter) {
	result := make([]tunnelStateMessage, 0, len(h.config.Tunnels))
	for _, t := range h.config.Tunnels {
		result = append(result, tunnelStateFor(h.manager, t))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handlers) createTunnel(w http.ResponseWriter, r *http.Request) {
	name, providerStr, port := h.parseTunnelRequest(r)
	if name == "" || providerStr == "" {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}

	if port == 0 {
		port = 30000
	}
	if port < 1 || port > 65535 {
		http.Error(w, "invalid port", http.StatusBadRequest)
		return
	}

	tunnel := config.TunnelConfig{
		ID:        h.config.NewTunnelID(name),
		Name:      name,
		Provider:  config.Provider(providerStr),
		LocalPort: port,
	}

	h.config.AddTunnel(tunnel)
	h.server.updateConfig()

	h.server.BroadcastTunnelUpdate(tunnel)
	h.writeTunnelJSON(w, tunnel)
}

func (h *Handlers) parseTunnelRequest(r *http.Request) (name, provider string, port int) {
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var req struct {
			Name      string `json:"name"`
			Provider  string `json:"provider"`
			LocalPort int    `json:"localPort"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return "", "", 0
		}
		return req.Name, req.Provider, req.LocalPort
	}

	if err := r.ParseForm(); err != nil {
		return "", "", 0
	}
	name = r.FormValue("name")
	provider = r.FormValue("provider")
	portStr := r.FormValue("port")
	port, _ = strconv.Atoi(portStr)
	return name, provider, port
}

func (h *Handlers) handleTunnelActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tunnels/")
	path = strings.Trim(path, "/")
	if path == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	id := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	if len(parts) > 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if action == "" {
			h.getTunnel(w, id)
		} else {
			http.Error(w, "unknown action", http.StatusBadRequest)
		}
	case http.MethodPut:
		if action == "" {
			h.updateTunnel(w, r, id)
		} else {
			http.Error(w, "unknown action", http.StatusBadRequest)
		}
	case http.MethodPost:
		switch action {
		case "start":
			h.startTunnel(w, id)
		case "stop":
			h.stopTunnel(w, id)
		default:
			http.Error(w, "unknown action", http.StatusBadRequest)
		}
	case http.MethodDelete:
		if action != "" {
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
		h.deleteTunnel(w, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) updateTunnel(w http.ResponseWriter, r *http.Request, id string) {
	name, providerStr, port := h.parseTunnelRequest(r)

	tunnel := h.server.getTunnel(id)
	if tunnel == nil {
		http.Error(w, "tunnel not found", http.StatusNotFound)
		return
	}

	if name != "" {
		tunnel.Name = name
	}
	if providerStr != "" {
		tunnel.Provider = config.Provider(providerStr)
	}
	if port >= 1 && port <= 65535 {
		tunnel.LocalPort = port
	}
	h.server.updateConfig()

	h.server.broadcastJSON(tunnelStateFor(h.manager, *tunnel))

	h.writeTunnelJSON(w, *tunnel)
}

func (h *Handlers) getTunnel(w http.ResponseWriter, id string) {
	tunnel := h.server.getTunnel(id)
	if tunnel == nil {
		http.Error(w, "tunnel not found", http.StatusNotFound)
		return
	}
	h.writeTunnelJSON(w, *tunnel)
}

func (h *Handlers) startTunnel(w http.ResponseWriter, id string) {
	tunnel := h.server.getTunnel(id)
	if tunnel == nil {
		http.Error(w, "tunnel not found", http.StatusNotFound)
		return
	}

	if needsInstall, canInstall := h.manager.CheckInstallation(tunnel.Provider); needsInstall && canInstall {
		h.server.broadcastJSON(tunnelStatePatch{
			Type:     "tunnel_state",
			ID:       tunnel.ID,
			State:    string(config.TunnelStateInstall),
			Provider: string(tunnel.Provider),
		})
		h.server.broadcastInstallingNotification(*tunnel)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       tunnel.ID,
			"name":     tunnel.Name,
			"provider": string(tunnel.Provider),
			"port":     tunnel.LocalPort,
			"state":    "installing",
		})

		go h.installAndStart(*tunnel)
		return
	}

	err := h.manager.Start(*tunnel, func(status config.TunnelStatus) {})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.writeTunnelJSON(w, *tunnel)
}

func (h *Handlers) installAndStart(tunnel config.TunnelConfig) {
	if err := h.manager.InstallProvider(tunnel.Provider); err != nil {
		h.server.broadcastJSON(tunnelStatePatch{
			Type:         "tunnel_state",
			ID:           tunnel.ID,
			State:        string(config.TunnelStateError),
			ErrorMessage: "Installation failed: " + err.Error(),
		})
		h.server.broadcastNotification("tunnel_error", "Tunnel Error", tunnel.Name+": Installation failed: "+err.Error(), "error", "error")
		return
	}

	if err := h.manager.Start(tunnel, func(status config.TunnelStatus) {}); err != nil {
		h.server.broadcastJSON(tunnelStatePatch{
			Type:         "tunnel_state",
			ID:           tunnel.ID,
			State:        string(config.TunnelStateError),
			ErrorMessage: err.Error(),
		})
		h.server.broadcastNotification("tunnel_error", "Tunnel Error", tunnel.Name+": "+err.Error(), "error", "error")
	}
}

func (h *Handlers) stopTunnel(w http.ResponseWriter, id string) {
	if err := h.manager.Stop(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tunnel := h.server.getTunnel(id)
	if tunnel != nil {
		h.writeTunnelJSON(w, *tunnel)
	}
}

func (h *Handlers) deleteTunnel(w http.ResponseWriter, id string) {
	h.manager.Stop(id)

	for i, t := range h.config.Tunnels {
		if t.ID == id {
			h.config.Tunnels = append(h.config.Tunnels[:i], h.config.Tunnels[i+1:]...)
			break
		}
	}
	h.server.updateConfig()
	h.server.BroadcastTunnelDeleted(id)

	w.WriteHeader(http.StatusOK)
}
