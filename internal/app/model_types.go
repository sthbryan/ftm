package app

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/viewport"

	"github.com/sthbryan/ftm/internal/app/ui/views"
	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/providers"
	"github.com/sthbryan/ftm/internal/updater"
)

type viewState int

const (
	viewList viewState = iota
	viewLogs
	viewNewTunnel
	viewEditTunnel
	viewConfirm
	viewDownloading
	viewSettings
)

type confirmKind int

const (
	confirmDeleteTunnel confirmKind = iota
	confirmClearTunnels
)

type Settings struct {
	NotificationsEnabled bool
	NotificationSound    bool
	Theme                string
}

const TwoColumnThreshold = 100

type Model struct {
	App                 *App
	Keys                KeyMap
	Help                help.Model
	State               viewState
	Width               int
	Height              int
	Cursor              int
	Items               []list.Item
	LogViewport         viewport.Model
	SelectedTunnel      string
	EditorFocus         int
	Draft               TunnelDraft
	editingTunnelID     string
	Message             string
	DownloadProgress    providers.DownloadProgress
	DownloadingProvider string
	ProgressBar         progress.Model
	SettingsView        *views.SettingsView
	UpdateAvailable     *updater.Info
	RelaunchOnExit      bool
	listTop             int
	listFirst           int
	pendingDeleteID     string
	pendingDeleteName   string
	pendingConfirm      confirmKind
	messageUntil        time.Time
	events              <-chan events.Event
	stopEvents          func()
}

type TunnelDraft struct {
	ID       string
	Name     string
	Provider string
	Port     string
}

type TunnelItem struct {
	Tunnel config.TunnelConfig
	Status config.TunnelStatus
}

func (i TunnelItem) FilterValue() string { return i.Tunnel.Name }

func (i TunnelItem) Title() string { return i.Tunnel.Name }

func (i TunnelItem) Description() string {
	status := i18n.T("status_offline")
	switch i.Status.State {
	case config.TunnelStateStarting:
		status = i18n.T("status_starting")
	case config.TunnelStateConnecting:
		status = i18n.T("status_connecting")
	case config.TunnelStateOnline:
		status = i18n.T("status_online")
	case config.TunnelStateError:
		status = i18n.T("status_error")
	case config.TunnelStateTimeout:
		status = i18n.T("status_timeout")
	}
	return fmt.Sprintf("%s | %s %d | %s", i.Tunnel.Provider, i18n.T("port"), i.Tunnel.LocalPort, status)
}
