package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sthbryan/ftm/internal/app/ui"
	"github.com/sthbryan/ftm/internal/version"
)

type EmptyState struct {
	Height    int
	Width     int
	Dashboard string
}

func NewEmptyState() *EmptyState {
	return &EmptyState{}
}

func (e *EmptyState) Render() string {
	var b strings.Builder

	bg := ui.ThemeDefault.Bg
	gold := ui.ThemeDefault.Gold
	bronze := ui.ThemeDefault.Bronze
	text := ui.ThemeDefault.Text
	textDim := ui.ThemeDefault.TextDim

	title := lipgloss.NewStyle().
		Foreground(gold).
		Bold(true).
		Render("Welcome, Dungeon Master!")

	subtitle := lipgloss.NewStyle().
		Foreground(text).
		Render("You haven't created any tunnels yet.")

	desc := lipgloss.NewStyle().
		Foreground(textDim).
		Render("Tunnels let your players connect to your Foundry world.")

	cta := lipgloss.NewStyle().
		Background(gold).
		Foreground(bg).
		Bold(true).
		Padding(0, 2).
		Render("[ Create First Tunnel ]")

	hint := lipgloss.NewStyle().
		Foreground(textDim).
		Render("Or press 'a' to start")

	tip := lipgloss.NewStyle().
		Foreground(bronze).
		Render("💡 Tip: Web dashboard at " + e.Dashboard)

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
