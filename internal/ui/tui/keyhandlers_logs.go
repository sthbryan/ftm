package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m *Model) handleLogsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.LogViewport, cmd = m.LogViewport.Update(msg)
	return m, cmd
}

func (m *Model) updateLogViewport() {
	if m.SelectedTunnel == "" {
		return
	}

	following := m.LogViewport.AtBottom()

	m.LogViewport.SetContent(strings.Join(m.App.Manager.GetLogs(m.SelectedTunnel), "\n"))

	if following {
		m.LogViewport.GotoBottom()
	}
}
