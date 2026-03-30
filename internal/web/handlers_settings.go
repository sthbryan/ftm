package web

import (
	"encoding/json"
	"net/http"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/notifications"
)

func (h *Handlers) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetSettings(w)
	case http.MethodPatch:
		h.handlePatchSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) handleGetSettings(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"notifications_enabled": h.config.NotificationsStatus == config.NotificationGranted,
		"notification_sound":    h.config.NotificationSound,
		"theme":                 h.config.Theme,
	})
}

func (h *Handlers) handlePatchSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NotificationsEnabled *bool   `json:"notifications_enabled,omitempty"`
		NotificationSound    *bool   `json:"notification_sound,omitempty"`
		Theme                *string `json:"theme,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.NotificationsEnabled != nil {
		if *req.NotificationsEnabled {
			h.config.NotificationsStatus = config.NotificationGranted
		} else {
			h.config.NotificationsStatus = config.NotificationRejected
		}
	}

	if req.NotificationSound != nil {
		h.config.NotificationSound = *req.NotificationSound
	}

	if req.Theme != nil {
		if *req.Theme == "light" || *req.Theme == "dark" || *req.Theme == "system" {
			h.config.Theme = *req.Theme
		}
	}

	notifications.SetNotificationsEnabled(h.config.NotificationsStatus == config.NotificationGranted)
	notifications.SetSoundEnabled(h.config.NotificationSound)
	if err := h.config.Save(); err != nil {
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"notifications_enabled": h.config.NotificationsStatus == config.NotificationGranted,
		"notification_sound":    h.config.NotificationSound,
		"theme":                 h.config.Theme,
	})
}
