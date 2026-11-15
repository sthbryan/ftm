package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

const (
	AppName              = "foundry-tunnel"
	ConfigFile           = "config.yaml"
	NotificationPending  = "pending"
	NotificationGranted  = "granted"
	NotificationRejected = "rejected"

	ConfigVersion = 4

	LanguageAuto = "auto"
)

type Config struct {
	Version  int            `yaml:"version"`
	Tunnels  []TunnelConfig `yaml:"tunnels"`
	WebPort  int            `yaml:"web_port,omitempty"`
	Language string         `yaml:"language"`

	Onboarded bool `yaml:"onboarded"`

	NotificationsStatus string `yaml:"notifications_status"`
	NotificationSound   bool   `yaml:"notification_sound"`
	CountVisitors       bool   `yaml:"count_visitors"`

	ExpirationThresholds      []int          `yaml:"expiration_thresholds"`
	ProviderExpirationMinutes map[string]int `yaml:"provider_expiration_minutes"`
}

func DefaultConfig() *Config {
	return &Config{
		Version:              ConfigVersion,
		Language:             LanguageAuto,
		Tunnels:              []TunnelConfig{},
		NotificationsStatus:  NotificationPending,
		NotificationSound:    true,
		CountVisitors:        true,
		ExpirationThresholds: []int{30, 15, 10, 5, 1},
		ProviderExpirationMinutes: map[string]int{
			"pinggy":       60,
			"serveo":       0,
			"cloudflared":  0,
			"tunnelmole":   0,
			"localhostrun": 0,
		},
	}
}

func (c *Config) migrate() bool {
	changed := false

	if c.Version < 2 {
		if c.Language == "" || c.Language == "en" {
			c.Language = LanguageAuto
		}
		changed = true
	}

	if c.Version < 3 {
		c.CountVisitors = true
		changed = true
	}

	if c.Version < 4 {
		c.Onboarded = len(c.Tunnels) > 0
		changed = true
	}

	if c.Version != ConfigVersion {
		c.Version = ConfigVersion
		changed = true
	}

	return changed
}

func (c *Config) NormalizeNotificationsStatus() {
	if c.NotificationsStatus != NotificationGranted && c.NotificationsStatus != NotificationRejected {
		c.NotificationsStatus = NotificationPending
	}
}

// ConfigDir returns the directory that holds config.yaml. Windows keeps user
// data under %AppData%; the other platforms stay on ~/.config.
func ConfigDir() string {
	return configDir(runtime.GOOS, userHomeDir(), userConfigDir())
}

// configDir is the pure part of ConfigDir, so the Windows and POSIX layouts can
// be tested from either host.
func configDir(goos, userHome, userConfig string) string {
	if goos == "windows" && userConfig != "" {
		return filepath.Join(userConfig, AppName)
	}

	if userHome == "" {
		userHome = "."
	}

	return filepath.Join(userHome, ".config", AppName)
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func userConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return dir
}

// legacyConfigDir is where every platform used to keep the config. The Windows
// build still has to read it once to move it to %AppData%.
func legacyConfigDir() string {
	home := userHomeDir()
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".config", AppName)
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), ConfigFile)
}

func (c *Config) Save() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write a sibling temp file and rename it into place. A crash, a full disk
	// or a scanner holding the file then leaves the previous config intact
	// instead of a half-written one that cannot be parsed on the next start.
	tmp, err := os.CreateTemp(dir, ConfigFile+".tmp*")
	if err != nil {
		return fmt.Errorf("failed to create a temporary config: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write config: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to flush config: %w", err)
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close config: %w", err)
	}

	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to set config permissions: %w", err)
	}

	if err := os.Rename(tmpPath, ConfigPath()); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to replace config: %w", err)
	}

	return nil
}

func Load() (*Config, error) {
	path := ConfigPath()

	migrateLegacyConfig(path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := cfg.Save(); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.NormalizeNotificationsStatus()

	migrated := cfg.migrate()
	deduped := cfg.dedupeTunnelIDs()

	if migrated || deduped {
		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("failed to persist migrated config: %w", err)
		}
	}

	return &cfg, nil
}

// migrateLegacyConfig copies a config written by an older build, which always
// used ~/.config, to the location this build reads. It only runs when the
// destination is missing, so it can never overwrite newer state.
func migrateLegacyConfig(path string) {
	if _, err := os.Stat(path); err == nil {
		return
	}

	legacy := filepath.Join(legacyConfigDir(), ConfigFile)
	if legacy == path {
		return
	}

	data, err := os.ReadFile(legacy)
	if err != nil {
		return
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}

	_ = os.WriteFile(path, data, 0o644)
}

func (c *Config) GetTunnel(id string) *TunnelConfig {
	for i := range c.Tunnels {
		if c.Tunnels[i].ID == id {
			return &c.Tunnels[i]
		}
	}
	return nil
}

func (c *Config) AddTunnel(t TunnelConfig) {
	c.Tunnels = append(c.Tunnels, t)
	c.Onboarded = true
}

func (c *Config) ClearTunnels() int {
	removed := len(c.Tunnels)
	c.Tunnels = []TunnelConfig{}
	return removed
}

func (c *Config) RemoveTunnel(id string) bool {
	for i, t := range c.Tunnels {
		if t.ID == id {
			c.Tunnels = append(c.Tunnels[:i], c.Tunnels[i+1:]...)
			return true
		}
	}
	return false
}
