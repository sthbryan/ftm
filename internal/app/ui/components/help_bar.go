package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sthbryan/ftm/internal/app/ui"
)

type HelpBar struct{}

func NewHelpBar() *HelpBar {
	return &HelpBar{}
}

func (h *HelpBar) Render() string {
	shortcuts := []string{
		"↑/↓ navigate",
		"Enter toggle",
		"a add",
		"e edit",
		"d delete",
		"l logs",
		"w web",
		"o open config",
		"? help",
		"q quit",
	}

	firstLine := strings.Join(shortcuts[:5], "  •  ")
	secondLine := strings.Join(shortcuts[5:], "  •  ")
	content := firstLine + "\n" + secondLine

	return lipgloss.NewStyle().
		Foreground(ui.ThemeDefault.TextDim).
		Render(content)
}
