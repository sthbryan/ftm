package views

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/ui/tui/ui"
)

type EmptyState struct {
	Height    int
	Width     int
	Dashboard string
	Sessions  int
}

func NewEmptyState() *EmptyState {
	return &EmptyState{}
}

const emptyWidth = 62

func (e *EmptyState) Render() string {
	t := ui.ThemeDefault

	width := emptyWidth
	if e.Width > 0 && e.Width < width {
		width = e.Width
	}
	inner := ui.PanelInner(width)

	body := e.body(inner, e.steps(inner))
	if e.Height > 0 && lipgloss.Height(body)+ui.PanelChrome > e.Height {
		body = e.body(inner, "")
	}

	panel := ui.Panel(i18n.T("welcome_title"), body, width, lipgloss.Height(body)+ui.PanelChrome, t.Gold)

	return ui.Overlay("", panel, e.Width, e.Height)
}

func (e *EmptyState) body(inner int, steps string) string {
	t := ui.ThemeDefault
	centered := lipgloss.NewStyle().Width(inner).Align(lipgloss.Center)

	lines := []string{centered.Foreground(t.Text).Render(i18n.T("welcome_lead"))}

	if steps != "" {
		lines = append(lines, "", steps)
	}

	return strings.Join(append(lines,
		"",
		centered.Render(lipgloss.NewStyle().
			Background(t.ButtonActive).
			Foreground(t.ButtonActiveText).
			Bold(true).
			Padding(0, 3).
			Render(i18n.T("create_first"))),
		"",
		centered.Foreground(t.TextDim).Render(i18n.T("press_a_hint")),
		"",
		centered.Foreground(t.Bronze).Render(
			i18n.T("tip_dashboard")+" "+e.Dashboard+"  •  ws "+fmt.Sprintf("%d", e.Sessions)),
	), "\n")
}

func (e *EmptyState) steps(inner int) string {
	t := ui.ThemeDefault

	const numberWidth = 3
	textWidth := ui.Clamp(inner-numberWidth, 1)

	number := lipgloss.NewStyle().Width(numberWidth).Foreground(t.Gold).Bold(true)
	title := lipgloss.NewStyle().Width(textWidth).Foreground(t.Text).Bold(true)
	detail := lipgloss.NewStyle().Width(textWidth).Foreground(t.TextDim)

	keys := []struct{ title, body string }{
		{"welcome_step_provider_title", "welcome_step_provider_body"},
		{"welcome_step_port_title", "welcome_step_port_body"},
		{"welcome_step_share_title", "welcome_step_share_body"},
	}

	rows := make([]string, 0, len(keys))
	for i, key := range keys {
		text := title.Render(i18n.T(key.title)) + "\n" + detail.Render(i18n.T(key.body))
		rows = append(rows, lipgloss.JoinHorizontal(
			lipgloss.Top,
			number.Render(fmt.Sprintf("%d", i+1)),
			text,
		))
	}

	return strings.Join(rows, "\n\n")
}
