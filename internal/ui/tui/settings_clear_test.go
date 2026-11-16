package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/store"
)

func modelWithTunnels(t *testing.T, names ...string) *Model {
	t.Helper()

	cfg := core.DefaultConfig()
	for _, name := range names {
		cfg.AddTunnel(core.TunnelConfig{ID: cfg.NewTunnelID(name), Name: name, LocalPort: 30000})
	}
	st := testStore(t, cfg)

	m := &Model{
		App:    testApp(t, st),
		Keys:   DefaultKeys,
		State:  viewList,
		Width:  100,
		Height: 30,
	}
	m.refreshItems()

	return m
}

func TestSettingsShowsTheConnectionCount(t *testing.T) {
	m := modelWithTunnels(t, "First", "Second")
	m.openSettings()

	if got := m.SettingsView.TunnelCount; got != 2 {
		t.Fatalf("TunnelCount = %d, want 2", got)
	}

	out := m.viewSettings()
	if !strings.Contains(out, i18n.T("reset_connections")) {
		t.Error("the settings panel does not offer clearing every connection")
	}
}

func TestClearingAsksBeforeDeleting(t *testing.T) {
	m := modelWithTunnels(t, "First", "Second")
	m.openSettings()
	m.SettingsView.Focused = 3
	m.handleSettingsSelect()

	if m.State != viewConfirm {
		t.Fatalf("State = %v, want the confirmation", m.State)
	}

	out := m.viewConfirm()
	if !strings.Contains(out, i18n.T("confirm_reset_title")) {
		t.Error("the confirmation does not name what is about to happen")
	}
	if strings.Contains(out, i18n.T("confirm_delete_title")) {
		t.Error("clearing every connection reused the single-delete copy")
	}
	if len(m.App.Config().Tunnels) != 2 {
		t.Error("connections were removed before the user confirmed")
	}
}

func TestClearingUsesTheSingularCopyForOneConnection(t *testing.T) {
	m := modelWithTunnels(t, "Only one")
	m.pendingConfirm = confirmClearTunnels

	if got := m.clearTunnelsMessage(); got != i18n.T("confirm_reset_body_one") {
		t.Errorf("message = %q, want the singular copy", got)
	}
}

func TestConfirmingClearsEveryConnection(t *testing.T) {
	m := modelWithTunnels(t, "First", "Second")
	m.openSettings()
	m.SettingsView.Focused = 3
	m.handleSettingsSelect()
	m.Cursor = 1

	m.handleConfirmKey(tea.KeyPressMsg{Code: 'y', Text: "y"})

	if len(m.App.Config().Tunnels) != 0 {
		t.Fatalf("Tunnels = %+v, want empty", m.App.Config().Tunnels)
	}
	if len(m.Items) != 0 {
		t.Errorf("Items = %d, want the list rebuilt as empty", len(m.Items))
	}
	if m.Cursor != 0 {
		t.Errorf("Cursor = %d, want it back at the top", m.Cursor)
	}
	if m.State != viewList {
		t.Errorf("State = %v, want the list so the empty home is visible", m.State)
	}
	if m.Message != i18n.T("connections_cleared") {
		t.Errorf("Message = %q, want the cleared confirmation", m.Message)
	}
	if !m.App.Config().Onboarded {
		t.Error("Onboarded = false, want the empty home and not the welcome screen")
	}

	saved, err := store.Peek(m.App.Service.Dirs())
	if err != nil {
		t.Fatalf("Peek() failed: %v", err)
	}
	if len(saved.Tunnels) != 0 {
		t.Errorf("persisted Tunnels = %+v, want empty", saved.Tunnels)
	}
}

func TestCancellingGoesBackToSettings(t *testing.T) {
	m := modelWithTunnels(t, "First")
	m.openSettings()
	m.SettingsView.Focused = 3
	m.handleSettingsSelect()

	m.handleConfirmKey(tea.KeyPressMsg{Code: 'n', Text: "n"})

	if m.State != viewSettings {
		t.Errorf("State = %v, want to land back on the settings panel", m.State)
	}
	if len(m.App.Config().Tunnels) != 1 {
		t.Errorf("Tunnels = %+v, want them untouched", m.App.Config().Tunnels)
	}
}

func TestClearingIsANoOpWithoutConnections(t *testing.T) {
	m := modelWithTunnels(t)
	m.openSettings()
	m.SettingsView.Focused = 3
	m.handleSettingsSelect()

	if m.State != viewSettings {
		t.Errorf("State = %v, want the settings panel to stay put", m.State)
	}
}
