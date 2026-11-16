package components

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/ui/tui/ui"
)

type TunnelItem struct {
	Selected    bool
	Name        string
	Provider    string
	LocalPort   int
	StatusState int
	Width       int
	MetaWidth   int
}

func MetaText(provider string, localPort int) string {
	return fmt.Sprintf("%s:%d", provider, localPort)
}

func FitMetaColumn(rowWidth int, metas []string) int {
	widest := 0
	for _, meta := range metas {
		if w := lipgloss.Width(meta); w > widest {
			widest = w
		}
	}

	fixed := widestLabel(StatusBadge) + widestLabel(StatusLabel) + 3*columnGap
	if rowWidth-2*horizontalPadding-1-fixed-widest < minNameWidth {
		return 0
	}

	return widest
}

const (
	minNameWidth      = 8
	horizontalPadding = 1
	columnGap         = 2
)

const (
	TunnelStateOffline    = 0
	TunnelStateStarting   = 1
	TunnelStateConnecting = 2
	TunnelStateOnline     = 3
	TunnelStateError      = 4
	TunnelStateTimeout    = 5
	TunnelStateStopped    = 6
)

var allStates = []int{
	TunnelStateOffline,
	TunnelStateStarting,
	TunnelStateConnecting,
	TunnelStateOnline,
	TunnelStateError,
	TunnelStateTimeout,
	TunnelStateStopped,
}

func StatusBadge(state int) string {
	switch state {
	case TunnelStateStarting, TunnelStateConnecting:
		return i18n.T("badge_starting")
	case TunnelStateOnline:
		return i18n.T("badge_online")
	case TunnelStateError, TunnelStateTimeout:
		return i18n.T("badge_error")
	default:
		return i18n.T("badge_offline")
	}
}

func StatusLabel(state int) string {
	switch state {
	case TunnelStateStarting:
		return i18n.T("status_starting")
	case TunnelStateConnecting:
		return i18n.T("status_connecting")
	case TunnelStateOnline:
		return i18n.T("status_online")
	case TunnelStateError:
		return i18n.T("status_error")
	case TunnelStateStopped:
		return i18n.T("status_offline")
	default:
		return i18n.T("status_offline")
	}
}

func widestLabel(label func(int) string) int {
	widest := 0
	for _, state := range allStates {
		if w := lipgloss.Width(label(state)); w > widest {
			widest = w
		}
	}
	return widest
}

func StatusColor(state int) color.Color {
	switch state {
	case TunnelStateOnline:
		return ui.ThemeDefault.Success
	case TunnelStateError, TunnelStateTimeout:
		return ui.ThemeDefault.Danger
	case TunnelStateStarting, TunnelStateConnecting:
		return ui.ThemeDefault.Gold
	default:
		return ui.ThemeDefault.TextDim
	}
}

func (t *TunnelItem) background() color.Color {
	switch t.StatusState {
	case TunnelStateStarting, TunnelStateConnecting:
		return ui.ThemeDefault.Connecting
	case TunnelStateOnline:
		return ui.ThemeDefault.Online
	case TunnelStateError, TunnelStateTimeout:
		return ui.ThemeDefault.Error
	case TunnelStateStopped:
		return ui.ThemeDefault.Stopped
	default:
		return ui.ThemeDefault.Offline
	}
}

type column struct {
	text  string
	width int
	style lipgloss.Style
}

func (t *TunnelItem) Render() string {
	background := t.background()

	cell := lipgloss.NewStyle().Background(background)

	inner := t.Width - 2*horizontalPadding - 1
	if inner < minNameWidth {
		inner = minNameWidth
	}

	trailing := []column{
		{
			text:  StatusLabel(t.StatusState),
			width: widestLabel(StatusLabel),
			style: cell.Foreground(StatusColor(t.StatusState)),
		},
	}

	if t.MetaWidth > 0 {
		trailing = append(trailing, column{
			text:  MetaText(t.Provider, t.LocalPort),
			width: t.MetaWidth,
			style: cell.Foreground(ui.ThemeDefault.TextDim).Align(lipgloss.Right),
		})
	}

	leading := []column{
		{
			text:  StatusBadge(t.StatusState),
			width: widestLabel(StatusBadge),
			style: cell.Foreground(StatusColor(t.StatusState)).Bold(true),
		},
	}

	nameWidth := t.nameWidth(inner, leading, trailing)
	for nameWidth < minNameWidth && len(trailing) > 0 {
		trailing = trailing[:len(trailing)-1]
		nameWidth = t.nameWidth(inner, leading, trailing)
	}

	name := column{
		text:  ui.Truncate(t.Name, nameWidth),
		width: nameWidth,
		style: cell.Foreground(ui.ThemeDefault.Text).Bold(t.Selected),
	}

	columns := make([]column, 0, len(leading)+1+len(trailing))
	columns = append(columns, leading...)
	columns = append(columns, name)
	columns = append(columns, trailing...)

	rendered := make([]string, 0, len(columns))
	for _, c := range columns {
		rendered = append(rendered, c.style.Width(c.width).Render(c.text))
	}

	itemStyle := lipgloss.NewStyle().
		Background(background).
		Width(t.Width).
		MaxHeight(1).
		Padding(0, horizontalPadding)

	if t.Selected {
		itemStyle = itemStyle.
			BorderStyle(lipgloss.Border{Left: "▌"}).
			BorderLeft(true).
			BorderForeground(ui.ThemeDefault.Gold).
			BorderBackground(background)
	} else {
		itemStyle = itemStyle.PaddingLeft(horizontalPadding + 1)
	}

	return itemStyle.Render(strings.Join(rendered, cell.Render(strings.Repeat(" ", columnGap))))
}

func (t *TunnelItem) nameWidth(inner int, leading, trailing []column) int {
	used := 0
	for _, c := range leading {
		used += c.width
	}
	for _, c := range trailing {
		used += c.width
	}

	gaps := (len(leading) + len(trailing)) * columnGap

	return inner - used - gaps
}
