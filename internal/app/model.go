package app

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"foundry-tunnel/internal/config"
	"foundry-tunnel/internal/providers"
)

type viewState int

const (
	viewList viewState = iota
	viewLogs
	viewAddForm
	viewConfirm
	viewDownloading
)

const TwoColumnThreshold = 100

type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Start    key.Binding
	Stop     key.Binding
	Logs     key.Binding
	Copy     key.Binding
	Web      key.Binding
	Add      key.Binding
	Delete   key.Binding
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
	Start: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "start"),
	),
	Stop: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "stop"),
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
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
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
	Message             string
	MessageTimer        int
	DownloadProgress    providers.DownloadProgress
	DownloadingProvider string
	PendingTunnel       *config.TunnelConfig
}

type FormData struct {
	Name     string
	Provider string
	Port     string
}

type TunnelItem struct {
	Tunnel config.TunnelConfig
	Status config.TunnelStatus
}

func (i TunnelItem) FilterValue() string { return i.Tunnel.Name }
func (i TunnelItem) Title() string       { return i.Tunnel.Name }
func (i TunnelItem) Description() string {
	status := "● OFFLINE"
	if i.Status.Running {
		if i.Status.Starting {
			status = "⟳ STARTING"
		} else {
			status = "▶ ONLINE"
		}
	}
	return fmt.Sprintf("%s | Port %d | %s", i.Tunnel.Provider, i.Tunnel.LocalPort, status)
}

type (
	statusUpdateMsg struct {
		tunnelID string
		status   config.TunnelStatus
	}
	tickMsg struct{}
)

func NewModel(app *App) *Model {
	h := help.New()
	h.ShowAll = true

	m := &Model{
		App:    app,
		Keys:   DefaultKeys,
		Help:   h,
		State:  viewList,
		Cursor: 0,
		FormValues: FormData{
			Provider: string(config.ProviderPlayitgg),
			Port:     "30000",
		},
	}

	m.refreshItems()
	return m
}

func (m *Model) refreshItems() {
	items := make([]list.Item, 0, len(m.App.Config.Tunnels))
	for _, t := range m.App.Config.Tunnels {
		status := t.Status()
		if s, ok := m.App.Manager.GetStatus(t.ID); ok {
			status = s
		}
		items = append(items, TunnelItem{Tunnel: t, Status: status})
	}
	m.Items = items
}

func (m *Model) selectedItem() (TunnelItem, bool) {
	if m.Cursor < 0 || m.Cursor >= len(m.Items) {
		return TunnelItem{}, false
	}
	item, ok := m.Items[m.Cursor].(TunnelItem)
	return item, ok
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		m.checkDownloadProgress(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Every(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

type downloadProgressMsg providers.DownloadProgress

func (m *Model) showMessage(msg string) {
	m.Message = msg
	m.MessageTimer = 30
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func (m *Model) startTunnel(item TunnelItem) tea.Cmd {
	return func() tea.Msg {

		if needsInstall, canInstall := m.App.Manager.CheckInstallation(item.Tunnel.Provider); needsInstall && canInstall {
			m.DownloadingProvider = string(item.Tunnel.Provider)
			m.State = viewDownloading
			m.PendingTunnel = &item.Tunnel
			return m.installProvider(item.Tunnel.Provider)()
		}

		err := m.App.Manager.Start(item.Tunnel, func(status config.TunnelStatus) {
		})

		if err != nil {
			return statusUpdateMsg{
				tunnelID: item.Tunnel.ID,
				status:   config.TunnelStatus{Error: err.Error()},
			}
		}

		return nil
	}
}

func (m *Model) installProvider(providerType config.Provider) tea.Cmd {
	return func() tea.Msg {
		err := m.App.Manager.InstallProvider(providerType)
		if err != nil {
			return statusUpdateMsg{
				tunnelID: "",
				status:   config.TunnelStatus{Error: "Install failed: " + err.Error()},
			}
		}
		return downloadProgressMsg{Done: true, Percent: 100}
	}
}

func (m *Model) stopTunnel(item TunnelItem) tea.Cmd {
	return func() tea.Msg {
		m.App.Manager.Stop(item.Tunnel.ID)
		return nil
	}
}

func (m *Model) checkDownloadProgress() tea.Cmd {
	return func() tea.Msg {
		for progress := range m.App.DownloadProgress {
			return downloadProgressMsg(progress)
		}
		return nil
	}
}
