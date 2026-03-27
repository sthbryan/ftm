package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sthbryan/ftm/internal/app/ui"
)

type LogsView struct {
	Width      int
	TunnelName string
	Content    string
}

func NewLogsView() *LogsView {
	return &LogsView{}
}

func (l *LogsView) Render() string {
	var b strings.Builder

	gold := ui.ThemeDefault.Gold
	bronze := ui.ThemeDefault.Bronze
	text := ui.ThemeDefault.Text
	textDim := ui.ThemeDefault.TextDim

	header := lipgloss.NewStyle().
		Foreground(gold).
		Bold(true).
		Render("📋  Tunnel Logs")

	b.WriteString(header)
	b.WriteString("\n\n")

	nameStyle := lipgloss.NewStyle().Foreground(text).Bold(true)
	b.WriteString(nameStyle.Render(l.TunnelName))
	b.WriteString("\n")

	dividerStyle := lipgloss.NewStyle().Foreground(bronze)
	b.WriteString(dividerStyle.Render(strings.Repeat("─", l.Width-2)))
	b.WriteString("\n")

	b.WriteString(l.Content)
	b.WriteString("\n")

	b.WriteString(lipgloss.NewStyle().
		Foreground(textDim).
		Render("esc/b: back • ↑/↓: scroll"))

	return b.String()
}
