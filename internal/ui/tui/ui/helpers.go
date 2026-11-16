package ui

import (
	"strings"

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
