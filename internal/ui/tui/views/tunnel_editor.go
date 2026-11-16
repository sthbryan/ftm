package views

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/ui/tui/ui"
)

type TunnelEditor struct {
	Providers  []string
	Focus      int
	IsEditMode bool
	Name       string
	Provider   string
	Port       string

	HasCursor    bool
	CursorRow    int
	CursorColumn int
}

func NewTunnelEditor() *TunnelEditor {
	return &TunnelEditor{}
}

const (
	editorWidth = 54
	submitFocus = 3
)

func (f *TunnelEditor) Render() string {
	t := ui.ThemeDefault
	inner := ui.PanelInner(editorWidth)

	title, subtitle := i18n.T("new_tunnel"), i18n.T("new_tunnel_desc")
	if f.IsEditMode {
		title, subtitle = i18n.T("edit_tunnel"), i18n.T("edit_tunnel_desc")
	}

	dim := lipgloss.NewStyle().Foreground(t.TextDim).Width(inner)

	var lines []string
	add := func(section string) {
		lines = append(lines, strings.Split(section, "\n")...)
	}

	addField := func(index int, label, hint, focusedHint, value string) {
		if f.Focus == index && entersText(index) {
			f.markCursor(len(lines)+valueRowInField, value)
		}
		add(f.field(inner, index, label, hint, focusedHint, value))
	}

	add(dim.Render(subtitle))
	add("")
	addField(0, i18n.T("name_label"), i18n.T("tunnel_name_hint"), i18n.T("type_hint"), f.Name)
	add("")
	addField(1, i18n.T("provider_label"), f.providerHint(inner), i18n.T("arrow_hint"), f.Provider)
	add("")
	addField(2, i18n.T("local_port"), i18n.T("port_hint"), i18n.T("numbers_hint"), f.Port)
	add("")
	add(lipgloss.NewStyle().Width(inner).Align(lipgloss.Center).Render(f.submitButton()))
	add("")
	add(dim.Align(lipgloss.Center).Render(i18n.T("editor_nav_hint")))

	body := strings.Join(lines, "\n")

	return ui.Panel(title, body, editorWidth, lipgloss.Height(body)+ui.PanelChrome, t.Gold)
}

const (
	valueRowInField = 2
	valueColumn     = ui.PanelChrome + 2
)

func entersText(index int) bool {
	return index == 0 || index == 2
}

func (f *TunnelEditor) markCursor(row int, value string) {
	f.HasCursor = true
	f.CursorRow = row + 1
	f.CursorColumn = valueColumn + lipgloss.Width(value)
}

func (f *TunnelEditor) providerHint(width int) string {
	if len(f.Providers) == 0 {
		return i18n.T("provider_hint")
	}

	return ui.Truncate(strings.Join(f.Providers, " • "), width)
}

func (f *TunnelEditor) field(inner, index int, label, hint, focusedHint, value string) string {
	t := ui.ThemeDefault
	focused := f.Focus == index

	if value == "" {
		value = "…"
	}

	labelStyle := lipgloss.NewStyle().Foreground(t.TextDim)
	border, foreground := t.Bronze, t.TextDim

	if focused {
		label = "▸ " + label
		hint = focusedHint
		labelStyle = labelStyle.Foreground(t.Gold).Bold(true)
		border, foreground = t.Gold, t.Text

		if index == 1 {
			value = "‹ " + value + " ›"
		}
	}

	input := lipgloss.NewStyle().
		Width(inner).
		Padding(0, 1).
		Border(lipgloss.NormalBorder()).
		BorderForeground(border).
		Foreground(foreground).
		Render(value)

	return strings.Join([]string{
		labelStyle.Render(label),
		input,
		lipgloss.NewStyle().Foreground(t.Bronze).Render(hint),
	}, "\n")
}

func (f *TunnelEditor) submitButton() string {
	t := ui.ThemeDefault

	label := i18n.T("submit_new")
	if f.IsEditMode {
		label = i18n.T("submit_edit")
	}

	background, foreground := t.Button, t.ButtonText
	if f.Focus == submitFocus {
		background, foreground = t.ButtonActive, t.ButtonActiveText
	}

	return lipgloss.NewStyle().
		Background(background).
		Foreground(foreground).
		Bold(true).
		Padding(0, 4).
		Render(label)
}
