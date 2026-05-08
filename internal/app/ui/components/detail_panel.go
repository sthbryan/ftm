package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sthbryan/ftm/internal/app/ui"
	"github.com/sthbryan/ftm/internal/i18n"
)

type DetailPanel struct {
	Name        string
	Provider    string
	LocalPort   int
	StatusState int
	StatusMsg   string
	PublicURL   string
	ErrorMsg    string
	Width       int
}

func NewDetailPanel() *DetailPanel {
	return &DetailPanel{}
}

func (d *DetailPanel) Render() string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().
		Render(""))

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
	b.WriteString(textStyle.Render(StatusLabel(d.StatusState)))
	b.WriteString("\n\n")

	if d.StatusState == TunnelStateOnline && d.PublicURL != "" {
		b.WriteString(labelStyle.Render("URL:"))
		b.WriteString("\n")

		urlBox := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ui.ThemeDefault.Gold).
			Foreground(ui.ThemeDefault.Text).
			Padding(0, 1).
			Width(d.Width - 2).
			Render(d.PublicURL)
		b.WriteString(urlBox)
		b.WriteString("\n")

		copyHint := lipgloss.NewStyle().
			Foreground(ui.ThemeDefault.Bronze).
			Render(i18n.T("press_c_copy"))
		b.WriteString(copyHint)
		b.WriteString("\n\n")
	}

	if d.ErrorMsg != "" {
		b.WriteString(labelStyle.Render("Error:"))
		b.WriteString("\n")

		errorBox := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff6b6b")).
			Width(d.Width - 2).
			Render(d.ErrorMsg)
		b.WriteString(errorBox)
		b.WriteString("\n\n")
	}

	b.WriteString(d.actions())

	return b.String()
}

func (d *DetailPanel) actions() string {
	var actions []string

	isActive := d.StatusState == TunnelStateOnline ||
		d.StatusState == TunnelStateStarting ||
		d.StatusState == TunnelStateConnecting

	buttonStyle := lipgloss.NewStyle().
		Background(ui.ThemeDefault.Bronze).
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
