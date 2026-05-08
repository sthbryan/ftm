package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sthbryan/ftm/internal/app/ui"
	"github.com/sthbryan/ftm/internal/i18n"
)

type HelpBar struct{}

func NewHelpBar() *HelpBar {
	return &HelpBar{}
}

func (h *HelpBar) Render() string {
	shortcuts := []string{
		i18n.T("navigation_hint"),
		"a " + i18n.T("create"),
		"e " + i18n.T("edit"),
		"d " + i18n.T("delete"),
		"s " + i18n.T("settings"),
		"l " + i18n.T("logs"),
		"w web",
		"o config",
		"q " + i18n.T("close"),
	}

	firstLine := strings.Join(shortcuts[:5], "  •  ")
	secondLine := strings.Join(shortcuts[5:], "  •  ")
	content := firstLine + "\n" + secondLine

	return lipgloss.NewStyle().
		Foreground(ui.ThemeDefault.TextDim).
		Render(content)
}
