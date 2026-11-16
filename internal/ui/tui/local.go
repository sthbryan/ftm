package tui

import (
	"context"
	"errors"
	"sync/atomic"

	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/provider"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/updater"
)

// Dashboard is the web server of the core run in this process.
type Dashboard interface {
	URL() string
	ClientCount() int
}

// Local is the core run in this process: changes go through Service and
// updates arrive on Bus.
type Local struct {
	Service *service.Service
	Bus     *events.Bus
	Web     Dashboard

	watched atomic.Pointer[string]
}

// NewLocal returns a Backend over the core run in this process.
func NewLocal(svc *service.Service, bus *events.Bus, web Dashboard) *Local {
	return &Local{Service: svc, Bus: bus, Web: web}
}

func (l *Local) Attached() bool { return false }

func (l *Local) DashboardURL() string {
	if l.Web == nil {
		return ""
	}
	return l.Web.URL()
}

func (l *Local) Sessions() int {
	if l.Web == nil {
		return -1
	}
	return l.Web.ClientCount()
}

func (l *Local) ConfigDir() string { return l.Service.Dirs().Config }

// RestartRequested is closed when an update applied from the web view asks
// this process to start again.
func (l *Local) RestartRequested() <-chan struct{} { return l.Service.RestartRequested() }

func (l *Local) Tunnels(context.Context) ([]TunnelItem, error) {
	tunnels := l.Service.Tunnels()
	items := make([]TunnelItem, 0, len(tunnels))
	for _, t := range tunnels {
		status := t.Status()
		if s, ok := l.Service.Status(t.ID); ok {
			status = s
		}
		items = append(items, TunnelItem{Tunnel: t, Status: status, Running: l.Service.Running(t.ID)})
	}
	return items, nil
}

func (l *Local) Prefs(context.Context) (Prefs, error) {
	cfg := l.Service.Config()
	return Prefs{
		Notifications: cfg.NotificationsStatus == core.NotificationGranted,
		Sound:         cfg.NotificationSound,
		Language:      cfg.Language,
		Onboarded:     cfg.Onboarded,
	}, nil
}

func (l *Local) SavePrefs(_ context.Context, p Prefs) error {
	status := core.NotificationRejected
	if p.Notifications {
		status = core.NotificationGranted
	}
	_, err := l.Service.UpdateSettings(service.SettingsPatch{
		NotificationsStatus: &status,
		NotificationSound:   &p.Sound,
		Language:            &p.Language,
	})
	return err
}

func (l *Local) CreateTunnel(_ context.Context, in TunnelInput) error {
	_, err := l.Service.CreateTunnel(service.TunnelInput{
		Name:      in.Name,
		Provider:  in.Provider,
		LocalPort: in.LocalPort,
	})
	return err
}

func (l *Local) UpdateTunnel(_ context.Context, id string, in TunnelInput) error {
	_, err := l.Service.UpdateTunnel(id, service.TunnelPatch{
		Name:      &in.Name,
		Provider:  &in.Provider,
		LocalPort: &in.LocalPort,
	})
	return err
}

func (l *Local) DeleteTunnel(_ context.Context, id string) error { return l.Service.DeleteTunnel(id) }

func (l *Local) ClearTunnels(context.Context) error {
	_, err := l.Service.ClearTunnels()
	return err
}

func (l *Local) StartTunnel(_ context.Context, id string) (bool, error) {
	return l.Service.StartTunnel(id)
}

func (l *Local) StopTunnel(_ context.Context, id string) error { return l.Service.StopTunnel(id) }

func (l *Local) Logs(_ context.Context, id string) ([]string, error) {
	return l.Service.Logs(id), nil
}

func (l *Local) ProviderBinaries(ctx context.Context) ([]service.ProviderBinary, error) {
	return l.Service.ProviderBinaries(ctx, false), nil
}

func (l *Local) UpdateInfo(context.Context) (*updater.Info, error) {
	return l.Service.UpdateInfo(), nil
}

func (l *Local) ApplyUpdate(context.Context) error { return l.Service.ApplyUpdate() }

func (l *Local) WatchLogs(id string) { l.watched.Store(&id) }

func (l *Local) watching(id string) bool {
	w := l.watched.Load()
	return w != nil && *w == id
}

// Events subscribes to the bus and turns each event into a model message.
func (l *Local) Events(ctx context.Context) <-chan tea.Msg {
	out := make(chan tea.Msg)
	if l.Bus == nil {
		close(out)
		return out
	}
	in, cancel := l.Bus.Subscribe()
	go func() {
		defer close(out)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			case e, ok := <-in:
				if !ok {
					return
				}
				msg := l.message(e)
				if msg == nil {
					continue
				}
				select {
				case out <- msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}

func (l *Local) message(e events.Event) tea.Msg {
	switch e := e.(type) {
	case events.TunnelStatus:
		return statusUpdateMsg{tunnelID: e.Status.ID, status: e.Status}
	case events.TunnelSaved, events.TunnelDeleted, events.TunnelStats:
		return tunnelsChangedMsg{}
	case events.SettingsChanged:
		return prefsChangedMsg{}
	case events.InstallProgress:
		p := provider.DownloadProgress{
			Name:    e.Provider,
			Percent: e.Percent,
			Current: e.Current,
			Total:   e.Total,
			Done:    e.Done,
		}
		if e.Err != "" {
			p.Error = errors.New(e.Err)
		}
		return downloadProgressMsg(p)
	case events.UpdateAvailable:
		return updateCheckMsg{info: &updater.Info{
			CurrentVersion: e.Current,
			LatestVersion:  e.Latest,
			Tag:            e.Tag,
			AssetName:      e.AssetName,
			ReleaseURL:     e.ReleaseURL,
			HasUpdate:      true,
			Method:         updater.Method(e.Method),
		}}
	case events.LogLine:
		if l.watching(e.TunnelID) {
			return logLineMsg{tunnelID: e.TunnelID, line: e.Line}
		}
	}
	return nil
}
