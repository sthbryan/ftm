package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/platform/autostart"
	"github.com/sthbryan/ftm/internal/service"
)

// Settings are the preferences the dashboard shows.
type Settings struct {
	NotificationsEnabled string `json:"notifications_enabled" doc:"granted, denied or unset"`
	NotificationSound    bool   `json:"notification_sound"`
	Language             string `json:"language" doc:"A language code, or auto"`
	LanguageResolved     string `json:"language_resolved"`
	Onboarded            bool   `json:"onboarded"`
	AutostartSupported   bool   `json:"autostart_supported"`
	AutostartEnabled     bool   `json:"autostart_enabled"`
}

type settingsPatch struct {
	_                    struct{} `json:"-" additionalProperties:"true"`
	NotificationsEnabled *string  `json:"notifications_enabled,omitempty"`
	NotificationSound    *bool    `json:"notification_sound,omitempty"`
	Language             *string  `json:"language,omitempty"`
	Autostart            *bool    `json:"autostart_enabled,omitempty"`
}

type settingsOutput struct {
	Body Settings
}

type settingsInput struct {
	Body settingsPatch
}

type i18nInput struct {
	Lang string `query:"lang" doc:"Language code; the current language when empty or unknown"`
}

type i18nOutput struct {
	Body struct {
		Translations map[string]string `json:"translations"`
		Current      string            `json:"current"`
		Available    []string          `json:"available"`
	}
}

type i18nCurrentOutput struct {
	Body struct {
		Translations map[string]string `json:"translations"`
		Language     string            `json:"language"`
	}
}

func isSelectableLanguage(lang string) bool {
	if lang == core.LanguageAuto {
		return true
	}
	for _, supported := range i18n.AvailableLanguages() {
		if supported == lang {
			return true
		}
	}
	return false
}

func (s *Server) settings() Settings {
	supported := s.autostart.Supported()
	enabled := false
	if supported {
		enabled, _ = s.autostart.Enabled()
	}

	cfg := s.svc.Config()
	return Settings{
		NotificationsEnabled: cfg.NotificationsStatus,
		NotificationSound:    cfg.NotificationSound,
		Language:             cfg.Language,
		LanguageResolved:     i18n.CurrentLanguage(),
		Onboarded:            cfg.Onboarded,
		AutostartSupported:   supported,
		AutostartEnabled:     enabled,
	}
}

func (s *Server) applyAutostart(enable bool) error {
	if !s.autostart.Supported() {
		return autostart.ErrUnsupported
	}
	if enable {
		return s.autostart.Enable()
	}
	return s.autostart.Disable()
}

func (s *Server) patchSettings(ctx context.Context, in *settingsInput) (*settingsOutput, error) {
	req := in.Body
	if req.Autostart != nil {
		if err := s.applyAutostart(*req.Autostart); err != nil {
			switch {
			case errors.Is(err, autostart.ErrUnsupported):
				return nil, apiError(http.StatusNotImplemented, "autostart_unsupported", err.Error())
			case errors.Is(err, autostart.ErrDisabledByUser):
				return nil, apiError(http.StatusConflict, "autostart_disabled_by_user", err.Error())
			default:
				return nil, apiError(http.StatusInternalServerError, "autostart_failed", err.Error())
			}
		}
	}

	if req.Language != nil && !isSelectableLanguage(*req.Language) {
		return nil, apiError(http.StatusBadRequest, "unsupported_language", "Unsupported language")
	}

	if _, err := s.svc.UpdateSettings(service.SettingsPatch{
		NotificationsStatus: req.NotificationsEnabled,
		NotificationSound:   req.NotificationSound,
		Language:            req.Language,
	}); err != nil {
		return nil, errSaveFailed(err)
	}
	return &settingsOutput{Body: s.settings()}, nil
}

func (s *Server) registerSettings(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-settings",
		Method:      http.MethodGet,
		Path:        "/api/settings",
		Summary:     "Read the preferences",
		Tags:        []string{tagSettings},
	}, func(ctx context.Context, _ *struct{}) (*settingsOutput, error) {
		return &settingsOutput{Body: s.settings()}, nil
	})

	for _, method := range []string{http.MethodPatch, http.MethodPost} {
		op := huma.Operation{
			OperationID: "update-settings",
			Method:      method,
			Path:        "/api/settings",
			Summary:     "Change the preferences that are set",
			Tags:        []string{tagSettings},
		}
		if method == http.MethodPost {
			op.OperationID = "update-settings-post"
			op.Summary = "Change the preferences that are set (same as PATCH)"
			op.Deprecated = true
		}
		huma.Register(api, op, s.patchSettings)
	}

	huma.Register(api, huma.Operation{
		OperationID: "get-translations",
		Method:      http.MethodGet,
		Path:        "/api/i18n",
		Summary:     "Translations for one language",
		Tags:        []string{tagSettings},
	}, func(ctx context.Context, in *i18nInput) (*i18nOutput, error) {
		lang := in.Lang
		if lang == "" {
			lang = i18n.CurrentLanguage()
		}
		all := i18n.TranslationsMap()
		current := all[lang]
		if current == nil {
			current = all[i18n.DefaultLang]
			lang = i18n.DefaultLang
		}

		out := &i18nOutput{}
		out.Body.Translations = current
		out.Body.Current = lang
		out.Body.Available = i18n.AvailableLanguages()
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-current-translations",
		Method:      http.MethodGet,
		Path:        "/api/i18n/current",
		Summary:     "Translations for the current language",
		Tags:        []string{tagSettings},
	}, func(ctx context.Context, _ *struct{}) (*i18nCurrentOutput, error) {
		out := &i18nCurrentOutput{}
		out.Body.Translations = i18n.GetCurrentTranslations()
		out.Body.Language = i18n.CurrentLanguage()
		return out, nil
	})
}
