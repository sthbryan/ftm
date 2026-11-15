package app

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/clipboard"
	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/notifications"
	"github.com/sthbryan/ftm/internal/updater"
)

func (m *Model) handleListKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.Keys.Up):
		m.moveCursorUp()

	case key.Matches(msg, m.Keys.Down):
		m.moveCursorDown()

	case key.Matches(msg, m.Keys.Settings):
		m.openSettings()
		return m, m.loadProviderSummary()

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
		m.startNewTunnel()

	case key.Matches(msg, m.Keys.Edit):
		return m.startEditTunnel()

	case key.Matches(msg, m.Keys.Delete):
		return m.handleListDelete()

	case key.Matches(msg, m.Keys.Update):
		return m.handleListUpdate()
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

func (m *Model) startNewTunnel() {
	m.State = viewNewTunnel
	m.EditorFocus = 0
	m.Draft = TunnelDraft{
		Provider: string(config.ProviderCloudflared),
		Port:     "30000",
	}
}

func (m *Model) startEditTunnel() (tea.Model, tea.Cmd) {
	if item, ok := m.selectedItem(); ok {
		if item.Status.State != config.TunnelStateStopped {
			m.showMessage(i18n.T("error_tunnel_running"))
			return m, nil
		}
		m.State = viewEditTunnel
		m.editingTunnelID = item.Tunnel.ID
		m.EditorFocus = 0
		m.Draft = TunnelDraft{
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
		m.pendingDeleteID = item.Tunnel.ID
		m.pendingDeleteName = item.Tunnel.Name
		m.State = viewConfirm
	}
	return m, nil
}

func (m *Model) confirmDelete() (tea.Model, tea.Cmd) {
	id := m.pendingDeleteID
	m.cancelDelete()

	if id == "" {
		return m, nil
	}

	saveErr := m.App.Service.DeleteTunnel(id)
	m.refreshItems()

	if m.Cursor >= len(m.Items) && m.Cursor > 0 {
		m.Cursor--
	}

	if saveErr != nil {
		m.showMessage(i18n.T("error_saving_config") + ": " + saveErr.Error())
	} else {
		m.showMessage(i18n.T("tunnel_deleted"))
	}

	return m, nil
}

func (m *Model) cancelDelete() {
	m.pendingDeleteID = ""
	m.pendingDeleteName = ""
	m.State = viewList
}

func (m *Model) cancelConfirm() {
	if m.pendingConfirm == confirmClearTunnels {
		m.pendingConfirm = confirmDeleteTunnel
		m.State = viewSettings
		return
	}
	m.cancelDelete()
}

func (m *Model) copyTunnelURL(item TunnelItem) {
	if item.Status.PublicURL != "" {
		if err := clipboard.Write(item.Status.PublicURL); err != nil {
			m.showMessage(i18n.T("copy_failed") + ": " + err.Error())
			return
		}
		m.showMessage(i18n.T("url_copied"))
		return
	}
	m.showMessage(i18n.T("error_no_url"))
}

func (m *Model) openDashboard() {
	if err := m.App.OpenDashboard(); err != nil {
		m.showMessage(i18n.TF("error_dashboard", err.Error()))
		return
	}
	m.showMessage(i18n.T("dashboard_opened"))
}

func (m *Model) openConfigDir() {
	if err := m.App.OpenConfigDir(); err != nil {
		m.showMessage(i18n.TF("error_config", err.Error()))
		return
	}
	m.showMessage(i18n.T("config_opened"))
}

func (m *Model) handleListUpdate() (tea.Model, tea.Cmd) {
	if m.UpdateAvailable == nil {
		return m, nil
	}
	if m.UpdateAvailable.Method != updater.MethodSelf {
		m.showMessage(i18n.T(m.UpdateAvailable.Method.HintKey()))
		return m, nil
	}
	m.showMessage(i18n.T("update_applying"))
	return m, m.applyUpdateCmd()
}

func (m *Model) playBeep() {
	if m.App.Config().NotificationSound {
		_ = notifications.PlaySound(notifications.SoundInfo)
	}
}
