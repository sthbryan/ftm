package httpapi

import (
	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/service"
)

// The types in this file are what goes over the wire: in REST responses, as
// WebSocket messages and as Server-Sent Events. An SSE event carries the same
// JSON as the WebSocket message of the same type, and its event name is that
// type, so the web can move from one to the other without new parsing.

// Event names on /api/events. Each one is also the "type" field of its data.
const (
	eventTunnelState             = "tunnel_state"
	eventTunnelDeleted           = "tunnel_deleted"
	eventTunnelStats             = "tunnel_stats"
	eventInstall                 = "install"
	eventNotification            = "notification"
	eventUpdateAvailable         = "update_available"
	eventProviderUpdateAvailable = "provider_update_available"
	eventProviderUpdated         = "provider_updated"
	eventResync                  = "resync"
	eventLog                     = "log"
	eventLogSnapshot             = "log_snapshot"
)

// TunnelState is the only shape a tunnel state takes on the wire. The REST
// list, the broadcasts and the action responses all build it here, so the
// payload cannot drift between them or from the TypeScript types.
type TunnelState struct {
	Type           string `json:"type" enum:"tunnel_state"`
	ID             string `json:"id"`
	Name           string `json:"name"`
	Provider       string `json:"provider"`
	Port           int    `json:"port"`
	State          string `json:"state" doc:"stopped, need_installing, installing, starting, connecting, online, stopping, timeout or error"`
	PublicURL      string `json:"publicUrl"`
	ErrorMessage   string `json:"errorMessage"`
	ExpiresAt      int64  `json:"expiresAt" doc:"Unix milliseconds; 0 when the provider does not expire"`
	StartedAt      int64  `json:"startedAt" doc:"Unix milliseconds; 0 when not running"`
	Visitors       int    `json:"visitors,omitempty"`
	ActiveSessions int    `json:"activeSessions,omitempty"`
}

// tunnelStatePatch is for the short-lived states that are not driven by the
// stored tunnel, such as installing. On /api/events it is a tunnel_state event
// with only these fields.
type tunnelStatePatch struct {
	Type         string `json:"type"`
	ID           string `json:"id"`
	State        string `json:"state"`
	Provider     string `json:"provider,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

func tunnelStateFromView(v service.TunnelView) TunnelState {
	msg := TunnelState{
		Type:     eventTunnelState,
		ID:       v.Tunnel.ID,
		Name:     v.Tunnel.Name,
		Provider: string(v.Tunnel.Provider),
		Port:     v.Tunnel.LocalPort,
		State:    string(config.TunnelStateStopped),
	}

	if v.Running {
		msg.State = string(v.Status.State)
		msg.PublicURL = v.Status.PublicURL
		msg.ErrorMessage = v.Status.ErrorMessage
		msg.ExpiresAt = v.Status.ExpiresAt
		msg.StartedAt = v.Status.StartedAt
		msg.Visitors = v.Status.Visitors
		msg.ActiveSessions = v.Status.ActiveSessions
	}

	if msg.State == string(config.TunnelStateStopped) && v.NeedsInstall {
		msg.State = "need_installing"
	}
	return msg
}

// tunnelStateFromStatus builds the message from a status the supervisor
// already published.
func tunnelStateFromStatus(status config.TunnelStatus) TunnelState {
	return TunnelState{
		Type:         eventTunnelState,
		ID:           status.ID,
		Name:         status.Name,
		Provider:     string(status.Provider),
		Port:         status.LocalPort,
		State:        string(status.State),
		PublicURL:    status.PublicURL,
		ErrorMessage: status.ErrorMessage,
		ExpiresAt:    status.ExpiresAt,
		StartedAt:    status.StartedAt,
	}
}

type tunnelDeletedMessage struct {
	Type string `json:"type" enum:"tunnel_deleted"`
	ID   string `json:"id"`
}

type tunnelStatsMessage struct {
	Type           string `json:"type" enum:"tunnel_stats"`
	ID             string `json:"id"`
	Visitors       int    `json:"visitors"`
	ActiveSessions int    `json:"activeSessions"`
}

type installStep struct {
	Provider string  `json:"provider"`
	Percent  float64 `json:"percent"`
	Step     string  `json:"step" doc:"Installing... while downloading, Done at the end"`
}

type installMessage struct {
	Type    string      `json:"type" enum:"install"`
	Install installStep `json:"install"`
	Percent float64     `json:"percent"`
	Current int64       `json:"current"`
	Total   int64       `json:"total"`
	Done    bool        `json:"done"`
}

func installMessageFrom(p events.InstallProgress) installMessage {
	if p.Err != "" {
		// The failure reaches the tunnel as an error state; the progress bar
		// only needs to know the download ended.
		p.Done = true
	}
	step := "Installing..."
	if p.Done {
		step = "Done"
	}
	return installMessage{
		Type:    eventInstall,
		Install: installStep{Provider: p.Provider, Percent: p.Percent, Step: step},
		Percent: p.Percent,
		Current: p.Current,
		Total:   p.Total,
		Done:    p.Done,
	}
}

type updateAvailableMessage struct {
	Type       string `json:"type" enum:"update_available"`
	Current    string `json:"current"`
	Latest     string `json:"latest"`
	Tag        string `json:"tag"`
	AssetName  string `json:"assetName"`
	ReleaseURL string `json:"releaseUrl"`
	Method     string `json:"method"`
}

type notificationBody struct {
	Channel      string `json:"channel" enum:"toast,os"`
	Title        string `json:"title"`
	Body         string `json:"body"`
	ToastType    string `json:"toastType"`
	SoundType    string `json:"soundType"`
	SoundEnabled bool   `json:"soundEnabled"`
}

type notificationMessage struct {
	Type         string           `json:"type" enum:"notification"`
	Notification notificationBody `json:"notification"`
}

// ProviderBinaryRow is one row of the installed providers list.
type ProviderBinaryRow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Installed  bool   `json:"installed"`
	Version    string `json:"version"`
	Latest     string `json:"latest"`
	Source     string `json:"source" enum:"bundled,path,system,none"`
	Path       string `json:"path"`
	HasUpdate  bool   `json:"hasUpdate"`
	Updatable  bool   `json:"updatable"`
	Updating   bool   `json:"updating"`
	ReleaseURL string `json:"releaseUrl"`
	CheckedAt  int64  `json:"checkedAt" doc:"Unix milliseconds of the last release lookup; 0 before the first"`
	Error      string `json:"error,omitempty"`
}

func providerBinaryRow(b service.ProviderBinary) ProviderBinaryRow {
	row := ProviderBinaryRow{
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
	if !b.CheckedAt.IsZero() {
		row.CheckedAt = b.CheckedAt.UnixMilli()
	}
	return row
}

type providerUpdateNotice struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Latest     string `json:"latest"`
	Source     string `json:"source"`
	Updatable  bool   `json:"updatable"`
	ReleaseURL string `json:"releaseUrl"`
}

type providerUpdateAvailableMessage struct {
	Type      string                 `json:"type" enum:"provider_update_available"`
	Providers []providerUpdateNotice `json:"providers"`
}

func providerUpdateAvailableFrom(e events.ProviderUpdatesAvailable) providerUpdateAvailableMessage {
	msg := providerUpdateAvailableMessage{Type: eventProviderUpdateAvailable, Providers: make([]providerUpdateNotice, 0, len(e.Providers))}
	for _, b := range e.Providers {
		msg.Providers = append(msg.Providers, providerUpdateNotice{
			ID:         b.ID,
			Name:       b.Name,
			Version:    b.Version,
			Latest:     b.Latest,
			Source:     b.Source,
			Updatable:  b.Updatable,
			ReleaseURL: b.ReleaseURL,
		})
	}
	return msg
}

type providerUpdatedMessage struct {
	Type      string             `json:"type" enum:"provider_updated"`
	Provider  string             `json:"provider"`
	OK        bool               `json:"ok"`
	Version   string             `json:"version,omitempty"`
	Error     string             `json:"error,omitempty"`
	Restarted []string           `json:"restarted"`
	Binary    *ProviderBinaryRow `json:"binary,omitempty"`
}

func providerUpdatedFrom(e events.ProviderUpdated) providerUpdatedMessage {
	binary := providerBinaryRow(e.Binary)
	restarted := e.Restarted
	if restarted == nil {
		restarted = []string{}
	}
	return providerUpdatedMessage{
		Type:      eventProviderUpdated,
		Provider:  e.Provider,
		OK:        e.OK,
		Version:   e.Version,
		Error:     e.Error,
		Restarted: restarted,
		Binary:    &binary,
	}
}

// resyncMessage tells an SSE client that it missed events it cannot get back
// (it was away too long, or the server restarted), so it should reload its
// state over REST.
type resyncMessage struct {
	Type   string `json:"type" enum:"resync"`
	Reason string `json:"reason"`
}

type logMessage struct {
	Type string `json:"type" enum:"log"`
	ID   string `json:"id"`
	Line string `json:"line"`
}

// LogSnapshot is a tunnel's recent output.
type LogSnapshot struct {
	Type  string   `json:"type,omitempty" enum:"log_snapshot"`
	ID    string   `json:"id"`
	Lines []string `json:"lines"`
}
