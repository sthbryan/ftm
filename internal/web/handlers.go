package web

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sthbryan/ftm/internal/clipboard"
	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/process"
	"github.com/sthbryan/ftm/internal/version"
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
	case r.URL.Path == "/api/tunnels" || r.URL.Path == "/api/tunnels/":
		h.handleTunnels(w, r)
	case r.URL.Path == "/api/copy-url":
		h.handleCopyURL(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/logs/"):
		if strings.HasSuffix(r.URL.Path, "/stream") {
			h.handleLogsStream(w, r)
		} else {
			h.handleLogs(w, r)
		}
	case r.URL.Path == "/api/tunnels/reorder":
		h.handleReorder(w, r)
	case r.URL.Path == "/api/status":
		h.handleStatus(w)
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
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

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

	if needsInstall, canInstall := h.manager.CheckInstallation(t.Provider); needsInstall && canInstall {
		item["state"] = "installing"
		item["installProgress"] = 0
	}

	return item
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

func (h *Handlers) handleCopyURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	clipboard.Write(req.URL)
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id := h.extractLogID(r.URL.Path)
	if id == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	logs := h.manager.GetLogs(id)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(strings.Join(logs, "\n")))
}

func (h *Handlers) handleLogsStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/logs/")
	path = strings.TrimSuffix(path, "/stream")
	id := strings.Split(path, "/")[0]
	if id == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	for _, line := range h.manager.GetLogs(id) {
		fmt.Fprintf(w, "data: %s\n\n", line)
		flusher.Flush()
	}

	logChan := h.manager.SubscribeLogs(id)
	if logChan == nil {
		fmt.Fprintf(w, "data: [tunnel stopped]\n\n")
		flusher.Flush()
		return
	}

	for line := range logChan {
		fmt.Fprintf(w, "data: %s\n\n", line)
		flusher.Flush()
	}
}

func (h *Handlers) extractLogID(path string) string {
	path = strings.TrimPrefix(path, "/api/logs/")
	parts := strings.Split(path, "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return ""
}

func (h *Handlers) handleReorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var newOrder []string
	if err := json.NewDecoder(r.Body).Decode(&newOrder); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	existing := h.config.Tunnels
	tunnelByID := make(map[string]config.TunnelConfig, len(existing))
	for i := range existing {
		tunnelByID[existing[i].ID] = existing[i]
	}

	reordered := make([]config.TunnelConfig, 0, len(existing))
	seen := make(map[string]bool, len(existing))
	for _, id := range newOrder {
		if t, ok := tunnelByID[id]; ok && !seen[id] {
			reordered = append(reordered, t)
			seen[id] = true
		}
	}

	for i := range existing {
		if !seen[existing[i].ID] {
			reordered = append(reordered, existing[i])
		}
	}

	h.config.Tunnels = reordered
	h.server.updateConfig()
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) handleStatus(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"port":    h.server.Port(),
		"version": version.Version,
	})
}

func (h *Handlers) handleProviders(w http.ResponseWriter) {
	providers := []map[string]string{
		{"id": "cloudflared", "name": "Cloudflared"},
		{"id": "tunnelmole", "name": "Tunnelmole"},
		{"id": "localhostrun", "name": "localhost.run"},
		{"id": "serveo", "name": "Serveo"},
		{"id": "pinggy", "name": "Pinggy"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

func (h *Handlers) handleDetectPort(w http.ResponseWriter) {
	commonPorts := []int{30000, 30001, 30002, 30003, 30004, 30005, 30006, 30007, 30008, 30009}
	found := []int{}

	for _, port := range commonPorts {
		if ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port)); err == nil {
			ln.Close()
		} else {
			found = append(found, port)
		}
	}

	suggested := 30000
	if len(found) > 0 {
		suggested = found[0]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ports":     found,
		"suggested": suggested,
	})
}

func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

type TunnelView struct {
	ID           string
	Name         string
	Provider     string
	ProviderName string
	Port         int
	State        string
	PublicURL    string
	ErrorMessage string
}

