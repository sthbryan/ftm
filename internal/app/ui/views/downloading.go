package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sthbryan/ftm/internal/app/ui"
)

type DownloadingView struct {
	Width   int
	Percent float64
	Name    string
	Current int64
	Total   int64
}

func NewDownloadingView() *DownloadingView {
	return &DownloadingView{}
}

func (d *DownloadingView) Render(progressBarView string) string {
	var b strings.Builder

	gold := ui.ThemeDefault.Gold
	text := ui.ThemeDefault.Text
	textDim := ui.ThemeDefault.TextDim

	header := lipgloss.NewStyle().
		Foreground(gold).
		Bold(true).
		Render("⬇️  Installing")

	b.WriteString(header)
	b.WriteString("\n\n")

	name := d.Name
	if name == "" {
		name = "binary"
	}

	var step string
	switch {
	case d.Percent < 90:
		step = fmt.Sprintf("Downloading %s...", name)
	case d.Percent < 100:
		step = fmt.Sprintf("Installing %s...", name)
	default:
		step = "Complete!"
	}

	stepStyle := lipgloss.NewStyle().Foreground(text)
	b.WriteString(stepStyle.Render(step))
	b.WriteString("\n\n")

	barWidth := d.Width - 10
	padding := (d.Width - barWidth) / 2

	progressContainer := lipgloss.NewStyle().
		Width(barWidth).
		Render("[" + progressBarView + "]")

	b.WriteString(strings.Repeat(" ", padding))
	b.WriteString(progressContainer)
	b.WriteString("\n")

	if d.Total > 0 && d.Percent < 50 {
		sizeStyle := lipgloss.NewStyle().Foreground(textDim)
		b.WriteString(sizeStyle.Render(fmt.Sprintf("%.1f MB / %.1f MB",
			float64(d.Current)/(1024*1024),
			float64(d.Total)/(1024*1024))))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().
		Foreground(textDim).
		Render("esc: cancel"))

	return b.String()
}
