package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type Theme struct {
	Gold       color.Color
	Bronze     color.Color
	Text       color.Color
	TextDim    color.Color
	Surface    color.Color
	Dimmed     color.Color
	Online     color.Color
	Offline    color.Color
	Connecting color.Color
	Error      color.Color
	Danger     color.Color
	Stopped    color.Color
	Success    color.Color

	Button           color.Color
	ButtonText       color.Color
	ButtonActive     color.Color
	ButtonActiveText color.Color
}

func BuildTheme(isDark bool) *Theme {
	c := lipgloss.LightDark(isDark)

	return &Theme{
		Gold:       c(lipgloss.Color("#8a6d1f"), lipgloss.Color("#c9a227")),
		Bronze:     c(lipgloss.Color("#6b5940"), lipgloss.Color("#8b7355")),
		Text:       c(lipgloss.Color("#1c1a17"), lipgloss.Color("#ffffff")),
		TextDim:    c(lipgloss.Color("#6b6660"), lipgloss.Color("#9a9590")),
		Surface:    c(lipgloss.Color("#f2eee6"), lipgloss.Color("#232019")),
		Dimmed:     c(lipgloss.Color("#bdb8ae"), lipgloss.Color("#4a453d")),
		Online:     c(lipgloss.Color("#cfe8da"), lipgloss.Color("#1e3a2f")),
		Offline:    c(lipgloss.Color("#e8e6e1"), lipgloss.Color("#2a2824")),
		Connecting: c(lipgloss.Color("#f3e6c2"), lipgloss.Color("#3a3020")),
		Error:      c(lipgloss.Color("#f5d6d6"), lipgloss.Color("#3a2020")),
		Danger:     c(lipgloss.Color("#b02a1c"), lipgloss.Color("#ff6b6b")),
		Stopped:    c(lipgloss.Color("#dedbd6"), lipgloss.Color("#3a3a3a")),
		Success:    c(lipgloss.Color("#2f7d5b"), lipgloss.Color("#7cb69d")),

		Button:           c(lipgloss.Color("#5a4a34"), lipgloss.Color("#8b7355")),
		ButtonText:       c(lipgloss.Color("#f6f3ec"), lipgloss.Color("#14120f")),
		ButtonActive:     c(lipgloss.Color("#8a6d1f"), lipgloss.Color("#c9a227")),
		ButtonActiveText: c(lipgloss.Color("#fff8e1"), lipgloss.Color("#1c1a17")),
	}
}

func DefaultTheme() *Theme {
	return BuildTheme(true)
}

var ThemeDefault = DefaultTheme()

func SetDarkBackground(isDark bool) {
	ThemeDefault = BuildTheme(isDark)
}
