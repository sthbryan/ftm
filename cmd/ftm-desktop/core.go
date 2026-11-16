package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/notifications"
	"github.com/sthbryan/ftm/internal/platform/instance"
	"github.com/sthbryan/ftm/internal/process"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/store"
	"github.com/sthbryan/ftm/internal/web"
)

// core is the application core this window owns: the store, the supervisor,
// the event bus, the service and the HTTP server every surface talks to.
// Only cmd/* wires these together; the desktop never goes through the TUI.
type core struct {
	service    *service.Service
	bus        *events.Bus
	manager    *process.Manager
	web        *web.Server
	expiration *notifications.ExpirationMonitor

	lock         *instance.Lock
	stop         context.CancelFunc
	shutdownOnce sync.Once
}

// startCore takes the single-instance lock, opens the config and serves the
// dashboard. When another ftm process (the TUI or `ftm serve`) already owns
// the core it returns that process as an *instance.RunningError and the
// window attaches to it instead.
func startCore(port int) (*core, error) {
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

	cfg := st.Snapshot()
	i18n.InitFromConfig(&cfg)

	manager := process.NewManager()
	bus := events.New()
	manager.SetEvents(bus)

	ctx, stop := context.WithCancel(context.Background())
	c := &core{
		service: service.New(st, manager, bus),
		bus:     bus,
		manager: manager,
		lock:    lock,
		stop:    stop,
	}

	manager.SetProviderExpiration(cfg.ProviderExpirationMinutes)
	manager.SetVisitorTracking(cfg.CountVisitors)
	c.wireNotifications(cfg)

	c.service.Run(ctx)

	if port > 0 {
		if err := c.service.SetWebPort(port); err != nil {
			log.Printf("Could not save the web port: %v", err)
		}
	}

	c.web = web.NewServer(manager, c.service, bus)
	if err := c.web.Start(); err != nil {
		c.web = nil
		c.shutdown()
		return nil, fmt.Errorf("failed to start web server: %w", err)
	}

	if err := lock.Publish(c.web.Port()); err != nil {
		log.Printf("Could not publish the runtime file: %v", err)
	}

	return c, nil
}

// wireNotifications sends OS notifications for tunnel changes while no
// dashboard is connected; an open window shows its own toasts instead.
func (c *core) wireNotifications(cfg config.Config) {
	notifications.Init()
	notifications.SetSoundEnabled(cfg.NotificationSound)
	notifications.SetNotificationsEnabled(cfg.NotificationsStatus == config.NotificationGranted)

	c.expiration = notifications.NewExpirationMonitor(notifications.ExpirationConfig{
		Thresholds:                cfg.ExpirationThresholds,
		ProviderExpirationMinutes: cfg.ProviderExpirationMinutes,
	}, func(name string, mins int) {
		if !c.nativeNotifications() {
			return
		}
		if mins == 0 {
			notifications.NotifyTunnelExpired(name)
		} else {
			notifications.NotifyTunnelExpiring(name, mins)
		}
	})

	c.manager.SetNotificationHandler(func(status config.TunnelStatus) {
		if !c.nativeNotifications() {
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

	c.manager.SetVisitorHandler(func(status config.TunnelStatus) {
		if c.web != nil {
			c.web.BroadcastNewVisitor(status)
		}
		if !c.nativeNotifications() {
			return
		}
		notifications.NotifyNewVisitor(status.Name, status.Visitors)
	})

	c.manager.SetExpirationCallbacks(
		c.expiration.Start,
		func(id string) { c.expiration.Stop(id) },
	)
}

func (c *core) nativeNotifications() bool {
	return c.web == nil || c.web.ClientCount() == 0
}

func (c *core) url() string { return c.web.URL() }

func (c *core) shutdown() {
	c.shutdownOnce.Do(func() {
		c.service.StopAll()
		if c.web != nil {
			if err := c.web.Stop(); err != nil {
				log.Printf("Error stopping web server: %v", err)
			}
		}
		c.stop()
		c.bus.Close()
		if c.expiration != nil {
			c.expiration.StopAll()
		}
		if err := c.lock.Release(); err != nil {
			log.Printf("Error releasing the instance lock: %v", err)
		}
	})
}
