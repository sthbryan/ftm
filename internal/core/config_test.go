package core

import (
	"testing"
)

func TestMigrateV1EnglishBecomesAuto(t *testing.T) {
	tests := []struct {
		name    string
		version int
		lang    string
		want    string
	}{
		{"v1 default english", 1, "en", LanguageAuto},
		{"v1 empty", 1, "", LanguageAuto},
		{"v1 explicit spanish is kept", 1, "es", "es"},
		{"v2 english is a real choice", 2, "en", "en"},
		{"v2 auto stays auto", 2, LanguageAuto, LanguageAuto},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Version: tt.version, Language: tt.lang}
			cfg.migrate()

			if cfg.Language != tt.want {
				t.Errorf("Language = %q, want %q", cfg.Language, tt.want)
			}
			if cfg.Version != ConfigVersion {
				t.Errorf("Version = %d, want %d", cfg.Version, ConfigVersion)
			}
		})
	}
}

func TestMigrateReportsWhetherItChangedAnything(t *testing.T) {
	current := &Config{Version: ConfigVersion, Language: "es"}
	if current.migrate() {
		t.Error("migrate() = true for an up-to-date config, want false")
	}

	old := &Config{Version: 1, Language: "en"}
	if !old.migrate() {
		t.Error("migrate() = false for a v1 config, want true")
	}
}

func TestNormalizeNotificationsStatus(t *testing.T) {
	tests := map[string]string{
		NotificationGranted:  NotificationGranted,
		NotificationRejected: NotificationRejected,
		NotificationPending:  NotificationPending,
		"":                   NotificationPending,
		"nonsense":           NotificationPending,
	}

	for input, want := range tests {
		cfg := &Config{NotificationsStatus: input}
		cfg.NormalizeNotificationsStatus()

		if cfg.NotificationsStatus != want {
			t.Errorf("normalizing %q gave %q, want %q", input, cfg.NotificationsStatus, want)
		}
	}
}

func TestTunnelCRUD(t *testing.T) {
	cfg := DefaultConfig()

	cfg.AddTunnel(TunnelConfig{ID: "a", Name: "First", LocalPort: 30000})
	cfg.AddTunnel(TunnelConfig{ID: "b", Name: "Second", LocalPort: 30001})

	if got := cfg.GetTunnel("b"); got == nil || got.Name != "Second" {
		t.Fatalf("GetTunnel(b) = %+v, want the second tunnel", got)
	}
	if got := cfg.GetTunnel("missing"); got != nil {
		t.Errorf("GetTunnel(missing) = %+v, want nil", got)
	}

	if !cfg.RemoveTunnel("a") {
		t.Error("RemoveTunnel(a) = false, want true")
	}
	if cfg.RemoveTunnel("a") {
		t.Error("RemoveTunnel(a) twice = true, want false")
	}
	if len(cfg.Tunnels) != 1 || cfg.Tunnels[0].ID != "b" {
		t.Fatalf("Tunnels = %+v, want only b", cfg.Tunnels)
	}
}

func TestMigrateDerivesOnboardedFromExistingTunnels(t *testing.T) {
	tests := []struct {
		name    string
		version int
		tunnels []TunnelConfig
		want    bool
	}{
		{"v3 with a tunnel has used the app", 3, []TunnelConfig{{ID: "a"}}, true},
		{"v3 without tunnels never got started", 3, nil, false},
		{"v1 with a tunnel has used the app", 1, []TunnelConfig{{ID: "a"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Version: tt.version, Tunnels: tt.tunnels}
			cfg.migrate()

			if cfg.Onboarded != tt.want {
				t.Errorf("Onboarded = %v, want %v", cfg.Onboarded, tt.want)
			}
		})
	}
}

func TestMigrateLeavesOnboardedAloneOnCurrentConfigs(t *testing.T) {
	cfg := &Config{Version: ConfigVersion, Onboarded: false, Tunnels: []TunnelConfig{{ID: "a"}}}
	cfg.migrate()

	if cfg.Onboarded {
		t.Error("Onboarded = true, want a v4 config to keep the flag it was saved with")
	}
}

func TestFreshConfigIsNotOnboarded(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Onboarded {
		t.Error("Onboarded = true on a fresh config, want false so the welcome screen shows")
	}
}

func TestAddTunnelMarksTheConfigOnboarded(t *testing.T) {
	cfg := DefaultConfig()

	cfg.AddTunnel(TunnelConfig{ID: "a", Name: "First", LocalPort: 30000})
	if !cfg.Onboarded {
		t.Fatal("Onboarded = false after adding a tunnel, want true")
	}

	cfg.RemoveTunnel("a")
	if !cfg.Onboarded {
		t.Error("Onboarded = false after removing the last tunnel, want it to stay true")
	}
}

func TestClearTunnelsRemovesEveryConnection(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AddTunnel(TunnelConfig{ID: "a", Name: "First", LocalPort: 30000})
	cfg.AddTunnel(TunnelConfig{ID: "b", Name: "Second", LocalPort: 30001})

	if got := cfg.ClearTunnels(); got != 2 {
		t.Errorf("ClearTunnels() = %d, want 2", got)
	}
	if len(cfg.Tunnels) != 0 {
		t.Fatalf("Tunnels = %+v, want empty", cfg.Tunnels)
	}
	if cfg.Tunnels == nil {
		t.Error("Tunnels = nil, want an empty slice so it marshals as [] and not null")
	}
	if got := cfg.ClearTunnels(); got != 0 {
		t.Errorf("ClearTunnels() on an empty config = %d, want 0", got)
	}
}

func TestClearTunnelsKeepsPreferences(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Language = "es"
	cfg.NotificationsStatus = NotificationGranted
	cfg.AddTunnel(TunnelConfig{ID: "a", Name: "First", LocalPort: 30000})

	cfg.ClearTunnels()

	if !cfg.Onboarded {
		t.Error("Onboarded = false after clearing, want true so the empty home shows and not the welcome")
	}
	if cfg.Language != "es" {
		t.Errorf("Language = %q, want it untouched", cfg.Language)
	}
	if cfg.NotificationsStatus != NotificationGranted {
		t.Errorf("NotificationsStatus = %q, want it untouched", cfg.NotificationsStatus)
	}
}

func TestGetTunnelReturnsLivePointer(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AddTunnel(TunnelConfig{ID: "a", Name: "First", LocalPort: 30000})

	cfg.GetTunnel("a").LocalPort = 30500

	if got := cfg.Tunnels[0].LocalPort; got != 30500 {
		t.Fatalf("LocalPort = %d, want the mutation to stick", got)
	}
}
