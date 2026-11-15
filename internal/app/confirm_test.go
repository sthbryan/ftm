package app

import (
	"testing"

	"charm.land/bubbles/v2/list"

	"github.com/sthbryan/ftm/internal/config"
)

func modelWithTunnel(t *testing.T, id, name string) *Model {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Tunnels = []config.TunnelConfig{
		{ID: id, Name: name, Provider: config.ProviderCloudflared, LocalPort: 30000},
	}

	m := &Model{
		App:   testApp(t, testStore(t, cfg)),
		Keys:  DefaultKeys,
		State: viewList,
		Items: []list.Item{TunnelItem{Tunnel: cfg.Tunnels[0]}},
	}

	return m
}

func TestDeleteAsksForConfirmationFirst(t *testing.T) {
	m := modelWithTunnel(t, "t1", "Foundry VTT")

	if _, cmd := m.handleListDelete(); cmd != nil {
		t.Error("delete issued a command before being confirmed")
	}

	if m.State != viewConfirm {
		t.Fatalf("State = %v after pressing delete, want viewConfirm", m.State)
	}
	if m.pendingDeleteName != "Foundry VTT" {
		t.Errorf("pendingDeleteName = %q, want the tunnel name", m.pendingDeleteName)
	}
	if len(m.App.Config().Tunnels) != 1 {
		t.Fatal("the tunnel was removed before the user confirmed")
	}
}

func TestCancellingKeepsTheTunnel(t *testing.T) {
	m := modelWithTunnel(t, "t1", "Foundry VTT")
	m.handleListDelete()

	m.cancelDelete()

	if m.State != viewList {
		t.Errorf("State = %v after cancelling, want viewList", m.State)
	}
	if len(m.App.Config().Tunnels) != 1 {
		t.Error("cancelling still deleted the tunnel")
	}
	if m.pendingDeleteID != "" {
		t.Error("pendingDeleteID survived a cancel")
	}
}

func TestEscapeCancelsTheConfirmation(t *testing.T) {
	m := modelWithTunnel(t, "t1", "Foundry VTT")
	m.handleListDelete()

	m.handleBack()

	if m.State != viewList {
		t.Errorf("State = %v after escape, want viewList", m.State)
	}
	if len(m.App.Config().Tunnels) != 1 {
		t.Error("escape deleted the tunnel")
	}
	if m.pendingDeleteID != "" {
		t.Error("escape left a pending delete behind")
	}
}

func TestConfirmingWithNothingPendingIsANoop(t *testing.T) {
	m := modelWithTunnel(t, "t1", "Foundry VTT")
	m.State = viewConfirm

	m.confirmDelete()

	if len(m.App.Config().Tunnels) != 1 {
		t.Error("confirming with no pending delete removed a tunnel")
	}
	if m.State != viewList {
		t.Errorf("State = %v, want viewList", m.State)
	}
}
