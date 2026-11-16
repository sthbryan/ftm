package tui

import (
	"context"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/client"
	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/provider"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/updater"
)

// Retry delays after the stream to the running ftm drops.
const (
	reconnectMin = 250 * time.Millisecond
	reconnectMax = 5 * time.Second
)

// Remote is a running ftm (the desktop app or `ftm serve`) driven over its
// API. Changes go through the REST endpoints and updates arrive on its
// Server-Sent Events stream, the same one the web view listens to.
type Remote struct {
	configDir string
	// find locates the running ftm again after the stream drops, since a
	// restarted instance has a new port and token.
	find func() (*client.Client, error)

	mu        sync.Mutex
	c         *client.Client
	out       chan tea.Msg
	ctx       context.Context
	watched   string
	stopWatch context.CancelFunc
}

// NewRemote returns a Backend over the running ftm c talks to. find, when not
// nil, locates it again after a restart.
func NewRemote(c *client.Client, configDir string, find func() (*client.Client, error)) *Remote {
	return &Remote{c: c, configDir: configDir, find: find}
}

func (r *Remote) client() *client.Client {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.c
}

func (r *Remote) Attached() bool       { return true }
func (r *Remote) DashboardURL() string { return r.client().URL() }
func (r *Remote) Sessions() int        { return -1 }
func (r *Remote) ConfigDir() string    { return r.configDir }

func (r *Remote) Tunnels(ctx context.Context) ([]TunnelItem, error) {
	tunnels, err := r.client().Tunnels(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]TunnelItem, 0, len(tunnels))
	for _, t := range tunnels {
		items = append(items, remoteItem(t))
	}
	return items, nil
}

func remoteItem(t client.Tunnel) TunnelItem {
	status := remoteStatus(t)
	return TunnelItem{
		Tunnel: core.TunnelConfig{
			ID:        t.ID,
			Name:      t.Name,
			Provider:  core.Provider(t.Provider),
			LocalPort: t.Port,
		},
		Status:  status,
		Running: running(status.State),
	}
}

func remoteStatus(t client.Tunnel) core.TunnelStatus {
	state := core.TunnelState(t.State)
	if state == "need_installing" {
		state = core.TunnelStateStopped
	}
	return core.TunnelStatus{
		ID:             t.ID,
		Name:           t.Name,
		Provider:       core.Provider(t.Provider),
		LocalPort:      t.Port,
		PublicURL:      t.PublicURL,
		State:          state,
		ErrorMessage:   t.ErrorMessage,
		Visitors:       t.Visitors,
		ActiveSessions: t.ActiveSessions,
		ExpiresAt:      t.ExpiresAt,
		StartedAt:      t.StartedAt,
	}
}

// running matches service.Running: starting, connecting or online.
func running(s core.TunnelState) bool {
	switch s {
	case core.TunnelStateStarting, core.TunnelStateConnecting, core.TunnelStateOnline:
		return true
	}
	return false
}

func (r *Remote) Prefs(ctx context.Context) (Prefs, error) {
	s, err := r.client().Settings(ctx)
	if err != nil {
		return Prefs{}, err
	}
	return Prefs{
		Notifications: s.NotificationsEnabled == core.NotificationGranted,
		Sound:         s.NotificationSound,
		Language:      s.Language,
		Onboarded:     s.Onboarded,
	}, nil
}

func (r *Remote) SavePrefs(ctx context.Context, p Prefs) error {
	status := core.NotificationRejected
	if p.Notifications {
		status = core.NotificationGranted
	}
	_, err := r.client().UpdateSettings(ctx, client.SettingsPatch{
		NotificationsEnabled: &status,
		NotificationSound:    &p.Sound,
		Language:             &p.Language,
	})
	return err
}

func (r *Remote) CreateTunnel(ctx context.Context, in TunnelInput) error {
	_, err := r.client().CreateTunnel(ctx, client.TunnelInput{
		Name:      in.Name,
		Provider:  string(in.Provider),
		LocalPort: in.LocalPort,
	})
	return err
}

func (r *Remote) UpdateTunnel(ctx context.Context, id string, in TunnelInput) error {
	_, err := r.client().UpdateTunnel(ctx, id, client.TunnelInput{
		Name:      in.Name,
		Provider:  string(in.Provider),
		LocalPort: in.LocalPort,
	})
	return err
}

func (r *Remote) DeleteTunnel(ctx context.Context, id string) error {
	return r.client().DeleteTunnel(ctx, id)
}

func (r *Remote) ClearTunnels(ctx context.Context) error { return r.client().ClearTunnels(ctx) }

func (r *Remote) StartTunnel(ctx context.Context, id string) (bool, error) {
	t, err := r.client().StartTunnel(ctx, id)
	if err != nil {
		return false, err
	}
	return core.TunnelState(t.State) == core.TunnelStateInstall, nil
}

func (r *Remote) StopTunnel(ctx context.Context, id string) error {
	return r.client().StopTunnel(ctx, id)
}

func (r *Remote) Logs(ctx context.Context, id string) ([]string, error) {
	return r.client().Logs(ctx, id)
}

func (r *Remote) ProviderBinaries(ctx context.Context) ([]service.ProviderBinary, error) {
	report, err := r.client().Providers(ctx, false)
	if err != nil {
		return nil, err
	}
	out := make([]service.ProviderBinary, 0, len(report))
	for _, b := range report {
		out = append(out, service.ProviderBinary{
			ID:        b.ID,
			Name:      b.Name,
			Installed: b.Installed,
			Version:   b.Version,
			Latest:    b.Latest,
			Source:    b.Source,
			HasUpdate: b.HasUpdate,
		})
	}
	return out, nil
}

func (r *Remote) UpdateInfo(ctx context.Context) (*updater.Info, error) {
	u, err := r.client().Update(ctx)
	if err != nil || !u.HasUpdate {
		return nil, err
	}
	return &updater.Info{
		CurrentVersion: u.Current,
		LatestVersion:  u.Latest,
		Tag:            u.Tag,
		AssetName:      u.AssetName,
		ReleaseURL:     u.ReleaseURL,
		HasUpdate:      true,
		Method:         updater.Method(u.Method),
	}, nil
}

func (r *Remote) ApplyUpdate(ctx context.Context) error { return r.client().ApplyUpdate(ctx) }

// Events follows the running ftm's event stream until ctx ends. When the
// stream drops it says so, finds the instance again and asks the model to
// read everything again once it is back.
func (r *Remote) Events(ctx context.Context) <-chan tea.Msg {
	out := make(chan tea.Msg)

	r.mu.Lock()
	r.out, r.ctx = out, ctx
	watched := r.watched
	r.mu.Unlock()
	if watched != "" {
		r.WatchLogs(watched)
	}

	// out is never closed: the log followers may still be sending when ctx
	// ends. The model stops reading when ctx ends.
	go func() {
		send := func(msg tea.Msg) bool {
			select {
			case out <- msg:
				return true
			case <-ctx.Done():
				return false
			}
		}

		delay := reconnectMin
		lost := false
		for ctx.Err() == nil {
			opened := false
			_ = r.client().Events(ctx, func(e client.Event) bool {
				if e.Name == client.EventOpen {
					opened = true
					delay = reconnectMin
					if lost {
						lost = false
						if !send(connectionMsg{lost: false}) {
							return false
						}
					}
				}
				if msg := remoteMessage(e); msg != nil {
					return send(msg)
				}
				return true
			})
			if ctx.Err() != nil {
				return
			}
			if !lost {
				lost = true
				if !send(connectionMsg{lost: true}) {
					return
				}
			}
			if !opened {
				delay = min(delay*2, reconnectMax)
			}
			if !sleep(ctx, delay) {
				return
			}
			r.rediscover()
		}
	}()
	return out
}

// rediscover points the backend at the running ftm again, which has a new
// port and token after a restart.
func (r *Remote) rediscover() {
	if r.find == nil {
		return
	}
	c, err := r.find()
	if err != nil || c == nil {
		return
	}
	r.mu.Lock()
	r.c = c
	watched := r.watched
	r.mu.Unlock()
	if watched != "" {
		r.WatchLogs(watched)
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// remoteMessage turns one event of /api/events into a model message.
func remoteMessage(e client.Event) tea.Msg {
	switch e.Name {
	case client.EventOpen, "resync":
		return prefsChangedMsg{}
	case "tunnel_state":
		var t client.Tunnel
		if e.Decode(&t) != nil {
			return nil
		}
		status := remoteStatus(t)
		return statusUpdateMsg{tunnelID: status.ID, status: status}
	case "tunnel_deleted", "tunnel_stats":
		return tunnelsChangedMsg{}
	case "install":
		var p struct {
			Install struct {
				Provider string `json:"provider"`
			} `json:"install"`
			Percent float64 `json:"percent"`
			Current int64   `json:"current"`
			Total   int64   `json:"total"`
			Done    bool    `json:"done"`
		}
		if e.Decode(&p) != nil {
			return nil
		}
		return downloadProgressMsg(provider.DownloadProgress{
			Name:    p.Install.Provider,
			Percent: p.Percent,
			Current: p.Current,
			Total:   p.Total,
			Done:    p.Done,
		})
	case "update_available":
		var u client.UpdateInfo
		if e.Decode(&u) != nil {
			return nil
		}
		return updateCheckMsg{info: &updater.Info{
			CurrentVersion: u.Current,
			LatestVersion:  u.Latest,
			Tag:            u.Tag,
			AssetName:      u.AssetName,
			ReleaseURL:     u.ReleaseURL,
			HasUpdate:      true,
			Method:         updater.Method(u.Method),
		}}
	}
	return nil
}

// WatchLogs follows one tunnel's output, replacing the one followed before.
// The recent lines are the model's to read with Logs; only new lines arrive
// on Events.
func (r *Remote) WatchLogs(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.watched = id
	if r.stopWatch != nil {
		r.stopWatch()
		r.stopWatch = nil
	}
	if id == "" || r.out == nil || r.ctx.Err() != nil {
		return
	}

	ctx, cancel := context.WithCancel(r.ctx)
	r.stopWatch = cancel
	c, out := r.c, r.out
	go followLogs(ctx, c, id, out)
}

func followLogs(ctx context.Context, c *client.Client, id string, out chan<- tea.Msg) {
	delay := reconnectMin
	for ctx.Err() == nil {
		_ = c.FollowLogs(ctx, id, func(e client.Event) bool {
			if e.Name != "log" {
				return true
			}
			var msg struct {
				Line string `json:"line"`
			}
			if e.Decode(&msg) != nil {
				return true
			}
			select {
			case out <- logLineMsg{tunnelID: id, line: msg.Line}:
				return true
			case <-ctx.Done():
				return false
			}
		})
		if !sleep(ctx, delay) {
			return
		}
		delay = min(delay*2, reconnectMax)
	}
}
