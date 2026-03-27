package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sthbryan/ftm/internal/clipboard"
	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/providers"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.LogViewport.Width = msg.Width - 4
		m.LogViewport.Height = msg.Height - 8
		return m, nil

	case tickMsg:
		m.refreshItems()

		if m.MessageTimer > 0 {
			m.MessageTimer--
			if m.MessageTimer == 0 {
				m.Message = ""
			}
		}
		return m, tickCmd()

	case downloadProgressMsg:
		m.DownloadProgress = providers.DownloadProgress(msg)
		if m.DownloadProgress.Done {
			m.State = viewList
			if m.PendingTunnel != nil {

				for _, item := range m.Items {
					if ti, ok := item.(TunnelItem); ok && ti.Tunnel.ID == m.PendingTunnel.ID {
						m.PendingTunnel = nil
						m.showMessage("Install complete! Starting tunnel...")
						return m, m.startTunnel(ti)
					}
				}
				m.PendingTunnel = nil
			}
			m.showMessage("Download complete!")
		}
		return m, m.checkDownloadProgress()

	case statusUpdateMsg:
		m.refreshItems()
		if msg.status.ErrorMessage != "" {
			m.playBeep()
			m.showMessage("Error: " + msg.status.ErrorMessage)
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.Keys.Quit):
		if m.State == viewList {
			return m, tea.Quit
		}
		m.State = viewList
		return m, nil

	case key.Matches(msg, m.Keys.Back):
		if m.State != viewList {
			m.State = viewList
			return m, nil
		}

	case key.Matches(msg, m.Keys.Help):
		m.Help.ShowAll = !m.Help.ShowAll
		return m, nil
	}

	switch m.State {
	case viewList:
		return m.handleListKey(msg)
	case viewLogs:
		return m.handleLogsKey(msg)
	case viewAddForm:
		return m.handleFormKey(msg)
	case viewDownloading:
		if key.Matches(msg, m.Keys.Back) || key.Matches(msg, m.Keys.Quit) {
			m.State = viewList
			return m, nil
		}
	}

	return m, nil
}

func (m *Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.Keys.Up):
		if m.Cursor > 0 {
			m.Cursor--
		}

	case key.Matches(msg, m.Keys.Down):
		if m.Cursor < len(m.Items)-1 {
			m.Cursor++
		}

	case key.Matches(msg, m.Keys.Enter), key.Matches(msg, m.Keys.Toggle):
		if item, ok := m.selectedItem(); ok {
			if m.App.Manager.IsRunning(item.Tunnel.ID) {
				m.playBeep()
				return m, m.stopTunnel(item)
			}
			m.playBeep()
			return m, m.startTunnel(item)
		}

	case key.Matches(msg, m.Keys.Logs):
		if item, ok := m.selectedItem(); ok {
			m.SelectedTunnel = item.Tunnel.ID
			m.State = viewLogs
			m.updateLogViewport()
		}

	case key.Matches(msg, m.Keys.Copy):
		if item, ok := m.selectedItem(); ok {
			m.copyTunnelURL(item)
		}

	case key.Matches(msg, m.Keys.Web):
		m.openDashboard()

	case key.Matches(msg, m.Keys.Config):
		m.openConfigDir()

	case key.Matches(msg, m.Keys.Add):
		m.State = viewAddForm
		m.FormFocus = 0
		m.FormValues = FormData{
			Provider: string(config.ProviderPlayitgg),
			Port:     "30000",
		}

	case key.Matches(msg, m.Keys.Delete):
		if item, ok := m.selectedItem(); ok {
			m.App.Manager.Stop(item.Tunnel.ID)
			m.App.Config.RemoveTunnel(item.Tunnel.ID)
			m.App.SaveConfig()
			m.refreshItems()
			if m.Cursor >= len(m.Items) && m.Cursor > 0 {
				m.Cursor--
			}
			m.showMessage("Tunnel deleted")
		}
	}

	return m, nil
}

func (m *Model) handleLogsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.LogViewport, cmd = m.LogViewport.Update(msg)
	m.updateLogViewport()
	return m, cmd
}

func (m *Model) updateLogViewport() {
	if m.SelectedTunnel == "" {
		return
	}

	logs := m.App.Manager.GetLogs(m.SelectedTunnel)
	content := strings.Join(logs, "\n")
	m.LogViewport.SetContent(content)
	m.LogViewport.GotoBottom()
}

func (m *Model) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.FormFocus = (m.FormFocus + 1) % 4

	case "shift+tab":
		m.FormFocus = (m.FormFocus - 1 + 4) % 4

	case "enter":
		if m.FormFocus == 3 {
			m.submitForm()
		} else {
			m.FormFocus++
		}

	case "esc":
		m.State = viewList

	case "left":
		if m.FormFocus == 1 {
			m.handleFormInput("left")
		}

	case "right":
		if m.FormFocus == 1 {
			m.handleFormInput("right")
		}

	default:
		m.handleFormInput(msg.String())
	}

	return m, nil
}

func (m *Model) handleFormInput(s string) {
	switch m.FormFocus {
	case 0:
		if s == "backspace" {
			if len(m.FormValues.Name) > 0 {
				m.FormValues.Name = m.FormValues.Name[:len(m.FormValues.Name)-1]
			}
		} else if len(s) == 1 {
			m.FormValues.Name += s
		}

	case 1:
		switch s {
		case "right":
			switch m.FormValues.Provider {
			case string(config.ProviderPlayitgg):
				m.FormValues.Provider = string(config.ProviderCloudflared)
			case string(config.ProviderCloudflared):
				m.FormValues.Provider = string(config.ProviderTunnelmole)
			case string(config.ProviderTunnelmole):
				m.FormValues.Provider = string(config.ProviderLocalhostRun)
			case string(config.ProviderLocalhostRun):
				m.FormValues.Provider = string(config.ProviderServeo)
			case string(config.ProviderServeo):
				m.FormValues.Provider = string(config.ProviderPinggy)
			case string(config.ProviderPinggy):
				m.FormValues.Provider = string(config.ProviderPlayitgg)
			default:
				m.FormValues.Provider = string(config.ProviderPlayitgg)
			}
		case "left":
			switch m.FormValues.Provider {
			case string(config.ProviderPlayitgg):
				m.FormValues.Provider = string(config.ProviderPinggy)
			case string(config.ProviderPinggy):
				m.FormValues.Provider = string(config.ProviderServeo)
			case string(config.ProviderServeo):
				m.FormValues.Provider = string(config.ProviderLocalhostRun)
			case string(config.ProviderLocalhostRun):
				m.FormValues.Provider = string(config.ProviderTunnelmole)
			case string(config.ProviderTunnelmole):
				m.FormValues.Provider = string(config.ProviderCloudflared)
			case string(config.ProviderCloudflared):
				m.FormValues.Provider = string(config.ProviderPlayitgg)
			default:
				m.FormValues.Provider = string(config.ProviderPlayitgg)
			}
		}

	case 2:
		if s == "backspace" {
			if len(m.FormValues.Port) > 0 {
				m.FormValues.Port = m.FormValues.Port[:len(m.FormValues.Port)-1]
			}
		} else if s >= "0" && s <= "9" && len(m.FormValues.Port) < 5 {
			m.FormValues.Port += s
		}
	}
}

func (m *Model) submitForm() {
	if m.FormValues.Name == "" || m.FormValues.Port == "" {
		m.showMessage("Name and Port are required")
		return
	}

	id := strings.ToLower(strings.ReplaceAll(m.FormValues.Name, " ", "-"))

	tunnel := config.TunnelConfig{
		ID:        id,
		Name:      m.FormValues.Name,
		Provider:  config.Provider(m.FormValues.Provider),
		LocalPort: parsePort(m.FormValues.Port),
	}

	m.App.Config.AddTunnel(tunnel)
	m.App.SaveConfig()
	m.refreshItems()
	m.State = viewList
	m.showMessage("Tunnel added!")
}

func parsePort(s string) int {
	var port int
	for _, c := range s {
		port = port*10 + int(c-'0')
	}
	return port
}

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.State != viewList {
		return m, nil
	}

	itemHeight := 3
	headerHeight := 4
	clickedIdx := (msg.Y - headerHeight) / itemHeight

	switch msg.Type {
	case tea.MouseLeft:
		if clickedIdx >= 0 && clickedIdx < len(m.Items) {
			m.Cursor = clickedIdx
		}

	case tea.MouseWheelUp:
		if m.Cursor > 0 {
			m.Cursor--
		}

	case tea.MouseWheelDown:
		if m.Cursor < len(m.Items)-1 {
			m.Cursor++
		}
	}

	return m, nil
}

func (m *Model) copyTunnelURL(item TunnelItem) {
	if item.Status.PublicURL != "" {
		clipboard.Write(item.Status.PublicURL)
		m.showMessage("Copied URL!")
		return
	}

	m.showMessage("No URL available - start tunnel first")
}

func (m *Model) openDashboard() {
	if err := m.App.OpenDashboard(); err != nil {
		m.showMessage("Error opening dashboard: " + err.Error())
		return
	}
	m.showMessage("Dashboard opened in browser 🌐")
}

func (m *Model) openConfigDir() {
	if err := m.App.OpenConfigDir(); err != nil {
		m.showMessage("Error opening config folder: " + err.Error())
		return
	}
	m.showMessage("Config folder opened 📁")
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Toggle, k.Logs, k.Copy, k.Web},
		{k.Add, k.Delete, k.Config},
		{k.Back, k.Help, k.Quit},
	}
}

func (m *Model) playBeep() {
	fmt.Fprint(os.Stdout, Bell)
}
