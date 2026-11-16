package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

func Repeat(s string, count int) string {
	if count <= 0 {
		return ""
	}
	return strings.Repeat(s, count)
}

func Clamp(v, min int) int {
	if v < min {
		return min
	}
	return v
}

func Truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= width {
		return s
	}
	if width <= 1 {
		return runewidth.Truncate(s, width, "")
	}
	return runewidth.Truncate(s, width, "…")
}

func Center(s string, width int) string {
	pad := (width - lipgloss.Width(s)) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + s
}
