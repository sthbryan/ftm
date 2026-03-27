package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sthbryan/ftm/internal/app/ui"
	"github.com/sthbryan/ftm/internal/app/ui/components"
	"github.com/sthbryan/ftm/internal/version"
)

type TunnelViewData struct {
	Name        string
	Provider    string
	LocalPort   int
	StatusState int
	StatusMsg   string
	PublicURL   string
	ErrorMsg    string
}

type ListView struct {
	Width          int
	Height         int
	Items          []TunnelViewData
	Cursor         int
	Message        string
	Dashboard      string
	TwoColumnLimit int
}

func NewListView() *ListView {
	return &ListView{
		TwoColumnLimit: 100,
	}
}

func (l *ListView) Render() string {
	if len(l.Items) == 0 {
		return ""
	}

	if l.Width >= l.TwoColumnLimit {
		return l.twoColumn()
	}

	return l.singleColumn()
}

func (l *ListView) twoColumn() string {
	var b strings.Builder

	gold := ui.ThemeDefault.Gold
	bronze := ui.ThemeDefault.Bronze
	text := ui.ThemeDefault.Text
	textDim := ui.ThemeDefault.TextDim

	title := lipgloss.NewStyle().
		Foreground(gold).
		Bold(true).
		Render("🎲  Foundry Tunnel Manager")
	versionStr := lipgloss.NewStyle().
		Foreground(textDim).
		Render("v" + version.Version)

	b.WriteString(title)
	b.WriteString(strings.Repeat(" ", l.Width-lipgloss.Width(title)-lipgloss.Width(versionStr)-ui.HeaderMargin))
	b.WriteString(versionStr)
	b.WriteString("\n\n")

	leftWidth := int(float64(l.Width) * 0.4)
	rightWidth := l.Width - leftWidth - 3

	leftHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(text).
		Render("Your Connections")

	rightHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(text).
		Render("Selected Tunnel")

	b.WriteString(leftHeader)
	b.WriteString(strings.Repeat(" ", leftWidth-lipgloss.Width(leftHeader)+3))
	b.WriteString(rightHeader)
	b.WriteString("\n")

	dividerStyle := lipgloss.NewStyle().Foreground(bronze)
	b.WriteString(dividerStyle.Render(strings.Repeat("─", l.Width-2)))
	b.WriteString("\n")

	listContent := l.renderTunnelList(leftWidth - 2)
	detailContent := l.renderDetailPanel(rightWidth - 2)

	listLines := strings.Split(listContent, "\n")
	detailLines := strings.Split(detailContent, "\n")

	maxLines := len(listLines)
	if len(detailLines) > maxLines {
		maxLines = len(detailLines)
	}

	for i := 0; i < maxLines; i++ {
		leftLine := ""
		if i < len(listLines) {
			leftLine = listLines[i]
		}
		rightLine := ""
		if i < len(detailLines) {
			rightLine = detailLines[i]
		}

		leftLine = lipgloss.NewStyle().Width(leftWidth).Render(leftLine)

		b.WriteString(leftLine)
		b.WriteString(" │ ")
		b.WriteString(rightLine)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	if l.Message != "" {
		msgStyle := lipgloss.NewStyle().Foreground(gold).Bold(true)
		b.WriteString(msgStyle.Render("✓ " + l.Message))
		b.WriteString("\n")
	}

	b.WriteString(components.NewHelpBar().Render())

	return b.String()
}

func (l *ListView) singleColumn() string {
	var b strings.Builder

	gold := ui.ThemeDefault.Gold
	textDim := ui.ThemeDefault.TextDim

	title := lipgloss.NewStyle().
		Foreground(gold).
		Bold(true).
		Render("🎲  Foundry Tunnel Manager")
	versionStr := lipgloss.NewStyle().
		Foreground(textDim).
		Render("v" + version.Version)

	b.WriteString(title)
	b.WriteString(strings.Repeat(" ", l.Width-lipgloss.Width(title)-lipgloss.Width(versionStr)-ui.HeaderMargin))
	b.WriteString(versionStr)
	b.WriteString("\n\n")

	if l.Dashboard != "" {
		urlStyle := lipgloss.NewStyle().Foreground(gold)
		b.WriteString(urlStyle.Render("🌐  Dashboard: " + l.Dashboard + " (press 'w')"))
		b.WriteString("\n\n")
	}

	b.WriteString(l.renderTunnelList(l.Width - 2))
	b.WriteString("\n")

	if l.Message != "" {
		msgStyle := lipgloss.NewStyle().Foreground(gold).Bold(true)
		b.WriteString(msgStyle.Render("✓ " + l.Message))
		b.WriteString("\n")
	}

	b.WriteString(components.NewHelpBar().Render())

	return b.String()
}

func (l *ListView) renderTunnelList(width int) string {
	var b strings.Builder

	for i, item := range l.Items {
		tunnelItem := components.TunnelItem{
			Selected:    i == l.Cursor,
			Name:        item.Name,
			Provider:    item.Provider,
			LocalPort:   item.LocalPort,
			StatusState: item.StatusState,
			StatusMsg:   item.StatusMsg,
			Width:       width,
		}
		line := tunnelItem.Render()
		b.WriteString(line)
		if i < len(l.Items)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (l *ListView) renderDetailPanel(width int) string {
	if l.Cursor < 0 || l.Cursor >= len(l.Items) {
		return lipgloss.NewStyle().
			Foreground(ui.ThemeDefault.TextDim).
			Render("Select a tunnel to view details")
	}

	item := l.Items[l.Cursor]

	panel := components.DetailPanel{
		Name:        item.Name,
		Provider:    item.Provider,
		LocalPort:   item.LocalPort,
		StatusState: item.StatusState,
		StatusMsg:   item.StatusMsg,
		PublicURL:   item.PublicURL,
		ErrorMsg:    item.ErrorMsg,
		Width:       width - 2,
	}

	return panel.Render()
}
