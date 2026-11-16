package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sthbryan/ftm/internal/autostart"
	"github.com/sthbryan/ftm/internal/config"
)

type stubAutostart struct {
	supported bool
	enabled   bool
	enableErr error
	disabled  bool
}

func (s *stubAutostart) Supported() bool        { return s.supported }
func (s *stubAutostart) Enabled() (bool, error) { return s.enabled, nil }
func (s *stubAutostart) Repair() error          { return nil }

func (s *stubAutostart) Enable() error {
	if s.enableErr != nil {
		return s.enableErr
	}
	s.enabled = true
	return nil
}

func (s *stubAutostart) Disable() error {
	s.enabled = false
	s.disabled = true
	return nil
}

func settingsServer(t *testing.T, manager autostart.Manager) *Server {
	t.Helper()

	s := newTestServer(t, nil, config.DefaultConfig())
	s.autostart = manager

	return s
}

func decodeSettings(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("response does not decode: %v (%s)", err, body)
	}

	return payload
}

func patchSettings(t *testing.T, s *Server, body string) *httptest.ResponseRecorder {
	t.Helper()

	return call(t, s, http.MethodPatch, "/api/settings", body)
}

func TestGetSettingsHidesAutostartOnUnsupportedBuilds(t *testing.T) {
	s := settingsServer(t, &stubAutostart{supported: false, enabled: true})

	rec := call(t, s, http.MethodGet, "/api/settings", "")

	payload := decodeSettings(t, rec.Body.Bytes())

	if payload["autostart_supported"] != false {
		t.Errorf("autostart_supported = %v, want false", payload["autostart_supported"])
	}

	if payload["autostart_enabled"] != false {
		t.Errorf("autostart_enabled = %v, want false so the UI never shows a stale toggle", payload["autostart_enabled"])
	}
}

func TestGetSettingsReportsAutostartOnDesktopBuilds(t *testing.T) {
	s := settingsServer(t, &stubAutostart{supported: true, enabled: true})

	rec := call(t, s, http.MethodGet, "/api/settings", "")

	payload := decodeSettings(t, rec.Body.Bytes())

	if payload["autostart_supported"] != true {
		t.Errorf("autostart_supported = %v, want true", payload["autostart_supported"])
	}

	if payload["autostart_enabled"] != true {
		t.Errorf("autostart_enabled = %v, want true", payload["autostart_enabled"])
	}
}

func TestPatchAutostartIsRefusedOnUnsupportedBuilds(t *testing.T) {
	manager := &stubAutostart{supported: false}
	s := settingsServer(t, manager)

	rec := patchSettings(t, s, `{"autostart_enabled":true}`)

	if rec.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want %d so the CLI cannot register a console binary at login", rec.Code, http.StatusNotImplemented)
	}

	if manager.enabled {
		t.Error("the unsupported build enabled autostart anyway, want it left untouched")
	}
}

func TestPatchAutostartReportsAUserBlockedToggle(t *testing.T) {
	s := settingsServer(t, &stubAutostart{supported: true, enableErr: autostart.ErrDisabledByUser})

	rec := patchSettings(t, s, `{"autostart_enabled":true}`)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d so the UI can point the user at the system settings", rec.Code, http.StatusConflict)
	}
}

func TestPatchAutostartTogglesAndEchoesTheFullPayload(t *testing.T) {
	manager := &stubAutostart{supported: true}
	s := settingsServer(t, manager)

	rec := patchSettings(t, s, `{"autostart_enabled":true}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if !manager.enabled {
		t.Error("autostart was not enabled")
	}

	payload := decodeSettings(t, rec.Body.Bytes())

	for _, key := range []string{"autostart_supported", "autostart_enabled", "onboarded", "notification_sound", "language"} {
		if _, ok := payload[key]; !ok {
			t.Errorf("PATCH response is missing %q; the store replaces its state with this payload", key)
		}
	}

	rec = patchSettings(t, s, `{"autostart_enabled":false}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if !manager.disabled {
		t.Error("autostart was not disabled")
	}
}

func TestPatchWithoutAutostartLeavesItAlone(t *testing.T) {
	manager := &stubAutostart{supported: true, enabled: true}
	s := settingsServer(t, manager)

	rec := patchSettings(t, s, `{"notification_sound":false}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if manager.disabled {
		t.Error("changing another setting turned autostart off")
	}
}
