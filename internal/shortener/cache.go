package shortener

import (
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"

	"foundry-tunnel/internal/config"
)

type URLCache struct {
	mu       sync.RWMutex
	Mappings map[string]URLMapping `yaml:"mappings"`
}

type URLMapping struct {
	TunnelID    string `yaml:"tunnel_id"`
	ShortURL    string `yaml:"short_url"`
	CurrentURL  string `yaml:"current_url"`
	Provider    string `yaml:"provider"`
}

func NewCache() *URLCache {
	return &URLCache{
		Mappings: make(map[string]URLMapping),
	}
}

func (c *URLCache) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	path := cachePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	return yaml.Unmarshal(data, c)
}

func (c *URLCache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	dir := config.ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	
	return os.WriteFile(cachePath(), data, 0644)
}

func (c *URLCache) Set(tunnelID, shortURL, currentURL, provider string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.Mappings[tunnelID] = URLMapping{
		TunnelID:   tunnelID,
		ShortURL:   shortURL,
		CurrentURL: currentURL,
		Provider:   provider,
	}
}

func (c *URLCache) Get(tunnelID string) (URLMapping, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	m, ok := c.Mappings[tunnelID]
	return m, ok
}

func cachePath() string {
	return filepath.Join(config.ConfigDir(), "url-cache.yaml")
}

func (c *URLCache) EnsureShortURL(tunnelID, currentURL, preferredShort string, client *ISGDClient) (string, error) {
	if mapping, ok := c.Get(tunnelID); ok {
		if mapping.CurrentURL != currentURL && currentURL != "" {
			shortURL, err := client.Update(preferredShort, currentURL)
			if err != nil {
				if IsDomainBlocked(err) {
					return "", err
				}
				return mapping.ShortURL, nil
			}
			c.Set(tunnelID, shortURL, currentURL, "isgd")
			c.Save()
			return shortURL, nil
		}
		return mapping.ShortURL, nil
	}
	
	shortURL, err := client.Shorten(currentURL, preferredShort)
	if err != nil {
		return "", err
	}
	
	c.Set(tunnelID, shortURL, currentURL, "isgd")
	c.Save()
	
	return shortURL, nil
}
