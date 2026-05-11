package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sthbryan/ftm/internal/app/ui"
	"github.com/sthbryan/ftm/internal/i18n"
)

type FormView struct {
	Width      int
	Focus      int
	IsEditMode bool
	Name       string
	Provider   string
	Port       string
}

func NewFormView() *FormView {
	return &FormView{}
}

func (f *FormView) Render() string {
	t := ui.ThemeDefault
	inputWidth := 25
	labelWidth := 17
	totalWidth := labelWidth + 2 + inputWidth

	newTunnelText := i18n.T("new_tunnel")
	editTunnelText := i18n.T("edit_tunnel")
	newTunnelDesc := i18n.T("new_tunnel_desc")
	editTunnelDesc := i18n.T("edit_tunnel_desc")
	nameLabel := i18n.T("name_label")
	nameHint := i18n.T("tunnel_name_hint")
	providerLabel := i18n.T("provider_label")
	providerHint := i18n.T("provider_hint")
	portLabel := i18n.T("local_port")
	portHint := i18n.T("port_hint")
	navHint := i18n.T("form_nav_hint")

	header := newTunnelText
	subheader := newTunnelDesc
	if f.IsEditMode {
		header = editTunnelText
		subheader = editTunnelDesc
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(t.Gold).
		Bold(true).
		Render(header)

	subheaderStyle := lipgloss.NewStyle().
		Foreground(t.TextDim).
		Render(subheader)

	if f.Focus == 0 {
		nameLabel = "▸ " + nameLabel
		nameHint = i18n.T("type_hint")
	}
	if f.Focus == 1 {
		providerLabel = "▸ " + providerLabel
		providerHint = i18n.T("arrow_hint")
	}
	if f.Focus == 2 {
		portLabel = "▸ " + portLabel
		portHint = i18n.T("numbers_hint")
	}

	lines := []string{
		headerStyle,
		"",
		subheaderStyle,
		"",
		f.fieldWithLabel(nameLabel, nameHint, f.Name, 0, t, inputWidth, labelWidth),
		"",
		f.fieldWithLabel(providerLabel, providerHint, f.Provider, 1, t, inputWidth, labelWidth),
		"",
		f.fieldWithLabel(portLabel, portHint, f.Port, 2, t, inputWidth, labelWidth),
		"",
		f.submitButton(t, inputWidth),
		"",
		lipgloss.NewStyle().Foreground(t.TextDim).Render(navHint),
	}

	content := strings.Join(lines, "\n")
	return centerBlock(content, f.Width, totalWidth)
}

func (f *FormView) fieldWithLabel(label, hint, value string, field int, t *ui.Theme, inputWidth, labelWidth int) string {
	if value == "" {
		value = "..."
	}
	if field == 1 && f.Focus == 1 {
		value = "‹ " + value + " ›"
	}

	labelStyle := f.labelStyle(t, field, labelWidth)
	inputStyle := f.inputStyle(t, field, inputWidth)
	hintStyle := lipgloss.NewStyle().Foreground(t.Bronze)

	return fmt.Sprintf("%s\n%s\n%s",
		labelStyle.Render(label+":"),
		inputStyle.Render(value),
		hintStyle.Render(hint),
	)
}

func (f *FormView) submitButton(t *ui.Theme, inputWidth int) string {
	btnText := i18n.T("submit_new")
	if f.IsEditMode {
		btnText = i18n.T("submit_edit")
	}

	btnStyle := lipgloss.NewStyle().
		Background(t.Bronze).
		Foreground(t.Text).
		Bold(true).
		Padding(0, 6).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(t.Bronze).Width(inputWidth)

	if f.Focus == 4 {
		btnStyle = lipgloss.NewStyle().
			Background(t.Gold).
			Foreground(t.Offline).
			Bold(true).
			Padding(0, 6).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(t.Gold).Width(inputWidth)
	}

	return btnStyle.Render(btnText)
}

func (f *FormView) labelStyle(t *ui.Theme, field, width int) lipgloss.Style {
	style := lipgloss.NewStyle().Width(width).Foreground(t.TextDim)

	if f.Focus == field {
		style = style.Bold(true).Foreground(t.Gold)
	}

	return style
}

func (f *FormView) inputStyle(t *ui.Theme, field, width int) lipgloss.Style {
	style := lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		BorderStyle(lipgloss.RoundedBorder())

	if f.Focus == field {
		style = style.
			BorderForeground(t.Gold).
			Foreground(t.Text)
	} else {
		style = style.
			BorderForeground(t.Bronze).
			Foreground(t.TextDim)
	}

	return style
}

func centerBlock(content string, screenWidth, blockWidth int) string {
	lines := strings.Split(content, "\n")
	var result []string

	indent := (screenWidth - blockWidth) / 2
	if indent < 0 {
		indent = 0
	}
	prefix := strings.Repeat(" ", indent)

	for _, line := range lines {
		result = append(result, prefix+line)
	}

	return strings.Join(result, "\n")
}
