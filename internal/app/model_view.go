package app

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"

	"github.com/sthbryan/ftm/internal/config"
)

func NewModel(app *App) *Model {
	h := help.New()
	h.ShowAll = true

	p := progress.New(
		progress.WithGradient("#c9a227", "#8b7355"),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	m := &Model{
		App:         app,
		Keys:        DefaultKeys,
		Help:        h,
		State:       viewList,
		Cursor:      0,
		ProgressBar: p,
		FormValues: FormData{
			Provider: string(config.ProviderCloudflared),
			Port:     "30000",
		},
	}

	m.refreshItems()
	return m
}

func (m *Model) refreshItems() {
	items := make([]list.Item, 0, len(m.App.Config.Tunnels))
	for _, t := range m.App.Config.Tunnels {
		status := t.Status()
		if s, ok := m.App.Manager.GetStatus(t.ID); ok {
			status = s
		}
		items = append(items, TunnelItem{Tunnel: t, Status: status})
	}
	m.Items = items
}

func (m *Model) selectedItem() (TunnelItem, bool) {
	if m.Cursor < 0 || m.Cursor >= len(m.Items) {
		return TunnelItem{}, false
	}
	item, ok := m.Items[m.Cursor].(TunnelItem)
	return item, ok
}
