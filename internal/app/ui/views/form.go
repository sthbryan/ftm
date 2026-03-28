package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sthbryan/ftm/internal/app/ui"
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

	header := "✨ New Tunnel"
	subheader := "Create a secure tunnel to your local service"
	if f.IsEditMode {
		header = "✏️ Edit Tunnel"
		subheader = "Modify your tunnel settings"
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(t.Gold).
		Bold(true).
		Render(header)

	subheaderStyle := lipgloss.NewStyle().
		Foreground(t.TextDim).
		Render(subheader)

	lines := []string{
		headerStyle,
		"",
		subheaderStyle,
		"",
		f.nameField(t, inputWidth, labelWidth),
		"",
		f.providerField(t, inputWidth, labelWidth),
		"",
		f.portField(t, inputWidth, labelWidth),
		"",
		f.submitButton(t, inputWidth),
		"",
		lipgloss.NewStyle().Foreground(t.TextDim).Render("TAB: to navigate fields\nENTER: to submit\nESC: to cancel"),
	}

	content := strings.Join(lines, "\n")
	return centerBlock(content, f.Width, totalWidth)
}

func (f *FormView) nameField(t *ui.Theme, inputWidth, labelWidth int) string {
	label := "Name"
	hint := "your tunnel identifier"

	if f.Focus == 0 {
		label = "▸ Name"
		hint = "type to enter"
	}

	value := f.Name
	if value == "" {
		value = "..."
	}

	labelStyle := f.labelStyle(t, 0, labelWidth)
	inputStyle := f.inputStyle(t, 0, inputWidth)
	hintStyle := lipgloss.NewStyle().Foreground(t.Bronze)

	return fmt.Sprintf("%s\n%s\n%s",
		labelStyle.Render(label+":"),
		inputStyle.Render(value),
		hintStyle.Render(hint),
	)
}

func (f *FormView) providerField(t *ui.Theme, inputWidth, labelWidth int) string {
	label := "Provider"
	value := f.Provider
	hint := "cloudflare | ngrok | local"

	if f.Focus == 1 {
		label = "▸ Provider"
		value = "‹ " + value + " ›"
		hint = "← → to change"
	}

	labelStyle := f.labelStyle(t, 1, labelWidth)
	inputStyle := f.inputStyle(t, 1, inputWidth)
	hintStyle := lipgloss.NewStyle().Foreground(t.Bronze)

	return fmt.Sprintf("%s\n%s\n%s",
		labelStyle.Render(label+":"),
		inputStyle.Render(value),
		hintStyle.Render(hint),
	)
}

func (f *FormView) portField(t *ui.Theme, inputWidth, labelWidth int) string {
	label := "Local Port"
	value := f.Port
	hint := "e.g. 3000, 8080"

	if f.Focus == 2 {
		label = "▸ Local Port"
		hint = "numbers only"
	}

	if value == "" {
		value = "..."
	}

	labelStyle := f.labelStyle(t, 2, labelWidth)
	inputStyle := f.inputStyle(t, 2, inputWidth)
	hintStyle := lipgloss.NewStyle().Foreground(t.Bronze)

	return fmt.Sprintf("%s\n%s\n%s",
		labelStyle.Render(label+":"),
		inputStyle.Render(value),
		hintStyle.Render(hint),
	)
}

func (f *FormView) submitButton(t *ui.Theme, inputWidth int) string {
	btnText := "Create Tunnel"
	if f.IsEditMode {
		btnText = "Save Changes"
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
