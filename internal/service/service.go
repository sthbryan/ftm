// Package service holds ftm's use cases. It is the only code that writes the
// store and the only code that starts or stops tunnels; the web handlers, the
// TUI and the desktop shell call it and learn about every change, including
// their own, from the events it and the process manager publish.
package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/platform/notify"
	"github.com/sthbryan/ftm/internal/store"
	"github.com/sthbryan/ftm/internal/updater"
	"github.com/sthbryan/ftm/internal/version"
)

var (
	// ErrTunnelNotFound is returned for an id that is not saved.
	ErrTunnelNotFound = errors.New("tunnel not found")
	// ErrInvalidTunnel is returned for a connection without a name or provider,
	// or with a port outside 1-65535.
	ErrInvalidTunnel = errors.New("invalid tunnel")
)

// Supervisor is what the service needs from the process manager.
type Supervisor interface {
	Start(core.TunnelConfig) error
	Stop(tunnelID string) error
	StopAll()
	CheckInstallation(core.Provider) (needsInstall, autoInstall bool)
	InstallProvider(core.Provider) error
}

// statsRunner is implemented by a supervisor that can publish visitor stats.
type statsRunner interface {
	RunStats(ctx context.Context, interval time.Duration)
}

const (
	statsInterval       = 5 * time.Second
	updateCheckInterval = 6 * time.Hour
	updateRepo          = "sthbryan/ftm"
)

// Service runs the use cases against one store, one supervisor and one bus.
type Service struct {
	store *store.Store
	sup   Supervisor
	bus   events.Publisher

	installMu sync.Mutex
	installs  map[core.Provider]*install

	updateMu    sync.RWMutex
	updateInfo  *updater.Info
	checkUpdate func() (*updater.Info, error)
	applyUpdate func(*updater.Info) error

	providerMu sync.Mutex
	announced  map[core.Provider]string

	runMu  sync.Mutex
	runCtx context.Context

	restart     chan struct{}
	restartOnce sync.Once

	// applySettings pushes saved preferences into the packages that keep
	// their own copy (language, notifications). Swapped out in tests.
	applySettings func(core.Config)
}

// New wires a service. bus may be nil, in which case nothing is published.
func New(st *store.Store, sup Supervisor, bus events.Publisher) *Service {
	return &Service{
		store:     st,
		sup:       sup,
		bus:       bus,
		installs:  make(map[core.Provider]*install),
		announced: make(map[core.Provider]string),
		restart:   make(chan struct{}),
		checkUpdate: func() (*updater.Info, error) {
			return updater.New(updateRepo).Check(version.Version)
		},
		applyUpdate: func(info *updater.Info) error {
			return updater.New(updateRepo).Apply(info)
		},
		applySettings: applySettings,
	}
}

func (s *Service) publish(e events.Event) {
	if s.bus != nil {
		s.bus.Publish(e)
	}
}

// background is the context of the work Run started, or a context that is
// never done before Run was called.
func (s *Service) background() context.Context {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if s.runCtx == nil {
		return context.Background()
	}
	return s.runCtx
}

// Run starts the background work: visitor stats, the periodic ftm update
// check and the provider binary check. It returns at once, before any check
// answers; the work stops when ctx is done.
func (s *Service) Run(ctx context.Context) {
	s.runMu.Lock()
	s.runCtx = ctx
	s.runMu.Unlock()

	go s.runProviderChecks(ctx)

	if r, ok := s.sup.(statsRunner); ok {
		go r.RunStats(ctx, statsInterval)
	}

	go func() {
		if _, err := s.CheckForUpdate(); err != nil {
			log.Printf("update check failed: %v", err)
		}

		t := time.NewTicker(updateCheckInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if _, err := s.CheckForUpdate(); err != nil {
					log.Printf("update check failed: %v", err)
				}
			}
		}
	}()
}

// Config returns a copy of the current config.
func (s *Service) Config() core.Config { return s.store.Snapshot() }

// Dirs returns the app directories.
func (s *Service) Dirs() store.Dirs { return s.store.Dirs() }

// Tunnels returns the saved connections.
func (s *Service) Tunnels() []core.TunnelConfig { return s.store.Tunnels() }

// Tunnel returns the saved connection with id.
func (s *Service) Tunnel(id string) (core.TunnelConfig, bool) { return s.store.Tunnel(id) }

// TunnelInput is a new connection.
type TunnelInput struct {
	Name      string
	Provider  core.Provider
	LocalPort int
}

// TunnelPatch changes the fields that are set.
type TunnelPatch struct {
	Name      *string
	Provider  *core.Provider
	LocalPort *int
}

func validPort(p int) bool { return p >= 1 && p <= 65535 }

// CreateTunnel saves a new connection and publishes events.TunnelSaved.
func (s *Service) CreateTunnel(in TunnelInput) (core.TunnelConfig, error) {
	if in.Name == "" || in.Provider == "" || !validPort(in.LocalPort) {
		return core.TunnelConfig{}, ErrInvalidTunnel
	}

	var tunnel core.TunnelConfig
	err := s.store.Update(func(c *core.Config) error {
		tunnel = core.TunnelConfig{
			ID:        c.NewTunnelID(in.Name),
			Name:      in.Name,
			Provider:  in.Provider,
			LocalPort: in.LocalPort,
		}
		c.AddTunnel(tunnel)
		return nil
	})
	if err != nil {
		return core.TunnelConfig{}, err
	}

	s.publish(events.TunnelSaved{Tunnel: tunnel, Created: true})
	return tunnel, nil
}

// UpdateTunnel edits a saved connection and publishes events.TunnelSaved.
// A running tunnel keeps its old settings until it is restarted.
func (s *Service) UpdateTunnel(id string, patch TunnelPatch) (core.TunnelConfig, error) {
	if patch.Name != nil && *patch.Name == "" ||
		patch.Provider != nil && *patch.Provider == "" ||
		patch.LocalPort != nil && !validPort(*patch.LocalPort) {
		return core.TunnelConfig{}, ErrInvalidTunnel
	}

	var tunnel core.TunnelConfig
	err := s.store.Update(func(c *core.Config) error {
		t := c.GetTunnel(id)
		if t == nil {
			return ErrTunnelNotFound
		}
		if patch.Name != nil {
			t.Name = *patch.Name
		}
		if patch.Provider != nil {
			t.Provider = *patch.Provider
		}
		if patch.LocalPort != nil {
			t.LocalPort = *patch.LocalPort
		}
		tunnel = *t
		return nil
	})
	if err != nil {
		return core.TunnelConfig{}, err
	}

	s.publish(events.TunnelSaved{Tunnel: tunnel})
	return tunnel, nil
}

// DeleteTunnel stops a connection if it runs, removes it and publishes
// events.TunnelDeleted. Deleting an id that is not saved is not an error.
func (s *Service) DeleteTunnel(id string) error {
	// Deleting a tunnel that is not running is the common case.
	_ = s.sup.Stop(id)

	if err := s.store.Update(func(c *core.Config) error {
		c.RemoveTunnel(id)
		return nil
	}); err != nil {
		return err
	}

	s.publish(events.TunnelDeleted{ID: id})
	return nil
}

// ClearTunnels stops and removes every saved connection, publishing
// events.TunnelDeleted for each. It returns how many were removed.
func (s *Service) ClearTunnels() (int, error) {
	tunnels := s.store.Tunnels()
	for _, t := range tunnels {
		_ = s.sup.Stop(t.ID)
	}

	var deleted int
	if err := s.store.Update(func(c *core.Config) error {
		deleted = c.ClearTunnels()
		return nil
	}); err != nil {
		return 0, err
	}

	for _, t := range tunnels {
		s.publish(events.TunnelDeleted{ID: t.ID})
	}
	return deleted, nil
}

// StartTunnel starts a saved connection. When its provider has to be
// downloaded first it publishes the installing state, returns installing=true
// at once and starts the tunnel once the download finishes. Failures after
// that point are published as an error state.
func (s *Service) StartTunnel(id string) (installing bool, err error) {
	tunnel, ok := s.store.Tunnel(id)
	if !ok {
		return false, ErrTunnelNotFound
	}

	if needsInstall, canInstall := s.sup.CheckInstallation(tunnel.Provider); needsInstall && canInstall {
		status := tunnel.Status()
		status.State = core.TunnelStateInstall
		s.publish(events.TunnelStatus{Status: status})

		go s.installThenStart(tunnel)
		return true, nil
	}

	return false, s.sup.Start(tunnel)
}

// install is one provider download that several starts may wait on.
type install struct {
	done chan struct{}
	err  error
}

// ensureInstalled downloads a provider once, however many tunnels ask for it
// at the same time.
func (s *Service) ensureInstalled(provider core.Provider) error {
	s.installMu.Lock()
	job, running := s.installs[provider]
	if !running {
		job = &install{done: make(chan struct{})}
		s.installs[provider] = job
	}
	s.installMu.Unlock()

	if running {
		<-job.done
		return job.err
	}

	job.err = s.sup.InstallProvider(provider)

	s.installMu.Lock()
	delete(s.installs, provider)
	s.installMu.Unlock()
	close(job.done)

	return job.err
}

func (s *Service) installThenStart(tunnel core.TunnelConfig) {
	if err := s.ensureInstalled(tunnel.Provider); err != nil {
		s.publishFailure(tunnel, i18n.TF("error_install_failed", err.Error()))
		return
	}

	// The connection may have been edited or deleted during the download.
	current, ok := s.store.Tunnel(tunnel.ID)
	if !ok {
		return
	}
	if err := s.sup.Start(current); err != nil {
		s.publishFailure(current, err.Error())
	}
}

func (s *Service) publishFailure(tunnel core.TunnelConfig, message string) {
	status := tunnel.Status()
	status.State = core.TunnelStateError
	status.ErrorMessage = message
	s.publish(events.TunnelStatus{Status: status})
}

// StopTunnel stops a running connection.
func (s *Service) StopTunnel(id string) error {
	return s.sup.Stop(id)
}

// StopAll stops every running connection.
func (s *Service) StopAll() {
	s.sup.StopAll()
}

// SettingsPatch changes the preferences that are set.
type SettingsPatch struct {
	NotificationsStatus *string
	NotificationSound   *bool
	Language            *string
}

// UpdateSettings saves preferences, applies them and publishes
// events.SettingsChanged.
func (s *Service) UpdateSettings(patch SettingsPatch) (core.Config, error) {
	err := s.store.Update(func(c *core.Config) error {
		if patch.NotificationsStatus != nil {
			c.NotificationsStatus = *patch.NotificationsStatus
		}
		if patch.NotificationSound != nil {
			c.NotificationSound = *patch.NotificationSound
		}
		if patch.Language != nil {
			c.Language = *patch.Language
		}
		return nil
	})
	cfg := s.store.Snapshot()
	if err != nil {
		return cfg, err
	}

	if s.applySettings != nil {
		s.applySettings(cfg)
	}
	s.publish(events.SettingsChanged{Config: cfg})
	return cfg, nil
}

func applySettings(cfg core.Config) {
	i18n.ChangeLanguage(i18n.ResolveLanguage(cfg.Language))
	notify.SetNotificationsEnabled(cfg.NotificationsStatus == core.NotificationGranted)
	notify.SetSoundEnabled(cfg.NotificationSound)
}

// SetWebPort makes the web server prefer port the next time it starts.
func (s *Service) SetWebPort(port int) error {
	if port != 0 && !validPort(port) {
		return fmt.Errorf("invalid web port %d", port)
	}
	return s.store.Update(func(c *core.Config) error {
		c.WebPort = port
		return nil
	})
}

// CheckForUpdate asks GitHub for a newer ftm release and remembers the
// answer. It publishes events.UpdateAvailable the first time one is found.
func (s *Service) CheckForUpdate() (*updater.Info, error) {
	info, err := s.checkUpdate()
	if err != nil {
		return nil, err
	}

	s.updateMu.Lock()
	hadUpdate := s.updateInfo != nil && s.updateInfo.HasUpdate
	s.updateInfo = info
	s.updateMu.Unlock()

	if info != nil && info.HasUpdate && !hadUpdate {
		s.publish(events.UpdateAvailable{
			Current:    version.Version,
			Latest:     info.LatestVersion,
			Tag:        info.Tag,
			AssetName:  info.AssetName,
			ReleaseURL: info.ReleaseURL,
			Method:     string(info.Method),
		})
	}
	return info, nil
}

// UpdateInfo returns the last update check, or nil before the first one.
func (s *Service) UpdateInfo() *updater.Info {
	s.updateMu.RLock()
	defer s.updateMu.RUnlock()
	return s.updateInfo
}

// ApplyUpdate installs the update the last check found. It does nothing when
// there is none.
func (s *Service) ApplyUpdate() error {
	info := s.UpdateInfo()
	if info == nil || !info.HasUpdate {
		return nil
	}
	return s.applyUpdate(info)
}
