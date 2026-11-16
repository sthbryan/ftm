package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	PanelChrome  = 2
	PanelPadding = 2
)

func PanelInner(width int) int {
	return Clamp(width-PanelChrome-PanelPadding, 0)
}

func Panel(title, content string, width, height int, accent color.Color) string {
	if width <= PanelChrome+2 || height <= PanelChrome {
		return content
	}

	edge := lipgloss.NewStyle().Foreground(accent)

	heading := ""
	if title != "" {
		heading = edge.Render("─ ") +
			edge.Bold(true).Render(Truncate(title, width-6)) +
			edge.Render(" ")
	}

	fill := Clamp(width-PanelChrome-lipgloss.Width(heading), 0)

	top := edge.Render("┌") + heading + edge.Render(strings.Repeat("─", fill)) + edge.Render("┐")

	body := lipgloss.NewStyle().
		Width(width).
		Height(height-1).
		MaxHeight(height-1).
		Border(lipgloss.NormalBorder()).
		BorderTop(false).
		BorderForeground(accent).
		Padding(0, 1).
		Render(content)

	return top + "\n" + body
}
