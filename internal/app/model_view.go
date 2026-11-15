package app

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/viewport"

	"github.com/sthbryan/ftm/internal/app/ui"
	"github.com/sthbryan/ftm/internal/config"
)

func NewModel(app *App) *Model {
	h := help.New()
	h.ShowAll = true

	p := progress.New(
		progress.WithColors(ui.ThemeDefault.Gold, ui.ThemeDefault.Bronze),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	logViewport := viewport.New()
	logViewport.SoftWrap = true

	m := &Model{
		App:           app,
		statusUpdates: make(chan config.TunnelStatus, 64),
		Keys:          DefaultKeys,
		Help:          h,
		State:         viewList,
		Cursor:        0,
		LogViewport:   logViewport,
		ProgressBar:   p,
		Draft: TunnelDraft{
			Provider: string(config.ProviderCloudflared),
			Port:     "30000",
		},
	}

	m.refreshItems()
	return m
}

func (m *Model) refreshItems() {
	tunnels := m.App.Store.Tunnels()
	items := make([]list.Item, 0, len(tunnels))
	for _, t := range tunnels {
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
