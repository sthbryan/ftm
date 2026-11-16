package tui

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/provider"
	"github.com/sthbryan/ftm/internal/ui/tui/views"
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
	App                 Backend
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
	DownloadProgress    provider.DownloadProgress
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
	messageSeq          int
	messageTimer        int
	prefs               Prefs
	logsID              string
	logs                []string
	lost                bool
	ctx                 context.Context
	events              <-chan tea.Msg
	stopEvents          func()
}

type TunnelDraft struct {
	ID       string
	Name     string
	Provider string
	Port     string
}

type TunnelItem struct {
	Tunnel core.TunnelConfig
	Status core.TunnelStatus
	// Running is true while the tunnel is starting, connecting or online.
	Running bool
}

func (i TunnelItem) FilterValue() string { return i.Tunnel.Name }

func (i TunnelItem) Title() string { return i.Tunnel.Name }

func (i TunnelItem) Description() string {
	status := i18n.T("status_offline")
	switch i.Status.State {
	case core.TunnelStateStarting:
		status = i18n.T("status_starting")
	case core.TunnelStateConnecting:
		status = i18n.T("status_connecting")
	case core.TunnelStateOnline:
		status = i18n.T("status_online")
	case core.TunnelStateError:
		status = i18n.T("status_error")
	case core.TunnelStateTimeout:
		status = i18n.T("status_timeout")
	}
	return fmt.Sprintf("%s | %s %d | %s", i.Tunnel.Provider, i18n.T("port"), i.Tunnel.LocalPort, status)
}
