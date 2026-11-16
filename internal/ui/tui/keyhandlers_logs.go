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

	var lines []string
	if m.SelectedTunnel == m.logsID {
		lines = m.logs
	}
	m.LogViewport.SetContent(strings.Join(lines, "\n"))

	if following {
		m.LogViewport.GotoBottom()
	}
}
