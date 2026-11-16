package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (m *Model) entersText() bool {
	return m.State == viewNewTunnel || m.State == viewEditTunnel
}

func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.Keys.ForceQuit) {
		return m, tea.Quit
	}

	if m.entersText() {
		if key.Matches(msg, m.Keys.Back) {
			return m.handleBack()
		}
		return m.handleEditorKey(msg)
	}

	if m.State == viewConfirm {
		return m.handleConfirmKey(msg)
	}

	switch {
	case key.Matches(msg, m.Keys.Quit):
		return m.handleQuit()

	case key.Matches(msg, m.Keys.Back):
		return m.handleBack()

	case key.Matches(msg, m.Keys.Help):
		m.Help.ShowAll = !m.Help.ShowAll
		return m, nil
	}

	switch m.State {
	case viewList:
		return m.handleListKey(msg)
	case viewLogs:
		return m.handleLogsKey(msg)
	case viewDownloading:
		return m.handleDownloadingKey(msg)
	case viewSettings:
		return m.handleSettingsKey(msg)
	}

	return m, nil
}

func (m *Model) handleQuit() (tea.Model, tea.Cmd) {
	if m.State != viewList {
		return m.handleBack()
	}
	return m, tea.Quit
}

func (m *Model) handleBack() (tea.Model, tea.Cmd) {
	if m.State == viewConfirm {
		m.cancelConfirm()
		return m, nil
	}

	if m.State == viewSettings {
		m.saveSettings()
	}
	if m.State != viewList {
		m.State = viewList
		m.editingTunnelID = ""
	}
	return m, nil
}

func (m *Model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.Keys.Back) {
		m.cancelConfirm()
		return m, nil
	}

	switch msg.String() {
	case "y", "Y", "enter":
		if m.pendingConfirm == confirmClearTunnels {
			return m.confirmClearTunnels()
		}
		return m.confirmDelete()
	case "n", "N", "q":
		m.cancelConfirm()
	}

	return m, nil
}

func (m *Model) handleDownloadingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.Keys.Back) || key.Matches(msg, m.Keys.Quit) {
		m.State = viewList
	}
	return m, nil
}
