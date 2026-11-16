package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func OverlayOrigin(foreground string, width, height int) (x, y int) {
	return Clamp((width-lipgloss.Width(foreground))/2, 0),
		Clamp((height-lipgloss.Height(foreground))/2, 0)
}

func Overlay(background, foreground string, width, height int) string {
	if width <= 0 || height <= 0 {
		return foreground
	}

	x, y := OverlayOrigin(foreground, width, height)

	canvas := lipgloss.NewCanvas(width, height).
		Compose(lipgloss.NewCompositor(
			lipgloss.NewLayer(Dim(background)),
			lipgloss.NewLayer(foreground).X(x).Y(y).Z(1),
		)).
		Render()

	return Fill(canvas, width, height)
}

func Dim(content string) string {
	if content == "" {
		return content
	}

	return lipgloss.NewStyle().
		Foreground(ThemeDefault.Dimmed).
		Render(ansi.Strip(content))
}

func Fill(content string, width, height int) string {
	lines := strings.Split(content, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}

	for i, line := range lines[:height] {
		lines[i] = line + Repeat(" ", width-lipgloss.Width(line))
	}

	return strings.Join(lines[:height], "\n")
}
