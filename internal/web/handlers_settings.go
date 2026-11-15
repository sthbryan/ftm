package web

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sthbryan/ftm/internal/autostart"
	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/service"
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

func isSelectableLanguage(lang string) bool {
	if lang == config.LanguageAuto {
		return true
	}

	for _, supported := range i18n.AvailableLanguages() {
		if supported == lang {
			return true
		}
	}

	return false
}

func (h *Handlers) autostartManager() autostart.Manager {
	if h.autostart == nil {
		return autostart.New()
	}
	return h.autostart
}

func (h *Handlers) settingsPayload() map[string]interface{} {
	manager := h.autostartManager()
	supported := manager.Supported()

	enabled := false
	if supported {
		enabled, _ = manager.Enabled()
	}

	cfg := h.svc.Config()

	return map[string]interface{}{
		"notifications_enabled": cfg.NotificationsStatus,
		"notification_sound":    cfg.NotificationSound,
		"language":              cfg.Language,
		"language_resolved":     i18n.CurrentLanguage(),
		"onboarded":             cfg.Onboarded,
		"autostart_supported":   supported,
		"autostart_enabled":     enabled,
	}
}

func (h *Handlers) handleGetSettings(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.settingsPayload())
}

func (h *Handlers) handlePatchSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NotificationsEnabled *string `json:"notifications_enabled,omitempty"`
		NotificationSound    *bool   `json:"notification_sound,omitempty"`
		Language             *string `json:"language,omitempty"`
		Autostart            *bool   `json:"autostart_enabled,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Autostart != nil {
		if err := h.applyAutostart(*req.Autostart); err != nil {
			switch {
			case errors.Is(err, autostart.ErrUnsupported):
				http.Error(w, err.Error(), http.StatusNotImplemented)
			case errors.Is(err, autostart.ErrDisabledByUser):
				http.Error(w, err.Error(), http.StatusConflict)
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}

	if req.Language != nil && !isSelectableLanguage(*req.Language) {
		http.Error(w, "Unsupported language", http.StatusBadRequest)
		return
	}

	if _, err := h.svc.UpdateSettings(service.SettingsPatch{
		NotificationsStatus: req.NotificationsEnabled,
		NotificationSound:   req.NotificationSound,
		Language:            req.Language,
	}); err != nil {
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.settingsPayload())
}

func (h *Handlers) applyAutostart(enable bool) error {
	manager := h.autostartManager()

	if !manager.Supported() {
		return autostart.ErrUnsupported
	}

	if enable {
		return manager.Enable()
	}

	return manager.Disable()
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
		"available":    i18n.AvailableLanguages(),
	})
}

func (h *Handlers) handleI18nCurrent(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"translations": i18n.GetCurrentTranslations(),
		"language":     i18n.CurrentLanguage(),
	})
}
