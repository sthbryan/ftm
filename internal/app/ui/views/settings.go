package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sthbryan/ftm/internal/app/ui"
)

type SettingsView struct {
	Width                int
	NotificationsEnabled bool
	NotificationSound    bool
	Focused              int
}

func NewSettingsView() *SettingsView {
	return &SettingsView{
		Focused: 0,
	}
}

func (s *SettingsView) Render() string {
	t := ui.ThemeDefault
	var b strings.Builder

	header := lipgloss.NewStyle().
		Foreground(t.Gold).
		Bold(true).
		Render("⚙ Settings")

	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 30))
	b.WriteString("\n\n")

	b.WriteString(s.renderToggle(
		"Enable Notifications",
		s.NotificationsEnabled,
		s.Focused == 0,
		t,
	))
	b.WriteString("\n")

	b.WriteString(s.renderToggle(
		"Sound Effects",
		s.NotificationSound,
		s.Focused == 1,
		t,
	))

	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().
		Foreground(t.TextDim).
		Render("↑/↓ navigate  •  space toggle  •  esc back"))

	return b.String()
}

func (s *SettingsView) renderToggle(label string, enabled bool, focused bool, t *ui.Theme) string {
	var b strings.Builder

	if focused {
		b.WriteString(lipgloss.NewStyle().Foreground(t.Gold).Render("▸ "))
	} else {
		b.WriteString("  ")
	}

	icon := "[ ]"
	if enabled {
		icon = "✓"
	}

	iconStyle := lipgloss.NewStyle().Foreground(t.Success).Bold(true)
	if focused {
		iconStyle = iconStyle.Underline(true)
	}

	b.WriteString(iconStyle.Render(icon))
	b.WriteString(" ")
	b.WriteString(label)

	return b.String()
}
