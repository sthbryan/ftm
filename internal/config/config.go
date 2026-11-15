package config

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/sthbryan/ftm/internal/store/dirs"
)

const (
	AppName              = dirs.AppName
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

// ConfigDir returns the directory that holds config.yaml. It only reads
// store.Dirs and stays for internal/providers until that package takes a Dirs;
// new code uses store.Dirs.
func ConfigDir() string {
	return dirs.Default().Config
}

// Parse decodes a config file, normalises it and runs the migrations. changed
// reports whether the result differs from what was on disk and should be
// written back.
func Parse(data []byte) (cfg *Config, changed bool, err error) {
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, false, fmt.Errorf("failed to parse config: %w", err)
	}

	c.NormalizeNotificationsStatus()

	migrated := c.migrate()
	deduped := c.dedupeTunnelIDs()

	return &c, migrated || deduped, nil
}

// Marshal encodes the config as it is stored on disk.
func (c *Config) Marshal() ([]byte, error) {
	data, err := yaml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	return data, nil
}

// Clone returns a deep copy, so a reader can keep it while the store changes.
func (c *Config) Clone() *Config {
	out := *c
	if c.Tunnels != nil {
		out.Tunnels = append([]TunnelConfig(nil), c.Tunnels...)
	}
	if c.ExpirationThresholds != nil {
		out.ExpirationThresholds = append([]int(nil), c.ExpirationThresholds...)
	}
	if c.ProviderExpirationMinutes != nil {
		out.ProviderExpirationMinutes = make(map[string]int, len(c.ProviderExpirationMinutes))
		for k, v := range c.ProviderExpirationMinutes {
			out.ProviderExpirationMinutes[k] = v
		}
	}
	return &out
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
