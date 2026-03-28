package app

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sthbryan/ftm/internal/clipboard"
	"github.com/sthbryan/ftm/internal/config"
)

func (m *Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.Keys.Up):
		m.moveCursorUp()

	case key.Matches(msg, m.Keys.Down):
		m.moveCursorDown()

	case key.Matches(msg, m.Keys.Settings):
		m.openSettings()
		return m, nil

	case key.Matches(msg, m.Keys.Enter), key.Matches(msg, m.Keys.Toggle):
		return m.handleListToggle()

	case key.Matches(msg, m.Keys.Logs):
		return m.handleListLogs()

	case key.Matches(msg, m.Keys.Copy):
		m.handleListCopy()

	case key.Matches(msg, m.Keys.Web):
		m.openDashboard()

	case key.Matches(msg, m.Keys.Config):
		m.openConfigDir()

	case key.Matches(msg, m.Keys.Add):
		m.startAddForm()

	case key.Matches(msg, m.Keys.Edit):
		return m.startEditForm()

	case key.Matches(msg, m.Keys.Delete):
		return m.handleListDelete()
	}

	return m, nil
}

func (m *Model) moveCursorUp() {
	if m.Cursor > 0 {
		m.Cursor--
	}
}

func (m *Model) moveCursorDown() {
	if m.Cursor < len(m.Items)-1 {
		m.Cursor++
	}
}

func (m *Model) handleListToggle() (tea.Model, tea.Cmd) {
	if item, ok := m.selectedItem(); ok {
		if m.App.Manager.IsRunning(item.Tunnel.ID) {
			m.playBeep()
			return m, m.stopTunnel(item)
		}
		m.playBeep()
		return m, m.startTunnel(item)
	}
	return m, nil
}

func (m *Model) handleListLogs() (tea.Model, tea.Cmd) {
	if item, ok := m.selectedItem(); ok {
		m.SelectedTunnel = item.Tunnel.ID
		m.State = viewLogs
		m.updateLogViewport()
	}
	return m, nil
}

func (m *Model) handleListCopy() {
	if item, ok := m.selectedItem(); ok {
		m.copyTunnelURL(item)
	}
}

func (m *Model) startAddForm() {
	m.State = viewAddForm
	m.FormFocus = 0
	m.FormValues = FormData{
		Provider: string(config.ProviderCloudflared),
		Port:     "30000",
	}
}

func (m *Model) startEditForm() (tea.Model, tea.Cmd) {
	if item, ok := m.selectedItem(); ok {
		if item.Status.State != config.TunnelStateStopped {
			m.showMessage("Stop tunnel first to edit")
			return m, nil
		}
		m.State = viewEditForm
		m.editingTunnelID = item.Tunnel.ID
		m.FormFocus = 0
		m.FormValues = FormData{
			ID:       item.Tunnel.ID,
			Name:     item.Tunnel.Name,
			Provider: string(item.Tunnel.Provider),
			Port:     fmt.Sprintf("%d", item.Tunnel.LocalPort),
		}
	}
	return m, nil
}

func (m *Model) handleListDelete() (tea.Model, tea.Cmd) {
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
	m.showMessage("Dashboard opened in browser")
}

func (m *Model) openConfigDir() {
	if err := m.App.OpenConfigDir(); err != nil {
		m.showMessage("Error opening config folder: " + err.Error())
		return
	}
	m.showMessage("Config folder opened")
}

func (m *Model) playBeep() {
	fmt.Fprint(os.Stdout, Bell)
}
