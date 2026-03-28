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
	Theme                string
	Focused              int
}

func NewSettingsView() *SettingsView {
	return &SettingsView{
		Theme:    "system",
		Focused:  0,
	}
}

func (s *SettingsView) Render() string {
	t := ui.ThemeDefault
	var b strings.Builder

	header := lipgloss.NewStyle().
		Foreground(t.Gold).
		Bold(true).
		Render("⚙️  Settings")

	subheader := lipgloss.NewStyle().
		Foreground(t.TextDim).
		Render("Configure your preferences")

	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 30))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().Foreground(t.Text).Bold(true).Render("Notifications"))
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

	b.WriteString(lipgloss.NewStyle().Foreground(t.Text).Bold(true).Render("Appearance"))
	b.WriteString("\n\n")

	themes := []string{"light", "dark", "system"}
	themeLabels := []string{"Light", "Dark", "System"}
	for i, theme := range themes {
		b.WriteString(s.renderOption(
			themeLabels[i],
			s.Theme == theme,
			s.Focused == i+2,
			t,
		))
		if i < len(themes)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(t.TextDim).Render("j/k or ↑/↓: navigate  |  space/enter: toggle/select  |  esc: back"))

	return b.String()
}

func (s *SettingsView) renderToggle(label string, enabled bool, focused bool, t *ui.Theme) string {
	var b strings.Builder

	if focused {
		b.WriteString(lipgloss.NewStyle().Foreground(t.Gold).Render(" ▸ "))
	} else {
		b.WriteString("   ")
	}

	icon := "○"
	if enabled {
		icon = "●"
	}

	iconStyle := lipgloss.NewStyle().Foreground(t.Success)
	if focused {
		iconStyle = iconStyle.Bold(true)
	}

	b.WriteString(iconStyle.Render(icon))
	b.WriteString(" ")
	b.WriteString(label)

	if focused {
		b.WriteString(lipgloss.NewStyle().Foreground(t.TextDim).Render(" [space to toggle]"))
	}

	return b.String()
}

func (s *SettingsView) renderOption(label string, selected bool, focused bool, t *ui.Theme) string {
	var b strings.Builder

	if focused {
		b.WriteString(lipgloss.NewStyle().Foreground(t.Gold).Render(" ▸ "))
	} else {
		b.WriteString("   ")
	}

	icon := "○"
	if selected {
		icon = "●"
	}

	iconStyle := lipgloss.NewStyle().Foreground(t.Gold)
	if selected {
		iconStyle = iconStyle.Bold(true)
	}

	b.WriteString(iconStyle.Render(icon))
	b.WriteString(" ")
	b.WriteString(label)

	if focused {
		b.WriteString(lipgloss.NewStyle().Foreground(t.TextDim).Render(" [select]"))
	}

	return b.String()
}
