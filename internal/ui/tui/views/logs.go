package views

import (
	"charm.land/lipgloss/v2"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/ui/tui/ui"
)

type LogsView struct {
	Width      int
	Height     int
	TunnelName string
	Content    string
}

func NewLogsView() *LogsView {
	return &LogsView{}
}

const (
	logsWidthRatio  = 0.8
	logsHeightRatio = 0.75
)

func LogsBox(width, height int) (int, int) {
	return ui.Clamp(int(float64(width)*logsWidthRatio), 20),
		ui.Clamp(int(float64(height)*logsHeightRatio), minBodyHeight+2)
}

func (l *LogsView) Render() string {
	t := ui.ThemeDefault

	footer := lipgloss.NewStyle().
		Foreground(t.TextDim).
		Width(l.Width).
		Align(lipgloss.Center).
		Render(i18n.T("logs_nav_hint"))

	title := i18n.T("tunnel_logs")
	if l.TunnelName != "" {
		title += "  •  " + l.TunnelName
	}

	body := lipgloss.NewStyle().
		Foreground(t.Text).
		Render(l.Content)

	panel := ui.Panel(title, body, l.Width, l.Height-lipgloss.Height(footer), t.Gold)

	return panel + "\n" + footer
}
