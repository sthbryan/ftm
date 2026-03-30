package app

import (
	"fmt"
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/notifications"
	"github.com/sthbryan/ftm/internal/process"
	"github.com/sthbryan/ftm/internal/providers"
	"github.com/sthbryan/ftm/internal/web"
)

type App struct {
	Config            *config.Config
	Manager           *process.Manager
	WebServer         *web.Server
	DownloadProgress   chan providers.DownloadProgress
	ExpirationMonitor *notifications.ExpirationMonitor
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
	notifications.Init()
	notifications.SetSoundEnabled(cfg.NotificationSound)
	notifications.SetNotificationsEnabled(cfg.NotificationsStatus == config.NotificationGranted)

	expConfig := notifications.ExpirationConfig{
		Thresholds:                 cfg.ExpirationThresholds,
		ProviderExpirationMinutes:  cfg.ProviderExpirationMinutes,
	}
	app.ExpirationMonitor = notifications.NewExpirationMonitor(expConfig, func(name string, mins int) {
		if !app.shouldUseNativeNotifications() {
			return
		}
		if mins == 0 {
			notifications.NotifyTunnelExpired(name)
		} else {
			notifications.NotifyTunnelExpiring(name, mins)
		}
	})

	app.Manager.SetNotificationHandler(func(status config.TunnelStatus) {
		if !app.shouldUseNativeNotifications() {
			return
		}
		switch status.State {
		case config.TunnelStateOnline:
			notifications.NotifyTunnelOnline(status.Name, status.PublicURL)
		case config.TunnelStateError:
			notifications.NotifyTunnelError(status.Name, status.ErrorMessage)
		case config.TunnelStateTimeout:
			notifications.NotifyTunnelTimeout(status.Name)
		case config.TunnelStateStopping:
			notifications.NotifyTunnelStopped(status.Name)
		}
	})

	app.Manager.SetExpirationCallbacks(
		app.ExpirationMonitor.Start,
		func(id string) { app.ExpirationMonitor.Stop(id) },
	)

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
		tea.WithMouseAllMotion(),
	)

	_, err := p.Run()
	return err
}

func (a *App) StartWebServer() error {
	a.WebServer = web.NewServer(a.Manager, a.Config)
	a.Manager.SetStatusChannel(a.WebServer.StatusChannel)
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
		},
	}
	a.Config.Save()
}

func (a *App) SaveConfig() error {
	notifications.SetSoundEnabled(a.Config.NotificationSound)
	notifications.SetNotificationsEnabled(a.Config.NotificationsStatus == config.NotificationGranted)
	return a.Config.Save()
}

func (a *App) shouldUseNativeNotifications() bool {
	if a.WebServer == nil {
		return true
	}
	return a.WebServer.ClientCount() == 0
}

func (a *App) Shutdown() {
	if a.WebServer != nil {
		a.WebServer.Stop()
	}
	if a.Manager != nil {
		a.Manager.StopAll()
	}
	if a.ExpirationMonitor != nil {
		a.ExpirationMonitor.StopAll()
	}
}
