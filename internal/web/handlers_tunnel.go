package web

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/service"
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
	deleted, err := h.svc.ClearTunnels()
	if err != nil {
		log.Printf("web: could not clear the connections: %v", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"deleted": deleted})
}

func (h *Handlers) listTunnels(w http.ResponseWriter) {
	tunnels := h.svc.Tunnels()
	result := make([]tunnelStateMessage, 0, len(tunnels))
	for _, t := range tunnels {
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

	tunnel, err := h.svc.CreateTunnel(service.TunnelInput{
		Name:      name,
		Provider:  config.Provider(providerStr),
		LocalPort: port,
	})
	if err != nil {
		log.Printf("web: could not save the connection: %v", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

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

	// Empty fields and an out-of-range port leave the saved value as it is.
	var patch service.TunnelPatch
	if name != "" {
		patch.Name = &name
	}
	if providerStr != "" {
		provider := config.Provider(providerStr)
		patch.Provider = &provider
	}
	if port >= 1 && port <= 65535 {
		patch.LocalPort = &port
	}

	tunnel, err := h.svc.UpdateTunnel(id, patch)
	if errors.Is(err, service.ErrTunnelNotFound) {
		http.Error(w, "tunnel not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("web: could not save the connection: %v", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	h.writeTunnelJSON(w, tunnel)
}

func (h *Handlers) getTunnel(w http.ResponseWriter, id string) {
	tunnel, ok := h.server.getTunnel(id)
	if !ok {
		http.Error(w, "tunnel not found", http.StatusNotFound)
		return
	}
	h.writeTunnelJSON(w, tunnel)
}

func (h *Handlers) startTunnel(w http.ResponseWriter, id string) {
	tunnel, ok := h.server.getTunnel(id)
	if !ok {
		http.Error(w, "tunnel not found", http.StatusNotFound)
		return
	}

	installing, err := h.svc.StartTunnel(id)
	if errors.Is(err, service.ErrTunnelNotFound) {
		http.Error(w, "tunnel not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if installing {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       tunnel.ID,
			"name":     tunnel.Name,
			"provider": string(tunnel.Provider),
			"port":     tunnel.LocalPort,
			"state":    "installing",
		})
		return
	}

	h.writeTunnelJSON(w, tunnel)
}

func (h *Handlers) stopTunnel(w http.ResponseWriter, id string) {
	if err := h.svc.StopTunnel(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if tunnel, ok := h.server.getTunnel(id); ok {
		h.writeTunnelJSON(w, tunnel)
	}
}

func (h *Handlers) deleteTunnel(w http.ResponseWriter, id string) {
	if err := h.svc.DeleteTunnel(id); err != nil {
		log.Printf("web: could not delete the connection: %v", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
