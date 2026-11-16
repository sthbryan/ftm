package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/platform/instance"
	"github.com/sthbryan/ftm/internal/platform/notify"
	"github.com/sthbryan/ftm/internal/platform/open"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/store"
	"github.com/sthbryan/ftm/internal/supervisor"
	"github.com/sthbryan/ftm/internal/transport/httpapi"
	"github.com/sthbryan/ftm/internal/ui/tui"
	"github.com/sthbryan/ftm/internal/updater"
)

// App wires the core for one ftm process: the store, the supervisor, the bus,
// the service and the web server. Every surface changes things through
// Service and learns about changes from Bus; Manager is only configured here.
type App struct {
	Service           *service.Service
	Bus               *events.Bus
	Manager           *supervisor.Manager
	WebServer         *httpapi.Server
	ExpirationMonitor *notify.ExpirationMonitor

	instance     *instance.Lock
	stop         context.CancelFunc
	shutdownOnce sync.Once
}

// New takes the single-instance lock and opens the config. When another ftm
// process already owns them it returns that process as an
// *instance.RunningError, and the caller attaches to it instead.
func newApp() (*App, error) {

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

	i18n.UseLanguage(cfg.Language)

	manager := supervisor.NewManager()
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
	notify.Init()
	notify.SetSoundEnabled(cfg.NotificationSound)
	notify.SetNotificationsEnabled(cfg.NotificationsStatus == core.NotificationGranted)

	expConfig := notify.ExpirationConfig{
		Thresholds:                cfg.ExpirationThresholds,
		ProviderExpirationMinutes: cfg.ProviderExpirationMinutes,
	}
	app.ExpirationMonitor = notify.NewExpirationMonitor(expConfig, func(name string, mins int) {
		if !app.shouldUseNativeNotifications() {
			return
		}
		if mins == 0 {
			notify.NotifyTunnelExpired(name)
		} else {
			notify.NotifyTunnelExpiring(name, mins)
		}
	})

	app.Manager.SetNotificationHandler(func(status core.TunnelStatus) {
		if !app.shouldUseNativeNotifications() {
			return
		}
		switch status.State {
		case core.TunnelStateOnline:
			notify.NotifyTunnelOnline(status.Name, status.PublicURL)
		case core.TunnelStateError:
			notify.NotifyTunnelError(status.Name, status.ErrorMessage)
		case core.TunnelStateTimeout:
			notify.NotifyTunnelTimeout(status.Name)
		case core.TunnelStateStopping:
			notify.NotifyTunnelStopped(status.Name)
		}
	})

	app.Manager.SetVisitorHandler(func(status core.TunnelStatus) {
		if app.WebServer != nil {
			app.WebServer.BroadcastNewVisitor(status)
		}
		if !app.shouldUseNativeNotifications() {
			return
		}
		notify.NotifyNewVisitor(status.Name, status.Visitors)
	})

	app.Manager.SetExpirationCallbacks(
		app.ExpirationMonitor.Start,
		func(id string) { app.ExpirationMonitor.Stop(id) },
	)

	app.Service.Run(ctx)

	return app, nil
}

// Run shows the TUI until the user quits, then relaunches when an update was
// applied from it or from the dashboard.
func (a *App) Run() error {
	if file := redirectLog(); file != nil {
		defer file.Close()
	}

	if err := a.StartWebServer(); err != nil {
		return fmt.Errorf("failed to start web server: %w", err)
	}

	relaunch, err := tui.Run(&tui.App{Service: a.Service, Bus: a.Bus, Web: a.WebServer})
	if err != nil {
		return err
	}

	if relaunch {
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

	return open.URL(a.WebServer.URL())
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
