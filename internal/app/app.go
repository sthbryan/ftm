package app

import (
	"fmt"
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/process"
	"github.com/sthbryan/ftm/internal/providers"
	"github.com/sthbryan/ftm/internal/web"
)

type App struct {
	Config           *config.Config
	Manager          *process.Manager
	WebServer        *web.Server
	DownloadProgress chan providers.DownloadProgress
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	manager := process.NewManager()

	app := &App{
		Config:           cfg,
		Manager:          manager,
		DownloadProgress: make(chan providers.DownloadProgress, 10),
	}

	app.Manager.SetProgressChannel(app.DownloadProgress)

	return app, nil
}

func (a *App) Run() error {
	if len(a.Config.Tunnels) == 0 {
		a.createDefaultTunnels()
	}

	if err := a.StartWebServer(); err != nil {
		return fmt.Errorf("failed to start web server: %w", err)
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

func (a *App) StartWebServer() error {
	a.WebServer = web.NewServer(a.Manager, a.Config)
	return a.WebServer.Start()
}

func (a *App) OpenDashboard() error {
	if a.WebServer == nil {
		return fmt.Errorf("web server not started")
	}
	url := a.WebServer.URL()

	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

func (a *App) OpenConfigDir() error {
	path := config.ConfigDir()

	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "windows":
		return exec.Command("explorer", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func (a *App) createDefaultTunnels() {
	a.Config.Tunnels = []config.TunnelConfig{
		{
			ID:        "foundry-default",
			Name:      "Foundry VTT (Default)",
			Provider:  config.ProviderCloudflared,
			LocalPort: 30000,
			AutoStart: false,
		},
	}
	a.Config.Save()
}

func (a *App) SaveConfig() error {
	return a.Config.Save()
}
