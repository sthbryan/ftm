package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/updater"
)

// Backend is the ftm the TUI drives: the core run in this process (Local) or
// a running instance over its API (Remote). Every change goes through it and
// every update arrives on Events, so the TUI and an open web view or desktop
// app see each other's changes as they happen.
type Backend interface {
	// Attached is true when another process owns the tunnels.
	Attached() bool
	// DashboardURL is the web view of the ftm the TUI drives.
	DashboardURL() string
	// Sessions is how many web views are open, or -1 when unknown.
	Sessions() int
	// ConfigDir is the folder that holds config.yaml.
	ConfigDir() string

	Tunnels(ctx context.Context) ([]TunnelItem, error)
	Prefs(ctx context.Context) (Prefs, error)
	SavePrefs(ctx context.Context, p Prefs) error

	CreateTunnel(ctx context.Context, in TunnelInput) error
	UpdateTunnel(ctx context.Context, id string, in TunnelInput) error
	DeleteTunnel(ctx context.Context, id string) error
	ClearTunnels(ctx context.Context) error
	// StartTunnel reports installing when the provider is downloaded first;
	// the tunnel starts once the download finishes.
	StartTunnel(ctx context.Context, id string) (installing bool, err error)
	StopTunnel(ctx context.Context, id string) error
	Logs(ctx context.Context, id string) ([]string, error)
	ProviderBinaries(ctx context.Context) ([]service.ProviderBinary, error)

	UpdateInfo(ctx context.Context) (*updater.Info, error)
	ApplyUpdate(ctx context.Context) error

	// Events delivers every change as a model message until ctx ends.
	Events(ctx context.Context) <-chan tea.Msg
	// WatchLogs asks for the output of one tunnel to arrive on Events as
	// logLineMsg. An empty id stops watching.
	WatchLogs(id string)
}

// TunnelInput is a connection as the editor fills it in.
type TunnelInput struct {
	Name      string
	Provider  core.Provider
	LocalPort int
}

// Prefs are the settings the TUI shows.
type Prefs struct {
	Notifications bool
	Sound         bool
	Language      string
	Onboarded     bool
}

// Messages a Backend delivers on Events.
type (
	statusUpdateMsg struct {
		tunnelID string
		status   core.TunnelStatus
	}

	// tunnelsChangedMsg is any change to the saved connections or their
	// numbers.
	tunnelsChangedMsg struct{}

	// prefsChangedMsg says the settings changed, or that the TUI may have
	// missed changes and should read everything again.
	prefsChangedMsg struct{}

	// logLineMsg is one new line of the watched tunnel's output.
	logLineMsg struct {
		tunnelID string
		line     string
	}

	// connectionMsg says whether the running ftm the TUI is attached to can
	// be reached.
	connectionMsg struct {
		lost bool
	}
)
