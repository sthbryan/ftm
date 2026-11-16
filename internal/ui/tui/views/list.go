package views

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/ui/tui/components"
	"github.com/sthbryan/ftm/internal/ui/tui/ui"
	"github.com/sthbryan/ftm/internal/version"
)

type TunnelViewData struct {
	Name           string
	Provider       string
	LocalPort      int
	StatusState    int
	PublicURL      string
	Visitors       int
	ActiveSessions int
	ErrorMsg       string
	Logs           []string
}

type ListView struct {
	Width          int
	Height         int
	Items          []TunnelViewData
	Cursor         int
	Message        string
	Dashboard      string
	Sessions       int
	TwoColumnLimit int
	UpdateBadge    string
	Shortcuts      []components.Shortcut

	ListTop int

	VisibleRows int
	FirstItem   int
}

const ItemHeight = 1

const (
	minBodyHeight  = 4
	listColumnSize = 0.42
)

func NewListView() *ListView {
	return &ListView{
		TwoColumnLimit: 100,
	}
}

func (l *ListView) Render() string {
	header := l.header()
	footer := l.footer()

	bodyHeight := l.Height - lipgloss.Height(header) - lipgloss.Height(footer)
	if bodyHeight < minBodyHeight {
		bodyHeight = minBodyHeight
	}

	l.ListTop = lipgloss.Height(header) + 1
	l.VisibleRows = l.rowsAvailable(bodyHeight)

	body := l.singleColumn(bodyHeight)
	if len(l.Items) > 0 && l.Width >= l.TwoColumnLimit {
		body = l.twoColumn(bodyHeight)
	}

	return header + "\n" + body + "\n" + footer
}

func (l *ListView) rowsAvailable(bodyHeight int) int {
	rows := bodyHeight - ui.PanelChrome
	if len(l.Items) > rows {
		rows -= 2
	}

	return ui.Clamp(rows, 1)
}

func (l *ListView) header() string {
	t := ui.ThemeDefault

	title := lipgloss.NewStyle().
		Foreground(t.Gold).
		Bold(true).
		Render(i18n.T("app_name_tui"))

	meta := lipgloss.NewStyle().
		Foreground(t.TextDim).
		Render(fmt.Sprintf("v%s  •  ws %d", version.Version, l.Sessions))

	gap := ui.Clamp(l.Width-lipgloss.Width(title)-lipgloss.Width(meta), 1)

	lines := []string{title + ui.Repeat(" ", gap) + meta}

	if l.UpdateBadge != "" {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(t.Gold).
			Bold(true).
			Render(l.UpdateBadge))
	}

	if l.Dashboard != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(t.TextDim).Render(i18n.T("dashboard_label")+" ")+
			lipgloss.NewStyle().Foreground(t.Bronze).Render(l.Dashboard))
	}

	return strings.Join(append(lines, ""), "\n")
}

func (l *ListView) footer() string {
	lines := []string{""}

	if l.Message != "" {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(ui.ThemeDefault.Success).
			Bold(true).
			Render(i18n.T("success_prefix")+" "+l.Message))
	}

	help := components.NewHelpBar()
	help.Shortcuts = l.Shortcuts
	help.Width = l.Width

	return strings.Join(append(lines, help.Render()), "\n")
}

func (l *ListView) twoColumn(bodyHeight int) string {
	leftWidth := int(float64(l.Width) * listColumnSize)
	rightWidth := l.Width - leftWidth - 1

	left := ui.Panel(
		i18n.T("connections"),
		l.renderTunnelList(ui.PanelInner(leftWidth)),
		leftWidth, bodyHeight, ui.ThemeDefault.Gold,
	)

	right := ui.Panel(
		i18n.T("selected_tunnel"),
		l.renderDetailPanel(ui.PanelInner(rightWidth), bodyHeight-ui.PanelChrome),
		rightWidth, bodyHeight, ui.ThemeDefault.Bronze,
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

func (l *ListView) singleColumn(bodyHeight int) string {
	inner := ui.PanelInner(l.Width)

	content := l.renderTunnelList(inner)
	if len(l.Items) == 0 {
		content = l.emptyBody(inner, bodyHeight-ui.PanelChrome)
	}

	return ui.Panel(i18n.T("connections"), content, l.Width, bodyHeight, ui.ThemeDefault.Gold)
}

func (l *ListView) emptyBody(width, height int) string {
	t := ui.ThemeDefault
	centered := lipgloss.NewStyle().Width(width).Align(lipgloss.Center)

	body := centered.Foreground(t.Text).Render(i18n.T("no_tunnels")) + "\n" +
		centered.Foreground(t.TextDim).Render(i18n.T("add_tunnel_prompt"))

	return lipgloss.NewStyle().
		Width(width).
		Height(ui.Clamp(height, lipgloss.Height(body))).
		AlignVertical(lipgloss.Center).
		Render(body)
}

func (l *ListView) visibleWindow() (first, count int) {
	total := len(l.Items)
	if l.VisibleRows <= 0 || l.VisibleRows >= total {
		return 0, total
	}

	first = l.Cursor - l.VisibleRows/2
	if first < 0 {
		first = 0
	}
	if first > total-l.VisibleRows {
		first = total - l.VisibleRows
	}

	return first, l.VisibleRows
}

func (l *ListView) renderTunnelList(width int) string {
	var b strings.Builder

	first, count := l.visibleWindow()
	l.FirstItem = first
	last := first + count

	metaWidth := components.FitMetaColumn(width, l.metaColumn())

	if first > 0 {
		b.WriteString(l.scrollHint(width, i18n.TF("more_above", first)))
		b.WriteString("\n")
		l.ListTop++
	}

	for i := first; i < last; i++ {
		item := l.Items[i]
		tunnelItem := components.TunnelItem{
			Selected:    i == l.Cursor,
			Name:        item.Name,
			Provider:    item.Provider,
			LocalPort:   item.LocalPort,
			StatusState: item.StatusState,
			Width:       width,
			MetaWidth:   metaWidth,
		}
		b.WriteString(tunnelItem.Render())
		if i < last-1 {
			b.WriteString("\n")
		}
	}

	if remaining := len(l.Items) - last; remaining > 0 {
		b.WriteString("\n")
		b.WriteString(l.scrollHint(width, i18n.TF("more_below", remaining)))
	}

	return b.String()
}

func (l *ListView) metaColumn() []string {
	metas := make([]string, 0, len(l.Items))
	for _, item := range l.Items {
		metas = append(metas, components.MetaText(item.Provider, item.LocalPort))
	}
	return metas
}

func (l *ListView) scrollHint(width int, text string) string {
	return lipgloss.NewStyle().
		Foreground(ui.ThemeDefault.TextDim).
		Width(ui.Clamp(width, 0)).
		Align(lipgloss.Center).
		Render(text)
}

func (l *ListView) renderDetailPanel(width, height int) string {
	if l.Cursor < 0 || l.Cursor >= len(l.Items) {
		return lipgloss.NewStyle().
			Foreground(ui.ThemeDefault.TextDim).
			Render(i18n.T("select_tunnel_details"))
	}

	item := l.Items[l.Cursor]

	panel := components.DetailPanel{
		Name:           item.Name,
		Provider:       item.Provider,
		LocalPort:      item.LocalPort,
		StatusState:    item.StatusState,
		PublicURL:      item.PublicURL,
		Visitors:       item.Visitors,
		ActiveSessions: item.ActiveSessions,
		ErrorMsg:       item.ErrorMsg,
		Logs:           item.Logs,
		Width:          width,
		Height:         height,
	}

	return panel.Render()
}
