// Package events carries every change in ftm from the code that makes it to
// whoever shows it. Publishers never wait for subscribers: each subscriber
// has its own bounded queue, so a stuck browser tab or a busy TUI cannot slow
// down the process manager or a download.
package events

import (
	"time"

	"github.com/sthbryan/ftm/internal/config"
)

// Kind names an event type, for filtering and logs.
type Kind string

const (
	KindTunnelSaved     Kind = "tunnel_saved"
	KindTunnelDeleted   Kind = "tunnel_deleted"
	KindTunnelStatus    Kind = "tunnel_status"
	KindTunnelStats     Kind = "tunnel_stats"
	KindInstallProgress Kind = "install_progress"
	KindLogLine         Kind = "log_line"
	KindSettingsChanged Kind = "settings_changed"
	KindUpdateAvailable Kind = "update_available"

	KindProviderUpdatesAvailable Kind = "provider_updates_available"
	KindProviderUpdated          Kind = "provider_updated"
)

// Event is one change. The set is closed: only the types in this package
// implement it.
type Event interface {
	Kind() Kind
}

// TunnelSaved is published when a saved connection is created or edited.
type TunnelSaved struct {
	Tunnel  config.TunnelConfig
	Created bool
}

// TunnelDeleted is published when a saved connection is removed.
type TunnelDeleted struct {
	ID string
}

// TunnelStatus is published whenever a tunnel changes state: installing,
// starting, connecting, online, stopping, stopped, timeout or error.
type TunnelStatus struct {
	Status config.TunnelStatus
}

// TunnelStats is published when the visitor numbers of an online tunnel
// change.
type TunnelStats struct {
	ID             string
	Visitors       int
	ActiveSessions int
}

// InstallProgress is one step of a provider download. The last event of an
// install has Done set, or Err when it failed.
type InstallProgress struct {
	Provider string
	Percent  float64
	Current  int64
	Total    int64
	Done     bool
	Err      string
}

// LogLine is one line a tunnel process printed.
type LogLine struct {
	TunnelID string
	Line     string
}

// SettingsChanged carries the config after a settings change was saved.
type SettingsChanged struct {
	Config config.Config
}

// UpdateAvailable is published once when a newer ftm release is found.
type UpdateAvailable struct {
	Current    string
	Latest     string
	Tag        string
	AssetName  string
	ReleaseURL string
	Method     string
}

// ProviderBinary describes the installed binary of one provider: where it
// comes from, its version and whether ftm can update it.
type ProviderBinary struct {
	ID         string
	Name       string
	Installed  bool
	Version    string
	Latest     string
	Source     string
	Path       string
	HasUpdate  bool
	Updatable  bool
	Updating   bool
	ReleaseURL string
	CheckedAt  time.Time
	Error      string
}

// ProviderUpdatesAvailable is published once per provider and release when a
// newer provider binary is found.
type ProviderUpdatesAvailable struct {
	Providers []ProviderBinary
}

// ProviderUpdated is published once when a provider binary update finishes,
// whether it worked or not. Restarted lists the tunnels started again after
// the swap.
type ProviderUpdated struct {
	Provider  string
	OK        bool
	Version   string
	Error     string
	Restarted []string
	Binary    ProviderBinary
}

func (TunnelSaved) Kind() Kind     { return KindTunnelSaved }
func (TunnelDeleted) Kind() Kind   { return KindTunnelDeleted }
func (TunnelStatus) Kind() Kind    { return KindTunnelStatus }
func (TunnelStats) Kind() Kind     { return KindTunnelStats }
func (InstallProgress) Kind() Kind { return KindInstallProgress }
func (LogLine) Kind() Kind         { return KindLogLine }
func (SettingsChanged) Kind() Kind { return KindSettingsChanged }
func (UpdateAvailable) Kind() Kind { return KindUpdateAvailable }
func (ProviderUpdatesAvailable) Kind() Kind {
	return KindProviderUpdatesAvailable
}
func (ProviderUpdated) Kind() Kind { return KindProviderUpdated }
