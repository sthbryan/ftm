package web

import (
	"encoding/json"
	"net/http"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/notifications"
)

func (h *Handlers) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetSettings(w)
	case http.MethodPatch, http.MethodPost:
		h.handlePatchSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) handleGetSettings(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"notifications_enabled": h.config.NotificationsStatus,
		"notification_sound":    h.config.NotificationSound,
		"language":              h.config.Language,
	})
}

func (h *Handlers) handlePatchSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NotificationsEnabled *string `json:"notifications_enabled,omitempty"`
		NotificationSound    *bool   `json:"notification_sound,omitempty"`
		Language             *string `json:"language,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.NotificationsEnabled != nil {
		h.config.NotificationsStatus = *req.NotificationsEnabled
	}

	if req.NotificationSound != nil {
		h.config.NotificationSound = *req.NotificationSound
	}

	if req.Language != nil {
		validLangs := i18n.AvailableLanguages()
		isValid := false
		for _, l := range validLangs {
			if l == *req.Language {
				isValid = true
				break
			}
		}
		if isValid {
			h.config.Language = *req.Language
			i18n.ChangeLanguage(*req.Language)
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
		"notifications_enabled": h.config.NotificationsStatus,
		"notification_sound":    h.config.NotificationSound,
		"language":              h.config.Language,
	})
}

func (h *Handlers) handleI18n(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = i18n.CurrentLanguage()
	}

	w.Header().Set("Content-Type", "application/json")

	allTranslations := i18n.TranslationsMap()
	currentTrans := allTranslations[lang]
	if currentTrans == nil {
		currentTrans = allTranslations[i18n.DefaultLang]
		lang = i18n.DefaultLang
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"translations": currentTrans,
		"current":      lang,
		"available":     i18n.AvailableLanguages(),
	})
}

func (h *Handlers) handleI18nCurrent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"translations": i18n.GetCurrentTranslations(),
		"language":     i18n.CurrentLanguage(),
	})
}
