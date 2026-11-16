package tui

import (
	"context"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/viewport"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/ui/tui/ui"
)

// NewModel returns the TUI over b, subscribed to its events. Close ends the
// subscription.
func NewModel(app Backend) *Model {
	h := help.New()
	h.ShowAll = true

	p := progress.New(
		progress.WithColors(ui.ThemeDefault.Accent, ui.ThemeDefault.AccentMuted),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	logViewport := viewport.New()
	logViewport.SoftWrap = true

	m := &Model{
		App:         app,
		Keys:        DefaultKeys,
		Help:        h,
		State:       viewList,
		Cursor:      0,
		LogViewport: logViewport,
		ProgressBar: p,
		Draft: TunnelDraft{
			Provider: string(core.ProviderCloudflared),
			Port:     "30000",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.ctx, m.stopEvents = ctx, cancel
	m.events = app.Events(ctx)

	m.refreshItems()
	m.refreshPrefs()
	m.followSelection()
	if app.Attached() {
		m.showMessage(i18n.T("tui_attached"))
	}
	return m
}

// Close ends the subscription to the backend's events.
func (m *Model) Close() {
	if m.stopEvents != nil {
		m.stopEvents()
	}
}

// refreshItems reads the saved connections and their state again. A failed
// read keeps what is on screen.
func (m *Model) refreshItems() {
	ctx, cancel := m.call()
	defer cancel()
	tunnels, err := m.App.Tunnels(ctx)
	if err != nil {
		return
	}
	items := make([]list.Item, 0, len(tunnels))
	for _, t := range tunnels {
		items = append(items, t)
	}
	m.Items = items
	if m.Cursor >= len(m.Items) {
		m.Cursor = max(len(m.Items)-1, 0)
	}
}

// refreshPrefs reads the settings again. A failed read keeps the last ones.
func (m *Model) refreshPrefs() {
	ctx, cancel := m.call()
	defer cancel()
	if p, err := m.App.Prefs(ctx); err == nil {
		m.prefs = p
	}
}

func (m *Model) selectedItem() (TunnelItem, bool) {
	if m.Cursor < 0 || m.Cursor >= len(m.Items) {
		return TunnelItem{}, false
	}
	item, ok := m.Items[m.Cursor].(TunnelItem)
	return item, ok
}
