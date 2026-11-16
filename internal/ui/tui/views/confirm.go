package views

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/ui/tui/ui"
)

type ConfirmView struct {
	Title   string
	Message string
	Danger  bool
}

func NewConfirmView() *ConfirmView {
	return &ConfirmView{}
}

const confirmWidth = 46

func (c *ConfirmView) Render() string {
	t := ui.ThemeDefault

	accent := t.Accent
	if c.Danger {
		accent = t.Danger
	}

	surface := lipgloss.NewStyle().Background(t.Surface).Width(confirmWidth)

	title := surface.
		Foreground(accent).
		Bold(true).
		Render(c.Title)

	message := surface.
		Foreground(t.Text).
		Render(c.Message)

	confirm := lipgloss.NewStyle().
		Background(t.ButtonActive).
		Foreground(t.ButtonActiveText).
		Bold(true).
		Padding(0, 2).
		Render(i18n.T("confirm_yes"))

	cancel := lipgloss.NewStyle().
		Background(t.Button).
		Foreground(t.ButtonText).
		Padding(0, 2).
		Render(i18n.T("confirm_no"))

	gap := lipgloss.NewStyle().Background(t.Surface).Render("  ")

	actions := surface.
		Align(lipgloss.Right).
		Render(confirm + gap + cancel)

	body := strings.Join([]string{title, surface.Render(""), message, surface.Render(""), actions}, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(accent).
		BorderBackground(t.Surface).
		Background(t.Surface).
		Padding(1, 2).
		Render(body)
}
