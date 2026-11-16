package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/notifications"
	"github.com/sthbryan/ftm/internal/platform/instance"
	"github.com/sthbryan/ftm/internal/process"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/store"
	"github.com/sthbryan/ftm/internal/transport/httpapi"
	"github.com/sthbryan/ftm/internal/updater"
)

// App wires the core for one process. Every surface changes things through
// Service and learns about changes from Bus; Manager is only read here.
type App struct {
	Service           *service.Service
	Bus               *events.Bus
	Manager           *process.Manager
	WebServer         *httpapi.Server
	ExpirationMonitor *notifications.ExpirationMonitor

	instance     *instance.Lock
	stop         context.CancelFunc
	shutdownOnce sync.Once
}

// New takes the single-instance lock and opens the config. When another ftm
// process already owns them it returns that process as an
// *instance.RunningError, and the caller attaches to it instead.
func New() (*App, error) {

	if err := i18n.Load(); err != nil {
		return nil, fmt.Errorf("failed to load translations: %w", err)
	}

	dirs := store.DefaultDirs()

	lock, err := instance.Acquire(dirs.Runtime)
	if err != nil {
		return nil, err
	}

	st, err := store.Open(dirs)
	if err != nil {
		_ = lock.Release()
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	snapshot := st.Snapshot()
	cfg := &snapshot

	i18n.InitFromConfig(cfg)

	manager := process.NewManager()
	bus := events.New()
	manager.SetEvents(bus)

	ctx, stop := context.WithCancel(context.Background())
	app := &App{
		Service:  service.New(st, manager, bus),
		Bus:      bus,
		instance: lock,
		Manager:  manager,
		stop:     stop,
	}

	app.Manager.SetProviderExpiration(cfg.ProviderExpirationMinutes)
	app.Manager.SetVisitorTracking(cfg.CountVisitors)
	notifications.Init()
	notifications.SetSoundEnabled(cfg.NotificationSound)
	notifications.SetNotificationsEnabled(cfg.NotificationsStatus == config.NotificationGranted)

	expConfig := notifications.ExpirationConfig{
		Thresholds:                cfg.ExpirationThresholds,
		ProviderExpirationMinutes: cfg.ProviderExpirationMinutes,
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

	app.Manager.SetVisitorHandler(func(status config.TunnelStatus) {
		if app.WebServer != nil {
			app.WebServer.BroadcastNewVisitor(status)
		}
		if !app.shouldUseNativeNotifications() {
			return
		}
		notifications.NotifyNewVisitor(status.Name, status.Visitors)
	})

	app.Manager.SetExpirationCallbacks(
		app.ExpirationMonitor.Start,
		func(id string) { app.ExpirationMonitor.Stop(id) },
	)

	app.Service.Run(ctx)

	return app, nil
}

func (a *App) Run() error {
	if file := redirectLog(); file != nil {
		defer file.Close()
	}

	if err := a.StartWebServer(); err != nil {
		return fmt.Errorf("failed to start web server: %w", err)
	}

	model := NewModel(a)
	p := tea.NewProgram(model)

	// An update applied from the dashboard ends the TUI the same way one
	// applied from the TUI does: quit, shut down, relaunch.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-a.RestartRequested():
			p.Send(updateApplyMsg{})
		case <-done:
		}
	}()

	final, err := p.Run()
	if err != nil {
		return err
	}

	if m, ok := final.(*Model); ok && m.RelaunchOnExit {
		a.Shutdown()
		return updater.Relaunch()
	}

	return nil
}

func (a *App) StartWebServer() error {
	if a.WebServer != nil {
		return nil
	}

	a.WebServer = httpapi.NewServer(a.Service, a.Bus)
	if a.instance != nil {
		a.WebServer.SetToken(a.instance.Token())
	}

	if err := a.WebServer.Start(); err != nil {
		a.WebServer = nil
		return err
	}

	if a.instance != nil {
		if err := a.instance.Publish(a.WebServer.Port()); err != nil {
			fmt.Fprintf(os.Stderr, "Could not publish the runtime file: %v\n", err)
		}
	}

	return nil
}

// RestartRequested is closed when something asked ftm to restart, such as an
// update applied from the dashboard. The owner of the process shuts down and
// relaunches when it fires.
func (a *App) RestartRequested() <-chan struct{} {
	return a.Service.RestartRequested()
}

// SetWebPort makes the web server prefer port the next time it starts.
func (a *App) SetWebPort(port int) error {
	return a.Service.SetWebPort(port)
}

func (a *App) OpenDashboard() error {
	if a.WebServer == nil {
		return fmt.Errorf("web server not started")
	}

	return OpenURL(a.WebServer.URL())
}

// OpenURL opens url in the default browser.
func OpenURL(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return startAndReap(exec.Command("open", url))
	case "windows":
		return startAndReap(exec.Command("rundll32", "url.dll,FileProtocolHandler", url))
	default:
		return startAndReap(exec.Command("xdg-open", url))
	}
}

func (a *App) OpenConfigDir() error {
	path := a.Service.Dirs().Config

	switch runtime.GOOS {
	case "darwin":
		return startAndReap(exec.Command("open", path))
	case "windows":
		return startAndReap(exec.Command("explorer", path))
	default:
		return startAndReap(exec.Command("xdg-open", path))
	}
}

var onReaped = func(*exec.Cmd) {}

func startAndReap(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		_ = cmd.Wait()
		onReaped(cmd)
	}()

	return nil
}

// Config returns a copy of the current config.
func (a *App) Config() config.Config {
	return a.Service.Config()
}

func (a *App) shouldUseNativeNotifications() bool {
	if a.WebServer == nil {
		return true
	}
	return a.WebServer.ClientCount() == 0
}

func (a *App) Shutdown() {
	a.shutdownOnce.Do(func() {

		if a.Service != nil {
			a.Service.StopAll()
		}
		if a.WebServer != nil {
			if err := a.WebServer.Stop(); err != nil {
				fmt.Fprintf(os.Stderr, "Error stopping web server: %v\n", err)
			}
		}
		if a.stop != nil {
			a.stop()
		}
		if a.Bus != nil {
			a.Bus.Close()
		}
		if a.ExpirationMonitor != nil {
			a.ExpirationMonitor.StopAll()
		}
		if err := a.instance.Release(); err != nil {
			fmt.Fprintf(os.Stderr, "Error releasing the instance lock: %v\n", err)
		}
	})
}
