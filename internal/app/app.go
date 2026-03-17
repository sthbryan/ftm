package app

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"foundry-tunnel/internal/config"
	"foundry-tunnel/internal/process"
	"foundry-tunnel/internal/shortener"
)

type App struct {
	Config      *config.Config
	Manager     *process.Manager
	Shortener   *shortener.ISGDClient
	URLCache    *shortener.URLCache
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	cache := shortener.NewCache()
	if err := cache.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load URL cache: %v\n", err)
	}

	return &App{
		Config:    cfg,
		Manager:   process.NewManager(),
		Shortener: shortener.NewISGD(),
		URLCache:  cache,
	}, nil
}

func (a *App) Run() error {
	if len(a.Config.Tunnels) == 0 {
		a.createDefaultTunnels()
	}

	model := NewModel(a)
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	return err
}

func (a *App) createDefaultTunnels() {
	a.Config.Tunnels = []config.TunnelConfig{
		{
			ID:        "foundry-default",
			Name:      "Foundry VTT (Default)",
			Provider:  config.ProviderPlayitgg,
			LocalPort: 30000,
			ShortURL:  "",
			AutoStart: false,
		},
	}
	a.Config.Save()
}

func (a *App) SaveConfig() error {
	return a.Config.Save()
}

func (a *App) EnsureShortURL(tunnelID, publicURL, preferred string) (string, error) {
	return a.URLCache.EnsureShortURL(tunnelID, publicURL, preferred, a.Shortener)
}
