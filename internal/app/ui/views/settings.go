package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sthbryan/ftm/internal/app/ui"
	"github.com/sthbryan/ftm/internal/i18n"
)

type SettingsView struct {
	Width                int
	NotificationsEnabled bool
	NotificationSound    bool
	Language             string
	Focused              int
}

func NewSettingsView() *SettingsView {
	return &SettingsView{
		Focused:  0,
		Language: i18n.GetCurrentLang(),
	}
}

func (s *SettingsView) Render() string {
	t := ui.ThemeDefault
	var b strings.Builder

	header := lipgloss.NewStyle().
		Foreground(t.Gold).
		Bold(true).
		Render(i18n.T("settings_title"))

	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 30))
	b.WriteString("\n\n")

	b.WriteString(s.renderToggle(
		i18n.T("enable_notifications"),
		s.NotificationsEnabled,
		s.Focused == 0,
		t,
	))
	b.WriteString("\n")

	b.WriteString(s.renderToggle(
		i18n.T("notification_sound"),
		s.NotificationSound,
		s.Focused == 1,
		t,
	))
	b.WriteString("\n\n")

	b.WriteString(s.renderLanguageSelector(t))

	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().
		Foreground(t.TextDim).
		Render(i18n.T("settings_nav_hint")))

	return b.String()
}

func (s *SettingsView) renderLanguageSelector(t *ui.Theme) string {
	var b strings.Builder

	label := i18n.T("language") + ":"

	if s.Focused == 2 {
		b.WriteString(lipgloss.NewStyle().Foreground(t.Gold).Render("▸ "))
	} else {
		b.WriteString("  ")
	}

	b.WriteString(label)
	b.WriteString(" ")

	for _, lang := range i18n.SupportedLanguages() {
		langName := i18n.LanguageName(lang)
		if lang == s.Language {
			b.WriteString(lipgloss.NewStyle().Foreground(t.Gold).Bold(true).Render("[" + langName + "]"))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(t.TextDim).Render("[" + langName + "]"))
		}
		b.WriteString(" ")
	}

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
		icon = "[✓]"
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
