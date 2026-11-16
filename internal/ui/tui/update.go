package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/provider"
	"github.com/sthbryan/ftm/internal/ui/tui/ui"
)

// Update handles msg, then follows the output of the tunnel now on screen and
// schedules clearing a new status message.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.update(msg)
	return model, tea.Batch(cmd, m.followSelection(), m.expireMessageCmd())
}

func (m *Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.BackgroundColorMsg:
		ui.SetDarkBackground(msg.IsDark())
		return m, nil

	case messageExpiredMsg:
		if msg.seq == m.messageSeq {
			m.Message = ""
		}
		return m, nil

	case busMsg:
		model, cmd := m.update(msg.msg)
		return model, tea.Batch(cmd, m.waitForEvent())

	case tunnelsChangedMsg:
		m.refreshItems()
		return m, nil

	case prefsChangedMsg:
		m.refreshItems()
		m.refreshPrefs()
		return m, nil

	case connectionMsg:
		return m.handleConnection(msg)

	case logsLoadedMsg:
		if msg.tunnelID == m.logsID {
			m.logs = mergeLogs(msg.lines, m.logs)
		}
		return m, nil

	case logLineMsg:
		if msg.tunnelID == m.logsID {
			m.appendLog(msg.line)
		}
		return m, nil

	case installStartedMsg:
		m.DownloadingProvider = string(msg.provider)
		m.DownloadProgress = provider.DownloadProgress{}
		m.State = viewDownloading
		return m, nil

	case downloadProgressMsg:
		return m.handleDownloadProgress(msg)

	case statusUpdateMsg:
		return m.handleStatusUpdate(msg)

	case updateCheckMsg:
		return m.handleUpdateCheck(msg)

	case updateApplyMsg:
		return m.handleUpdateApply(msg)

	case providerSummaryMsg:
		return m.handleProviderSummary(msg)
	}

	return m, nil
}

// handleConnection tells the user when the running ftm the TUI is attached
// to goes away and when it is back.
func (m *Model) handleConnection(msg connectionMsg) (tea.Model, tea.Cmd) {
	if msg.lost == m.lost {
		return m, nil
	}
	m.lost = msg.lost
	if msg.lost {
		m.showMessage(i18n.T("tui_connection_lost"))
	} else {
		m.showMessage(i18n.T("tui_connection_back"))
	}
	return m, nil
}

// handleDownloadProgress follows a provider download. The service starts the
// waiting tunnel itself once it finishes; the model only leaves the progress
// screen if it is still on it.
func (m *Model) handleDownloadProgress(msg downloadProgressMsg) (tea.Model, tea.Cmd) {
	m.DownloadProgress = provider.DownloadProgress(msg)

	failed := m.DownloadProgress.Error != nil
	if !m.DownloadProgress.Done && !failed {
		return m, nil
	}
	if m.State != viewDownloading {
		return m, nil
	}

	m.State = viewList
	if !failed {
		m.showMessage(i18n.T("install_complete"))
	}
	return m, nil
}

func (m *Model) handleStatusUpdate(msg statusUpdateMsg) (tea.Model, tea.Cmd) {
	if !m.applyStatus(msg.status) {
		m.refreshItems()
	}
	if msg.status.ErrorMessage != "" {
		if m.State == viewDownloading {
			m.State = viewList
		}
		m.playBeep()
		m.showMessage(i18n.TF("error_state", msg.status.ErrorMessage))
	}
	return m, nil
}

func (m *Model) handleUpdateCheck(msg updateCheckMsg) (tea.Model, tea.Cmd) {
	if msg.err == nil && msg.info != nil && msg.info.HasUpdate {
		m.UpdateAvailable = msg.info
	}
	return m, nil
}

func (m *Model) handleUpdateApply(msg updateApplyMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.showMessage(i18n.TF("update_apply_failed", msg.err.Error()))
		return m, nil
	}
	if m.App.Attached() {
		// The running ftm restarts on its own; the TUI reconnects to it.
		return m, nil
	}
	m.RelaunchOnExit = true
	return m, tea.Quit
}

// applyStatus puts a state change on the connection it belongs to, without
// reading the list again. It reports false when the connection is not listed
// yet.
func (m *Model) applyStatus(status core.TunnelStatus) bool {
	if status.ID == "" {
		return false
	}
	for i, item := range m.Items {
		ti, ok := item.(TunnelItem)
		if !ok || ti.Tunnel.ID != status.ID {
			continue
		}
		if status.Name != "" {
			ti.Tunnel.Name = status.Name
		}
		if status.Provider != "" {
			ti.Tunnel.Provider = status.Provider
		}
		if status.LocalPort != 0 {
			ti.Tunnel.LocalPort = status.LocalPort
		}
		ti.Status = status
		ti.Running = running(status.State)
		m.Items[i] = ti
		return true
	}
	return false
}
