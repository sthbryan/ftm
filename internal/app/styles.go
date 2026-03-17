package app

import "github.com/charmbracelet/lipgloss"

const (
	ColorBg       = "#1a1814"
	ColorGold     = "#c9a227"
	ColorBronze   = "#8b7355"
	ColorText     = "#e8e6e1"
	ColorTextDim  = "#9a9590"

	ColorOnline   = "#1e3a2f"
	ColorOffline  = "#2a2824"
	ColorStarting = "#3a3020"
	ColorError    = "#3a2020"
)

var TitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color(ColorGold))

var StatusOnlineStyle = lipgloss.NewStyle().
	Background(lipgloss.Color(ColorOnline)).
	Foreground(lipgloss.Color(ColorText))

var StatusOfflineStyle = lipgloss.NewStyle().
	Background(lipgloss.Color(ColorOffline)).
	Foreground(lipgloss.Color(ColorText))

var StatusStartingStyle = lipgloss.NewStyle().
	Background(lipgloss.Color(ColorStarting)).
	Foreground(lipgloss.Color(ColorText))

var StatusErrorStyle = lipgloss.NewStyle().
	Background(lipgloss.Color(ColorError)).
	Foreground(lipgloss.Color(ColorText))

var PanelStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color(ColorBronze)).
	Padding(1)

var SelectedStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorGold)).
	Bold(true)

var URLBoxStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(ColorGold)).
	Background(lipgloss.Color(ColorBg)).
	Foreground(lipgloss.Color(ColorText)).
	Padding(0, 1)

var ButtonStyle = lipgloss.NewStyle().
	Background(lipgloss.Color(ColorBronze)).
	Foreground(lipgloss.Color(ColorBg)).
	Padding(0, 2)

var ButtonActiveStyle = lipgloss.NewStyle().
	Background(lipgloss.Color(ColorGold)).
	Foreground(lipgloss.Color(ColorBg)).
	Bold(true).
	Padding(0, 2)

// Additional styles from model.go (maintained for compatibility)

var TitleAccentStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color(ColorGold))

var StatusOnline = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#7cb69d")).
	Bold(true)

var StatusOffline = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorTextDim))

var StatusStarting = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorGold))

var URLStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#7cb69d")).
	Underline(true)

var HelpStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(ColorTextDim))
