package views

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/ui/tui/ui"
)

type SettingsView struct {
	NotificationsEnabled bool
	NotificationSound    bool
	Language             string
	TunnelCount          int
	Focused              int
	// Providers is a read-only summary of the downloaded provider binaries,
	// filled in once it has loaded.
	Providers string
}

const SettingsRows = 4

func NewSettingsView() *SettingsView {
	return &SettingsView{
		Focused:  0,
		Language: i18n.GetCurrentLang(),
	}
}

const settingsWidth = 56

func (s *SettingsView) Render() string {
	t := ui.ThemeDefault
	inner := ui.PanelInner(settingsWidth)

	body := strings.Join([]string{
		s.toggle(inner, i18n.T("enable_notifications"), s.NotificationsEnabled, s.Focused == 0),
		s.toggle(inner, i18n.T("notification_sound"), s.NotificationSound, s.Focused == 1),
		"",
		s.languages(inner),
		"",
		s.clearAll(inner),
		"",
		s.providers(inner),
		"",
		lipgloss.NewStyle().
			Foreground(t.TextDim).
			Width(inner).
			Align(lipgloss.Center).
			Render(i18n.T("settings_nav_hint")),
	}, "\n")

	return ui.Panel(i18n.T("settings_title"), body, settingsWidth, lipgloss.Height(body)+ui.PanelChrome, t.Gold)
}

func (s *SettingsView) marker(focused bool) string {
	if focused {
		return lipgloss.NewStyle().Foreground(ui.ThemeDefault.Gold).Bold(true).Render("▸ ")
	}
	return "  "
}

func (s *SettingsView) toggle(width int, label string, enabled, focused bool) string {
	t := ui.ThemeDefault

	state := lipgloss.NewStyle().
		Background(t.Button).
		Foreground(t.ButtonText).
		Padding(0, 1).
		Render(i18n.T("toggle_off"))

	if enabled {
		state = lipgloss.NewStyle().
			Background(t.Success).
			Foreground(t.ButtonActiveText).
			Bold(true).
			Padding(0, 1).
			Render(i18n.T("toggle_on"))
	}

	labelStyle := lipgloss.NewStyle().Foreground(t.Text)
	if focused {
		labelStyle = labelStyle.Foreground(t.Gold).Bold(true)
	}

	marker := s.marker(focused)
	gap := ui.Clamp(width-lipgloss.Width(marker)-lipgloss.Width(label)-lipgloss.Width(state), 1)

	return marker + labelStyle.Render(label) + ui.Repeat(" ", gap) + state
}

func (s *SettingsView) clearAll(width int) string {
	t := ui.ThemeDefault
	focused := s.Focused == 3

	labelStyle := lipgloss.NewStyle().Foreground(t.TextDim)
	if s.TunnelCount > 0 {
		labelStyle = lipgloss.NewStyle().Foreground(t.Danger)
		if focused {
			labelStyle = labelStyle.Bold(true)
		}
	}

	label := i18n.T("reset_connections")
	count := lipgloss.NewStyle().Foreground(t.TextDim).Render(fmt.Sprintf("%d", s.TunnelCount))

	marker := s.marker(focused)
	gap := ui.Clamp(width-lipgloss.Width(marker)-lipgloss.Width(label)-lipgloss.Width(count), 1)

	return marker + labelStyle.Render(label) + ui.Repeat(" ", gap) + count
}

func (s *SettingsView) languages(width int) string {
	t := ui.ThemeDefault

	options := make([]string, 0, 4)
	for _, lang := range i18n.SupportedLanguages() {
		style := lipgloss.NewStyle().Foreground(t.TextDim)
		if lang == s.Language {
			style = lipgloss.NewStyle().
				Background(t.ButtonActive).
				Foreground(t.ButtonActiveText).
				Bold(true)
		}
		options = append(options, style.Padding(0, 1).Render(i18n.LanguageName(lang)))
	}

	labelStyle := lipgloss.NewStyle().Foreground(t.Text)
	if s.Focused == 2 {
		labelStyle = labelStyle.Foreground(t.Gold).Bold(true)
	}

	label := i18n.T("language")
	values := strings.Join(options, " ")

	marker := s.marker(s.Focused == 2)
	gap := ui.Clamp(width-lipgloss.Width(marker)-lipgloss.Width(label)-lipgloss.Width(values), 1)

	return marker + labelStyle.Render(label) + ui.Repeat(" ", gap) + values
}

func (s *SettingsView) providers(width int) string {
	t := ui.ThemeDefault

	value := s.Providers
	if value == "" {
		value = "…"
	}

	label := lipgloss.NewStyle().Foreground(t.Text).Render(i18n.T("providers_settings_label") + ": ")
	return lipgloss.NewStyle().
		Foreground(t.TextDim).
		Width(width).
		Render("  " + label + value)
}
