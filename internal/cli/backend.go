package cli

import (
	"context"
	"errors"
	"strings"

	"github.com/sthbryan/ftm/internal/client"
	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/service"
)

// Tunnel, TunnelInput and the other shapes commands work with are the API's,
// so a command reads the same whether it runs against another process or the
// core in this one.
type (
	Tunnel         = client.Tunnel
	TunnelInput    = client.TunnelInput
	ProviderBinary = client.ProviderBinary
	Settings       = client.Settings
	SettingsPatch  = client.SettingsPatch
)

// Backend is the ftm a command acts on: a running instance over its API, or
// the core run inside this process.
type Backend interface {
	// Attached is true when another process owns the tunnels.
	Attached() bool
	Tunnels(ctx context.Context) ([]Tunnel, error)
	CreateTunnel(ctx context.Context, in TunnelInput) (Tunnel, error)
	UpdateTunnel(ctx context.Context, id string, in TunnelInput) (Tunnel, error)
	DeleteTunnel(ctx context.Context, id string) error
	StartTunnel(ctx context.Context, id string) error
	StopTunnel(ctx context.Context, id string) error
	Logs(ctx context.Context, id string) ([]string, error)
	// FollowLogs calls fn with the recent lines, then with each new one, until
	// ctx ends.
	FollowLogs(ctx context.Context, id string, fn func(line string)) error
	// WaitTunnel calls done with the tunnel's state now and after every
	// change until done returns true, and returns that state.
	WaitTunnel(ctx context.Context, id string, done func(Tunnel) bool) (Tunnel, error)
	Providers(ctx context.Context, refresh bool) ([]ProviderBinary, error)
	// UpdateProvider downloads ftm's copy of a provider and waits for it to
	// finish, reporting progress in percent. It returns the new version.
	UpdateProvider(ctx context.Context, id string, progress func(percent float64)) (string, error)
	Settings(ctx context.Context) (Settings, error)
	UpdateSettings(ctx context.Context, patch SettingsPatch) (Settings, error)
}

// errUpdateFailed carries the reason a provider update reported.
type errUpdateFailed struct{ reason string }

func (e errUpdateFailed) Error() string { return e.reason }

// remote is a Backend over a running instance's API.
type remote struct{ c *client.Client }

// Remote returns a Backend for a running instance.
func Remote(c *client.Client) Backend { return remote{c: c} }

func (r remote) Attached() bool { return true }

func (r remote) Tunnels(ctx context.Context) ([]Tunnel, error) { return r.c.Tunnels(ctx) }

func (r remote) CreateTunnel(ctx context.Context, in TunnelInput) (Tunnel, error) {
	return r.c.CreateTunnel(ctx, in)
}

func (r remote) UpdateTunnel(ctx context.Context, id string, in TunnelInput) (Tunnel, error) {
	return r.c.UpdateTunnel(ctx, id, in)
}

func (r remote) DeleteTunnel(ctx context.Context, id string) error { return r.c.DeleteTunnel(ctx, id) }

func (r remote) StartTunnel(ctx context.Context, id string) error {
	_, err := r.c.StartTunnel(ctx, id)
	return err
}

func (r remote) StopTunnel(ctx context.Context, id string) error { return r.c.StopTunnel(ctx, id) }

func (r remote) Logs(ctx context.Context, id string) ([]string, error) { return r.c.Logs(ctx, id) }

func (r remote) FollowLogs(ctx context.Context, id string, fn func(string)) error {
	return r.c.FollowLogs(ctx, id, func(e client.Event) bool {
		switch e.Name {
		case "log_snapshot":
			var snap struct {
				Lines []string `json:"lines"`
			}
			if e.Decode(&snap) == nil {
				for _, l := range snap.Lines {
					fn(l)
				}
			}
		case "log":
			var msg struct {
				Line string `json:"line"`
			}
			if e.Decode(&msg) == nil {
				fn(msg.Line)
			}
		}
		return true
	})
}

func (r remote) WaitTunnel(ctx context.Context, id string, done func(Tunnel) bool) (Tunnel, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var last Tunnel
	var finished bool
	var callErr error
	err := r.c.Events(ctx, func(e client.Event) bool {
		switch e.Name {
		case client.EventOpen, "resync":
			last, callErr = r.c.Tunnel(ctx, id)
			if callErr != nil {
				return false
			}
		case "tunnel_state":
			var t Tunnel
			if e.Decode(&t) != nil || t.ID != id {
				return true
			}
			last = merge(last, t)
		case "tunnel_deleted":
			var t struct {
				ID string `json:"id"`
			}
			if e.Decode(&t) == nil && t.ID == id {
				callErr = service.ErrTunnelNotFound
				return false
			}
			return true
		default:
			return true
		}
		finished = done(last)
		return !finished
	})
	if callErr != nil {
		return last, callErr
	}
	if finished {
		return last, nil
	}
	return last, err
}

// merge applies a state event to what is known: the short-lived states such
// as installing only carry the id and the state.
func merge(prev, next Tunnel) Tunnel {
	if next.Name == "" {
		prev.State = next.State
		if next.ErrorMessage != "" {
			prev.ErrorMessage = next.ErrorMessage
		}
		return prev
	}
	return next
}

func (r remote) Providers(ctx context.Context, refresh bool) ([]ProviderBinary, error) {
	return r.c.Providers(ctx, refresh)
}

func (r remote) UpdateProvider(ctx context.Context, id string, progress func(float64)) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var version string
	var result, callErr error
	var finished bool
	err := r.c.Events(ctx, func(e client.Event) bool {
		switch e.Name {
		case client.EventOpen:
			if callErr = r.c.UpdateProvider(ctx, id); callErr != nil {
				return false
			}
		case "install":
			var msg struct {
				Install struct {
					Provider string  `json:"provider"`
					Percent  float64 `json:"percent"`
				} `json:"install"`
			}
			if e.Decode(&msg) == nil && msg.Install.Provider == id && progress != nil {
				progress(msg.Install.Percent)
			}
		case "provider_updated":
			var msg struct {
				Provider string `json:"provider"`
				OK       bool   `json:"ok"`
				Version  string `json:"version"`
				Error    string `json:"error"`
			}
			if e.Decode(&msg) != nil || msg.Provider != id {
				return true
			}
			finished = true
			version = msg.Version
			if !msg.OK {
				result = errUpdateFailed{msg.Error}
			}
			return false
		}
		return true
	})
	switch {
	case callErr != nil:
		return "", providerError(callErr)
	case finished:
		return version, result
	}
	return "", err
}

// providerError turns the API's provider errors back into the core's, so
// commands print the same message either way.
func providerError(err error) error {
	var apiErr *client.Error
	if !errors.As(err, &apiErr) {
		return err
	}
	switch apiErr.Code {
	case "unknown_provider":
		return core.ErrUnknownProvider
	case "not_updatable":
		return core.ErrNotUpdatable
	case "external_binary":
		return core.ErrExternalBinary
	case "update_in_progress":
		return core.ErrUpdateInProgress
	}
	return err
}

func (r remote) Settings(ctx context.Context) (Settings, error) { return r.c.Settings(ctx) }

func (r remote) UpdateSettings(ctx context.Context, patch SettingsPatch) (Settings, error) {
	return r.c.UpdateSettings(ctx, patch)
}

// Autostart turns ftm-desktop's start at login on and off. The command line
// only reports and switches it; the desktop app registers itself.
type Autostart interface {
	Supported() bool
	Enabled() (bool, error)
	Enable() error
	Disable() error
}

// local is a Backend over the core running in this process.
type local struct {
	svc       *service.Service
	bus       *events.Bus
	autostart Autostart
}

// Local returns a Backend over a core this process runs.
func Local(svc *service.Service, bus *events.Bus, autostart Autostart) Backend {
	return local{svc: svc, bus: bus, autostart: autostart}
}

func (l local) Attached() bool { return false }

func tunnelFromView(v service.TunnelView) Tunnel {
	t := Tunnel{
		ID:       v.Tunnel.ID,
		Name:     v.Tunnel.Name,
		Provider: string(v.Tunnel.Provider),
		Port:     v.Tunnel.LocalPort,
		State:    string(core.TunnelStateStopped),
	}
	if v.Running {
		t.State = string(v.Status.State)
		t.PublicURL = v.Status.PublicURL
		t.ErrorMessage = v.Status.ErrorMessage
		t.ExpiresAt = v.Status.ExpiresAt
		t.StartedAt = v.Status.StartedAt
		t.Visitors = v.Status.Visitors
		t.ActiveSessions = v.Status.ActiveSessions
	}
	if t.State == string(core.TunnelStateStopped) && v.NeedsInstall {
		t.State = stateNeedsInstall
	}
	return t
}

func (l local) Tunnels(context.Context) ([]Tunnel, error) {
	views := l.svc.TunnelViews()
	out := make([]Tunnel, 0, len(views))
	for _, v := range views {
		out = append(out, tunnelFromView(v))
	}
	return out, nil
}

func (l local) tunnel(id string) (Tunnel, error) {
	v, ok := l.svc.TunnelView(id)
	if !ok {
		return Tunnel{}, service.ErrTunnelNotFound
	}
	return tunnelFromView(v), nil
}

func (l local) CreateTunnel(_ context.Context, in TunnelInput) (Tunnel, error) {
	t, err := l.svc.CreateTunnel(service.TunnelInput{
		Name:      strings.TrimSpace(in.Name),
		Provider:  core.Provider(in.Provider),
		LocalPort: in.LocalPort,
	})
	if err != nil {
		return Tunnel{}, err
	}
	return tunnelFromView(l.svc.View(t)), nil
}

func (l local) UpdateTunnel(_ context.Context, id string, in TunnelInput) (Tunnel, error) {
	var patch service.TunnelPatch
	if name := strings.TrimSpace(in.Name); name != "" {
		patch.Name = &name
	}
	if in.Provider != "" {
		p := core.Provider(in.Provider)
		patch.Provider = &p
	}
	if in.LocalPort >= 1 && in.LocalPort <= 65535 {
		port := in.LocalPort
		patch.LocalPort = &port
	}
	t, err := l.svc.UpdateTunnel(id, patch)
	if err != nil {
		return Tunnel{}, err
	}
	return tunnelFromView(l.svc.View(t)), nil
}

func (l local) DeleteTunnel(_ context.Context, id string) error { return l.svc.DeleteTunnel(id) }

func (l local) StartTunnel(_ context.Context, id string) error {
	_, err := l.svc.StartTunnel(id)
	return err
}

func (l local) StopTunnel(_ context.Context, id string) error { return l.svc.StopTunnel(id) }

func (l local) Logs(_ context.Context, id string) ([]string, error) {
	if _, ok := l.svc.Tunnel(id); !ok {
		return nil, service.ErrTunnelNotFound
	}
	return l.svc.Logs(id), nil
}

func (l local) FollowLogs(ctx context.Context, id string, fn func(string)) error {
	sub, stop := l.bus.Subscribe(events.Only(events.KindLogLine))
	defer stop()
	lines, err := l.Logs(ctx, id)
	if err != nil {
		return err
	}
	for _, line := range lines {
		fn(line)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e, ok := <-sub:
			if !ok {
				return nil
			}
			if line, ok := e.(events.LogLine); ok && line.TunnelID == id {
				fn(line.Line)
			}
		}
	}
}

func (l local) WaitTunnel(ctx context.Context, id string, done func(Tunnel) bool) (Tunnel, error) {
	sub, stop := l.bus.Subscribe(events.Only(events.KindTunnelStatus, events.KindTunnelDeleted))
	defer stop()

	last, err := l.tunnel(id)
	if err != nil {
		return last, err
	}
	if done(last) {
		return last, nil
	}
	for {
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case e, ok := <-sub:
			if !ok {
				return last, context.Canceled
			}
			switch e := e.(type) {
			case events.TunnelDeleted:
				if e.ID == id {
					return last, service.ErrTunnelNotFound
				}
				continue
			case events.TunnelStatus:
				if e.Status.ID != id {
					continue
				}
				if t, err := l.tunnel(id); err == nil {
					last = t
				}
				last.State = string(e.Status.State)
				if e.Status.ErrorMessage != "" {
					last.ErrorMessage = e.Status.ErrorMessage
				}
				if e.Status.PublicURL != "" {
					last.PublicURL = e.Status.PublicURL
				}
			}
			if done(last) {
				return last, nil
			}
		}
	}
}

func (l local) Providers(ctx context.Context, refresh bool) ([]ProviderBinary, error) {
	report := l.svc.ProviderBinaries(ctx, refresh)
	out := make([]ProviderBinary, 0, len(report))
	for _, b := range report {
		out = append(out, providerFromService(b))
	}
	return out, nil
}

func providerFromService(b service.ProviderBinary) ProviderBinary {
	return ProviderBinary{
		ID:         b.ID,
		Name:       b.Name,
		Installed:  b.Installed,
		Version:    b.Version,
		Latest:     b.Latest,
		Source:     b.Source,
		Path:       b.Path,
		HasUpdate:  b.HasUpdate,
		Updatable:  b.Updatable,
		Updating:   b.Updating,
		ReleaseURL: b.ReleaseURL,
		Error:      b.Error,
	}
}

func (l local) UpdateProvider(ctx context.Context, id string, progress func(float64)) (string, error) {
	sub, stop := l.bus.Subscribe(events.Only(events.KindInstallProgress, events.KindProviderUpdated))
	defer stop()

	if err := l.svc.UpdateProvider(ctx, core.Provider(id)); err != nil {
		return "", err
	}
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case e, ok := <-sub:
			if !ok {
				return "", context.Canceled
			}
			switch e := e.(type) {
			case events.InstallProgress:
				if e.Provider == id && progress != nil {
					progress(e.Percent)
				}
			case events.ProviderUpdated:
				if e.Provider != id {
					continue
				}
				if !e.OK {
					return "", errUpdateFailed{e.Error}
				}
				return e.Version, nil
			}
		}
	}
}

func (l local) Settings(context.Context) (Settings, error) {
	cfg := l.svc.Config()
	s := Settings{
		NotificationsEnabled: cfg.NotificationsStatus,
		NotificationSound:    cfg.NotificationSound,
		Language:             cfg.Language,
	}
	if l.autostart != nil && l.autostart.Supported() {
		s.AutostartSupported = true
		s.AutostartEnabled, _ = l.autostart.Enabled()
	}
	return s, nil
}

func (l local) UpdateSettings(ctx context.Context, patch SettingsPatch) (Settings, error) {
	if patch.Autostart != nil {
		if l.autostart == nil || !l.autostart.Supported() {
			return Settings{}, errAutostartUnsupported
		}
		var err error
		if *patch.Autostart {
			err = l.autostart.Enable()
		} else {
			err = l.autostart.Disable()
		}
		if err != nil {
			return Settings{}, err
		}
	}
	if _, err := l.svc.UpdateSettings(service.SettingsPatch{
		NotificationsStatus: patch.NotificationsEnabled,
		NotificationSound:   patch.NotificationSound,
		Language:            patch.Language,
	}); err != nil {
		return Settings{}, err
	}
	return l.Settings(ctx)
}

// stateNeedsInstall is the state of a stopped tunnel whose provider has to
// be downloaded first.
const stateNeedsInstall = "need_installing"
