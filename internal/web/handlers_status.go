package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/version"
)

func (h *Handlers) handleStatus(w http.ResponseWriter) {
	status := h.config.NotificationsStatus
	if status != config.NotificationGranted && status != config.NotificationRejected {
		status = config.NotificationPending
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"port":                h.server.Port(),
		"version":             version.Version,
		"notificationsStatus": status,
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

func (h *Handlers) handleNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var data struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if data.Status != config.NotificationGranted && data.Status != config.NotificationRejected {
		http.Error(w, "Invalid status value", http.StatusBadRequest)
		return
	}

	h.config.NotificationsStatus = data.Status
	if err := h.config.Save(); err != nil {
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  h.config.NotificationsStatus,
	})
}
