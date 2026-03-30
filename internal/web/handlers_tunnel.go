package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sthbryan/ftm/internal/config"
)

func (h *Handlers) handleTunnels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listTunnels(w)
	case http.MethodPost:
		h.createTunnel(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) listTunnels(w http.ResponseWriter) {
	var result []map[string]interface{}
	for _, t := range h.config.Tunnels {
		item := h.tunnelToMap(t)
		result = append(result, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handlers) createTunnel(w http.ResponseWriter, r *http.Request) {
	name, providerStr, port := h.parseTunnelRequest(r)
	if name == "" || providerStr == "" || port < 1 {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}

	if port < 1 || port > 65535 {
		port = 30000
	}

	tunnel := config.TunnelConfig{
		ID:        fmt.Sprintf("tunnel-%d", time.Now().Unix()),
		Name:      name,
		Provider:  config.Provider(providerStr),
		LocalPort: port,
	}

	h.config.Tunnels = append(h.config.Tunnels, tunnel)
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
	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	id := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
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

	update := h.tunnelToMap(*tunnel)
	data, _ := MarshalJSON(update)
	h.server.broadcast(string(data))

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
		update := map[string]interface{}{
			"id":       tunnel.ID,
			"state":    "installing",
			"provider": string(tunnel.Provider),
		}
		data, _ := MarshalJSON(update)
		h.server.broadcast(string(data))

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
		update := map[string]interface{}{
			"id":           tunnel.ID,
			"state":        "error",
			"errorMessage": "Installation failed: " + err.Error(),
		}
		data, _ := MarshalJSON(update)
		h.server.broadcast(string(data))
		return
	}

	if err := h.manager.Start(tunnel, func(status config.TunnelStatus) {}); err != nil {
		update := map[string]interface{}{
			"id":           tunnel.ID,
			"state":        "error",
			"errorMessage": err.Error(),
		}
		data, _ := MarshalJSON(update)
		h.server.broadcast(string(data))
	}
}

func (h *Handlers) stopTunnel(w http.ResponseWriter, id string) {
	if err := h.manager.Stop(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	update := map[string]interface{}{
		"id":        id,
		"state":     "stopped",
		"publicUrl": "",
	}
	data, _ := MarshalJSON(update)
	h.server.broadcast(string(data))

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
	w.WriteHeader(http.StatusOK)
}
