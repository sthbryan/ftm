package components

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/sthbryan/ftm/internal/ui/tui/ui"
)

type Shortcut struct {
	Keys  string
	Label string
}

type HelpBar struct {
	Shortcuts []Shortcut
	Width     int
}

func NewHelpBar() *HelpBar {
	return &HelpBar{}
}

const helpSeparator = "  •  "

func (h *HelpBar) Render() string {
	if len(h.Shortcuts) == 0 {
		return ""
	}

	keyStyle := lipgloss.NewStyle().Foreground(ui.ThemeDefault.Accent)
	labelStyle := lipgloss.NewStyle().Foreground(ui.ThemeDefault.TextDim)
	separator := labelStyle.Render(helpSeparator)

	entries := make([]string, 0, len(h.Shortcuts))
	for _, s := range h.Shortcuts {
		entries = append(entries, keyStyle.Render(s.Keys)+" "+labelStyle.Render(s.Label))
	}

	var b strings.Builder
	lineWidth := 0

	for i, entry := range entries {
		entryWidth := lipgloss.Width(entry)

		switch {
		case i == 0:
			lineWidth = entryWidth
		case h.Width > 0 && lineWidth+lipgloss.Width(separator)+entryWidth > h.Width:
			b.WriteString("\n")
			lineWidth = entryWidth
		default:
			b.WriteString(separator)
			lineWidth += lipgloss.Width(separator) + entryWidth
		}

		b.WriteString(entry)
	}

	return b.String()
}
