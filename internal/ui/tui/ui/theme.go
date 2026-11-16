package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type Theme struct {
	Accent      color.Color
	AccentMuted color.Color
	Text        color.Color
	TextDim     color.Color
	Surface     color.Color
	Dimmed      color.Color
	Online      color.Color
	Offline     color.Color
	Connecting  color.Color
	Error       color.Color
	Danger      color.Color
	Stopped     color.Color
	Success     color.Color

	Button           color.Color
	ButtonText       color.Color
	ButtonActive     color.Color
	ButtonActiveText color.Color
}

// BuildTheme is Tinta, the ftm theme the web view and desktop app use, for a
// dark or light terminal. Surfaces are a quiet graphite; the only colour is
// ink blue, kept for what matters and the public link. Row fills are the
// status colours mixed into the background, the way the web view tints them.
func BuildTheme(isDark bool) *Theme {
	c := lipgloss.LightDark(isDark)

	return &Theme{
		Accent:      c(lipgloss.Color("#3a5bc7"), lipgloss.Color("#9eb1ff")),
		AccentMuted: c(lipgloss.Color("#3358d4"), lipgloss.Color("#5472e4")),
		Text:        c(lipgloss.Color("#21201c"), lipgloss.Color("#eeeeec")),
		TextDim:     c(lipgloss.Color("#63635e"), lipgloss.Color("#b5b3ad")),
		Surface:     c(lipgloss.Color("#f1f0ef"), lipgloss.Color("#222221")),
		Dimmed:      c(lipgloss.Color("#cfceca"), lipgloss.Color("#494844")),
		Online:      c(lipgloss.Color("#e4eee5"), lipgloss.Color("#1e2c20")),
		Offline:     c(lipgloss.Color("#f1f0ef"), lipgloss.Color("#191918")),
		Connecting:  c(lipgloss.Color("#f1e9de"), lipgloss.Color("#322b11")),
		Error:       c(lipgloss.Color("#f8e8e8"), lipgloss.Color("#322422")),
		Danger:      c(lipgloss.Color("#ce2c31"), lipgloss.Color("#ff9592")),
		Stopped:     c(lipgloss.Color("#e9e8e6"), lipgloss.Color("#2a2a28")),
		Success:     c(lipgloss.Color("#2a7e3b"), lipgloss.Color("#71d083")),

		Button:           c(lipgloss.Color("#e9e8e6"), lipgloss.Color("#2a2a28")),
		ButtonText:       c(lipgloss.Color("#21201c"), lipgloss.Color("#eeeeec")),
		ButtonActive:     c(lipgloss.Color("#3e63dd"), lipgloss.Color("#3e63dd")),
		ButtonActiveText: c(lipgloss.Color("#ffffff"), lipgloss.Color("#ffffff")),
	}
}

func DefaultTheme() *Theme {
	return BuildTheme(true)
}

var ThemeDefault = DefaultTheme()

func SetDarkBackground(isDark bool) {
	ThemeDefault = BuildTheme(isDark)
}
