package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sthbryan/ftm/internal/app/ui"
)

type FormView struct {
	Width    int
	Focus    int
	Name     string
	Provider string
	Port     string
}

func NewFormView() *FormView {
	return &FormView{}
}

func (f *FormView) Render() string {
	var b strings.Builder

	bg := ui.ThemeDefault.Bg
	gold := ui.ThemeDefault.Gold
	bronze := ui.ThemeDefault.Bronze
	text := ui.ThemeDefault.Text
	textDim := ui.ThemeDefault.TextDim

	header := lipgloss.NewStyle().
		Foreground(gold).
		Bold(true).
		Render("➕  Add New Tunnel")

	b.WriteString(header)
	b.WriteString("\n\n")

	inputWidth := 25

	b.WriteString(f.nameField(inputWidth, gold, bronze, text, textDim, bg))
	b.WriteString("\n\n")

	b.WriteString(f.providerField(inputWidth, gold, bronze, text, textDim, bg))
	b.WriteString("\n\n")

	b.WriteString(f.portField(inputWidth, gold, bronze, text, textDim, bg))
	b.WriteString("\n\n")

	b.WriteString(f.submitButton(inputWidth, gold, bronze, text, bg))

	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().
		Foreground(textDim).
		Render("tab: next • enter: submit • esc: cancel"))

	return b.String()
}

func (f *FormView) nameField(inputWidth int, gold, bronze, text, textDim, bg lipgloss.Color) string {
	nameLabel := "Name"
	nameHint := ""

	if f.Focus == 0 {
		nameLabel = "▶ Name"
		nameHint = "type to enter"
	}

	labelStyle := f.labelStyle(0, gold, textDim, bg)
	b := labelStyle.Render(nameLabel + ":")
	b += "\n"

	nameStyle := f.inputStyle(0, inputWidth, gold, bronze, text, textDim, bg)
	b += nameStyle.Render(f.Name)

	if nameHint != "" {
		hintStyle := lipgloss.NewStyle().Foreground(bronze)
		b += " " + hintStyle.Render(nameHint)
	}

	return b
}

func (f *FormView) providerField(inputWidth int, gold, bronze, text, textDim, bg lipgloss.Color) string {
	providerLabel := "Provider"
	providerValue := f.Provider
	providerHint := ""

	if f.Focus == 1 {
		providerLabel = "▶ Provider"
		providerValue = "← " + providerValue + " →"
		providerHint = "arrows to change"
	}

	labelStyle := f.labelStyle(1, gold, textDim, bg)
	b := labelStyle.Render(providerLabel + ":")
	b += "\n"

	providerStyle := f.inputStyle(1, inputWidth, gold, bronze, text, textDim, bg)
	b += providerStyle.Render(providerValue)

	if providerHint != "" {
		hintStyle := lipgloss.NewStyle().Foreground(bronze)

		b += " " + hintStyle.Render(providerHint)
	}

	return b
}

func (f *FormView) portField(inputWidth int, gold, bronze, text, textDim, bg lipgloss.Color) string {
	portLabel := "Local Port"
	portHint := ""

	if f.Focus == 2 {
		portLabel = "▶ Local Port"
		portHint = "numbers only"
	}

	labelStyle := f.labelStyle(2, gold, textDim, bg)
	b := labelStyle.Render(portLabel + ":")
	b += "\n"

	portStyle := f.inputStyle(2, inputWidth, gold, bronze, text, textDim, bg)
	b += portStyle.Render(f.Port)

	if portHint != "" {
		hintStyle := lipgloss.NewStyle().Foreground(bronze)

		b += " " + hintStyle.Render(portHint)
	}

	return b
}

func (f *FormView) submitButton(inputWidth int, gold, bronze, text, bg lipgloss.Color) string {
	buttonStyle := lipgloss.NewStyle().
		Background(bronze).
		Foreground(bg).
		Padding(0, 2)

	if f.Focus == 3 {
		buttonStyle = lipgloss.NewStyle().
			Background(gold).
			Foreground(bg).
			Bold(true).
			Padding(0, 2)
	}

	placeholder := lipgloss.NewStyle().
		Width(inputWidth).
		Render("")

	return placeholder + buttonStyle.Render(" Submit ")
}

func (f *FormView) labelStyle(field int, gold, textDim, bg lipgloss.Color) lipgloss.Style {
	style := lipgloss.NewStyle().Width(15).Foreground(textDim)

	if f.Focus == field {
		style = style.Bold(true).Foreground(gold)
	}

	return style
}

func (f *FormView) inputStyle(field, width int, gold, bronze, text, textDim, bg lipgloss.Color) lipgloss.Style {
	style := lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		BorderStyle(lipgloss.RoundedBorder())

	if f.Focus == field {
		style = style.
			BorderForeground(gold).
			Foreground(text)
	} else {
		style = style.
			BorderForeground(bronze).
			Foreground(textDim)
	}

	return style
}
