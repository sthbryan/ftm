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
		"l logs",
		"w web",
		"o open config",
		"q quit",
	}

	content := strings.Join(shortcuts, "  •  ")

	return lipgloss.NewStyle().
		Foreground(ui.ThemeDefault.TextDim).
		Background(ui.ThemeDefault.Bg).
		Render(content)
}
