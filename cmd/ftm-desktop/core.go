package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/platform/instance"
	"github.com/sthbryan/ftm/internal/platform/notify"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/store"
	"github.com/sthbryan/ftm/internal/supervisor"
	"github.com/sthbryan/ftm/internal/transport/httpapi"
)

// core is the application core this window owns: the store, the supervisor,
// the event bus, the service and the HTTP server every surface talks to.
// Only cmd/* wires these together; the desktop never goes through the TUI.
type appCore struct {
	service    *service.Service
	bus        *events.Bus
	manager    *supervisor.Manager
	web        *httpapi.Server
	expiration *notify.ExpirationMonitor

	lock         *instance.Lock
	stop         context.CancelFunc
	shutdownOnce sync.Once
}

// startCore takes the single-instance lock, opens the config and serves the
// dashboard. When another ftm process (the TUI or `ftm serve`) already owns
// the core it returns that process as an *instance.RunningError and the
// window attaches to it instead.
func startCore(port int) (*appCore, error) {
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

	manager := supervisor.NewManager()
	bus := events.New()
	manager.SetEvents(bus)

	ctx, stop := context.WithCancel(context.Background())
	c := &appCore{
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

	c.web = httpapi.NewServer(c.service, bus)
	c.web.SetToken(lock.Token())
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
func (c *appCore) wireNotifications(cfg core.Config) {
	notify.Init()
	notify.SetSoundEnabled(cfg.NotificationSound)
	notify.SetNotificationsEnabled(cfg.NotificationsStatus == core.NotificationGranted)

	c.expiration = notify.NewExpirationMonitor(notify.ExpirationConfig{
		Thresholds:                cfg.ExpirationThresholds,
		ProviderExpirationMinutes: cfg.ProviderExpirationMinutes,
	}, func(name string, mins int) {
		if !c.nativeNotifications() {
			return
		}
		if mins == 0 {
			notify.NotifyTunnelExpired(name)
		} else {
			notify.NotifyTunnelExpiring(name, mins)
		}
	})

	c.manager.SetNotificationHandler(func(status core.TunnelStatus) {
		if !c.nativeNotifications() {
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

	c.manager.SetVisitorHandler(func(status core.TunnelStatus) {
		if c.web != nil {
			c.web.BroadcastNewVisitor(status)
		}
		if !c.nativeNotifications() {
			return
		}
		notify.NotifyNewVisitor(status.Name, status.Visitors)
	})

	c.manager.SetExpirationCallbacks(
		c.expiration.Start,
		func(id string) { c.expiration.Stop(id) },
	)
}

func (c *appCore) nativeNotifications() bool {
	return c.web == nil || c.web.ClientCount() == 0
}

func (c *appCore) url() string { return c.web.URL() }

func (c *appCore) shutdown() {
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
