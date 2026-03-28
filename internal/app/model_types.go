package app

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"

	"github.com/sthbryan/ftm/internal/app/ui"
	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/providers"
)

type viewState int

const (
	viewList viewState = iota
	viewLogs
	viewAddForm
	viewEditForm
	viewConfirm
	viewDownloading
	viewSettings
)

type Settings struct {
	NotificationsEnabled bool
	NotificationSound   bool
	Theme                string
}

const TwoColumnThreshold = 100

type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Toggle   key.Binding
	Logs     key.Binding
	Copy     key.Binding
	Web      key.Binding
	Add      key.Binding
	Edit     key.Binding
	Delete   key.Binding
	Config   key.Binding
	Settings key.Binding
	Back     key.Binding
	Quit     key.Binding
	Help     key.Binding
}

var DefaultKeys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "toggle"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "start/stop"),
	),
	Logs: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "logs"),
	),
	Copy: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy URL"),
	),
	Web: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "open web"),
	),
	Add: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Config: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open config"),
	),
	Settings: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "settings"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "b"),
		key.WithHelp("esc/b", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

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
	FormFocus           int
	FormValues          FormData
	editingTunnelID     string
	Message             string
	MessageTimer        int
	DownloadProgress    providers.DownloadProgress
	DownloadingProvider string
	PendingTunnel       *config.TunnelConfig
	ProgressBar         progress.Model
	SettingsView        *ui.SettingsView
}

type FormData struct {
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
	status := "OFFLINE"
	switch i.Status.State {
	case config.TunnelStateStarting:
		status = "STARTING"
	case config.TunnelStateConnecting:
		status = "CONNECTING"
	case config.TunnelStateOnline:
		status = "ONLINE"
	}
	return fmt.Sprintf("%s | Port %d | %s", i.Tunnel.Provider, i.Tunnel.LocalPort, status)
}
