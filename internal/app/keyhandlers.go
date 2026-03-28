package app

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case viewAddForm, viewEditForm:
		return m.handleFormKey(msg)
	case viewDownloading:
		return m.handleDownloadingKey(msg)
	case viewSettings:
		return m.handleSettingsKey(msg)
	}

	return m, nil
}

func (m *Model) handleQuit() (tea.Model, tea.Cmd) {
	if m.State == viewList {
		return m, tea.Quit
	}
	m.State = viewList
	return m, nil
}

func (m *Model) handleBack() (tea.Model, tea.Cmd) {
	if m.State == viewSettings {
		m.saveSettings()
	}
	if m.State != viewList {
		m.State = viewList
		m.editingTunnelID = ""
	}
	return m, nil
}

func (m *Model) handleDownloadingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.Keys.Back) || key.Matches(msg, m.Keys.Quit) {
		m.State = viewList
	}
	return m, nil
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
