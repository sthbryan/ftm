package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sthbryan/ftm/internal/app/ui"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/version"
)

type EmptyState struct {
	Height    int
	Width     int
	Dashboard string
	Sessions  int
}

func NewEmptyState() *EmptyState {
	return &EmptyState{}
}

func (e *EmptyState) Render() string {
	var b strings.Builder

	gold := ui.ThemeDefault.Gold
	bronze := ui.ThemeDefault.Bronze
	text := ui.ThemeDefault.Text
	textDim := ui.ThemeDefault.TextDim

	title := lipgloss.NewStyle().
		Foreground(gold).
		Bold(true).
		Render(i18n.T("welcome_title"))

	subtitle := lipgloss.NewStyle().
		Foreground(text).
		Render(i18n.T("no_tunnels_yet"))

	desc := lipgloss.NewStyle().
		Foreground(textDim).
		Render(i18n.T("tunnels_desc"))

	cta := lipgloss.NewStyle().
		Background(gold).
		Bold(true).
		Padding(0, 2).
		Render(i18n.T("create_first"))

	hint := lipgloss.NewStyle().
		Foreground(textDim).
		Render(i18n.T("press_a_hint"))

	tip := lipgloss.NewStyle().
		Foreground(bronze).
		Render(i18n.T("tip_dashboard") + " " + e.Dashboard + "  •  ws:" + fmt.Sprintf("%d", e.Sessions))

	contentHeight := 12
	paddingTop := (e.Height - contentHeight) / 2
	if paddingTop < 2 {
		paddingTop = 2
	}

	b.WriteString(strings.Repeat("\n", paddingTop))
	b.WriteString(ui.Center(title, e.Width) + "\n\n")
	b.WriteString(ui.Center(subtitle, e.Width) + "\n")
	b.WriteString(ui.Center(desc, e.Width) + "\n\n")
	b.WriteString(ui.Center(cta, e.Width) + "\n\n")
	b.WriteString(ui.Center(hint, e.Width) + "\n\n")
	b.WriteString(ui.Center(tip, e.Width) + "\n")

	_ = version.Version

	return b.String()
}
