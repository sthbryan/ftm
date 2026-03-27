package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sthbryan/ftm/internal/app/ui"
)

type TunnelItem struct {
	Selected    bool
	Name        string
	Provider    string
	LocalPort   int
	StatusState int
	StatusMsg   string
	Width       int
}

func NewTunnelItem() *TunnelItem {
	return &TunnelItem{}
}

const (
	TunnelStateOffline    = 0
	TunnelStateStarting   = 1
	TunnelStateConnecting = 2
	TunnelStateOnline     = 3
	TunnelStateError      = 4
	TunnelStateTimeout    = 5
	TunnelStateStopped    = 6
)

func StatusBadge(state int) string {
	switch state {
	case TunnelStateStarting, TunnelStateConnecting:
		return "[...]"
	case TunnelStateOnline:
		return "[ON]"
	case TunnelStateError, TunnelStateTimeout:
		return "[ERR]"
	default:
		return "[OFF]"
	}
}

func StatusLabel(state int) string {
	switch state {
	case TunnelStateStarting:
		return "STARTING"
	case TunnelStateConnecting:
		return "CONNECTING"
	case TunnelStateOnline:
		return "ONLINE"
	case TunnelStateError:
		return "ERROR"
	case TunnelStateStopped:
		return "OFFLINE"
	default:
		return "OFFLINE"
	}
}

func (t *TunnelItem) Render() string {
	var bgColor lipgloss.Color

	switch t.StatusState {
	case TunnelStateStarting, TunnelStateConnecting:
		bgColor = ui.ThemeDefault.Connecting
	case TunnelStateOnline:
		bgColor = ui.ThemeDefault.Online
	case TunnelStateError, TunnelStateTimeout:
		bgColor = ui.ThemeDefault.Error
	case TunnelStateStopped:
		bgColor = ui.ThemeDefault.Stopped
	default:
		bgColor = ui.ThemeDefault.Offline
	}

	var parts []string

	if t.Selected {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(ui.ThemeDefault.Gold).
			Background(bgColor).
			Render(">"))
	} else {
		parts = append(parts, lipgloss.NewStyle().
			Background(bgColor).
			Render(" "))
	}

	parts = append(parts, t.statusBadge(bgColor))
	parts = append(parts, t.name(bgColor))
	parts = append(parts, t.statusText(bgColor))
	parts = append(parts, t.meta(bgColor))

	content := strings.Join(parts, lipgloss.NewStyle().Background(bgColor).Render(" "))

	itemStyle := lipgloss.NewStyle().
		Background(bgColor).
		Width(t.Width).
		Padding(0, 1)

	if t.Selected {
		itemStyle = itemStyle.BorderStyle(lipgloss.Border{
			Left: "█",
		}).BorderLeft(true).BorderForeground(ui.ThemeDefault.Gold)
	}

	return itemStyle.Render(content)
}

func (t *TunnelItem) statusBadge(bgColor lipgloss.Color) string {
	return lipgloss.NewStyle().
		Background(bgColor).
		Render(StatusBadge(t.StatusState))
}

func (t *TunnelItem) name(bgColor lipgloss.Color) string {
	name := t.Name
	if len(name) > 18 {
		name = name[:15] + "..."
	}

	return lipgloss.NewStyle().
		Bold(t.Selected).
		Foreground(ui.ThemeDefault.Text).
		Background(bgColor).
		Render(name)
}

func (t *TunnelItem) statusText(bgColor lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(ui.ThemeDefault.Text).
		Background(bgColor).
		Padding(0, 1).
		Render(t.StatusMsg)
}

func (t *TunnelItem) meta(bgColor lipgloss.Color) string {
	meta := fmt.Sprintf("%s :%d", t.Provider, t.LocalPort)

	return lipgloss.NewStyle().
		Foreground(ui.ThemeDefault.TextDim).
		Background(bgColor).
		Render(meta)
}
