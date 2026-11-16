package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/ui/tui/ui"
)

type DetailPanel struct {
	Name           string
	Provider       string
	LocalPort      int
	StatusState    int
	PublicURL      string
	Visitors       int
	ActiveSessions int
	ErrorMsg       string
	Logs           []string
	Width          int
	Height         int
}

func NewDetailPanel() *DetailPanel {
	return &DetailPanel{}
}

const minLogTail = 2

func (d *DetailPanel) Render() string {
	details := d.details()

	tail := d.logTail(ui.Clamp(d.Height-lipgloss.Height(details)-1, 0))
	if tail == "" {
		return details
	}

	return details + "\n\n" + tail
}

func (d *DetailPanel) details() string {
	var b strings.Builder

	nameStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.ThemeDefault.Gold).
		Width(d.Width)
	b.WriteString(nameStyle.Render(d.Name))
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().Foreground(ui.ThemeDefault.TextDim)
	textStyle := lipgloss.NewStyle().Foreground(ui.ThemeDefault.Text)

	b.WriteString(labelStyle.Render(i18n.T("provider_label") + ":"))
	b.WriteString(" ")
	b.WriteString(textStyle.Render(i18n.ProviderText(d.Provider)))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render(i18n.T("port_label") + ":"))
	b.WriteString(" ")
	b.WriteString(textStyle.Render(fmt.Sprintf(":%d", d.LocalPort)))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render(i18n.T("status_label") + ":"))
	b.WriteString(" ")
	b.WriteString(lipgloss.NewStyle().
		Foreground(StatusColor(d.StatusState)).
		Bold(true).
		Render(StatusLabel(d.StatusState)))
	b.WriteString("\n\n")

	if d.StatusState == TunnelStateOnline && d.PublicURL != "" {
		b.WriteString(labelStyle.Render(i18n.T("url_label")))
		b.WriteString("\n")

		urlBox := lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ui.ThemeDefault.Gold).
			Foreground(ui.ThemeDefault.Text).
			Padding(0, 1).
			Width(d.Width).
			Render(ui.Truncate(d.PublicURL, d.Width-4))
		b.WriteString(urlBox)
		b.WriteString("\n")

		copyHint := lipgloss.NewStyle().
			Foreground(ui.ThemeDefault.Bronze).
			Render(i18n.T("press_c_copy"))
		b.WriteString(copyHint)
		b.WriteString("\n\n")
	}

	if d.StatusState == TunnelStateOnline {
		b.WriteString(labelStyle.Render(i18n.T("detail_sessions") + ":"))
		b.WriteString(" ")
		b.WriteString(textStyle.Render(fmt.Sprintf("%d", d.ActiveSessions)))
		b.WriteString("\n")

		b.WriteString(labelStyle.Render(i18n.T("detail_visitors") + ":"))
		b.WriteString(" ")
		b.WriteString(textStyle.Render(fmt.Sprintf("%d", d.Visitors)))
		b.WriteString("\n\n")
	}

	if d.ErrorMsg != "" {
		b.WriteString(labelStyle.Render(i18n.T("error_label")))
		b.WriteString("\n")

		errorBox := lipgloss.NewStyle().
			Foreground(ui.ThemeDefault.Danger).
			Width(d.Width).
			Render(d.ErrorMsg)
		b.WriteString(errorBox)
		b.WriteString("\n\n")
	}

	b.WriteString(d.actions())

	return b.String()
}

func (d *DetailPanel) logTail(rows int) string {
	if len(d.Logs) == 0 || rows < minLogTail+1 {
		return ""
	}

	lines := d.Logs
	if tail := rows - 1; len(lines) > tail {
		lines = lines[len(lines)-tail:]
	}

	header := lipgloss.NewStyle().
		Foreground(ui.ThemeDefault.TextDim).
		Render(i18n.T("recent_activity"))

	body := make([]string, 0, len(lines))
	for _, line := range lines {
		body = append(body, lipgloss.NewStyle().
			Foreground(ui.ThemeDefault.Bronze).
			Render(ui.Truncate(line, d.Width)))
	}

	return header + "\n" + strings.Join(body, "\n")
}

func (d *DetailPanel) actions() string {
	var actions []string

	isActive := d.StatusState == TunnelStateOnline ||
		d.StatusState == TunnelStateStarting ||
		d.StatusState == TunnelStateConnecting

	buttonStyle := lipgloss.NewStyle().
		Background(ui.ThemeDefault.Button).
		Foreground(ui.ThemeDefault.ButtonText).
		Bold(true).
		Padding(0, 2)

	if isActive {
		actions = append(actions, buttonStyle.Render("[t] "+i18n.T("stop_action")))
	} else {
		actions = append(actions, buttonStyle.Render("[t] "+i18n.T("start_action")))
	}

	actions = append(actions, buttonStyle.Render("[l] "+i18n.T("logs_action")))
	actions = append(actions, buttonStyle.Render("[d] "+i18n.T("delete_action")))

	return strings.Join(actions, "  ")
}
