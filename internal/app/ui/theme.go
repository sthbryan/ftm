package ui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Gold       lipgloss.Color
	Bronze     lipgloss.Color
	Text       lipgloss.Color
	TextDim    lipgloss.Color
	Online     lipgloss.Color
	Offline    lipgloss.Color
	Connecting lipgloss.Color
	Error      lipgloss.Color
	Stopped    lipgloss.Color
	Success    lipgloss.Color
}

func DefaultTheme() *Theme {
	return &Theme{
		Gold:       lipgloss.Color("#c9a227"),
		Bronze:     lipgloss.Color("#8b7355"),
		Text:       lipgloss.Color("#ffffff"),
		TextDim:    lipgloss.Color("#9a9590"),
		Online:     lipgloss.Color("#1e3a2f"),
		Offline:    lipgloss.Color("#2a2824"),
		Connecting: lipgloss.Color("#3a3020"),
		Error:      lipgloss.Color("#3a2020"),
		Stopped:    lipgloss.Color("#3a3a3a"),
		Success:    lipgloss.Color("#7cb69d"),
	}
}

var ThemeDefault = DefaultTheme()
