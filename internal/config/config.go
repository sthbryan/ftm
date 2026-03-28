package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	AppName    = "foundry-tunnel"
	ConfigFile = "config.yaml"
)

type Config struct {
	Version int            `yaml:"version"`
	Tunnels []TunnelConfig `yaml:"tunnels"`
	WebPort int            `yaml:"web_port,omitempty"`

	NotificationsEnabled bool `yaml:"notifications_enabled"`
	NotificationSound    bool `yaml:"notification_sound"`

	ExpirationThresholds      []int          `yaml:"expiration_thresholds"`
	ProviderExpirationMinutes map[string]int `yaml:"provider_expiration_minutes"`
}

func DefaultConfig() *Config {
	return &Config{
		Version:              1,
		Tunnels:              []TunnelConfig{},
		NotificationsEnabled: false,
		NotificationSound:    true,
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

func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
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

	path := ConfigPath()
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func Load() (*Config, error) {
	path := ConfigPath()

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

	return &cfg, nil
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
