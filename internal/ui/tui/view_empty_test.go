package tui

import (
	"strings"
	"testing"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/i18n"
)

func modelWithoutTunnels(t *testing.T, onboarded bool) *Model {
	t.Helper()

	cfg := core.DefaultConfig()
	cfg.Onboarded = onboarded
	st := testStore(t, cfg)

	m := &Model{
		App:    testApp(t, st),
		Keys:   DefaultKeys,
		State:  viewList,
		Width:  100,
		Height: 30,
	}
	m.refreshPrefs()
	return m
}

func TestFirstRunGetsTheWelcomeScreen(t *testing.T) {
	out := modelWithoutTunnels(t, false).viewList()

	if !strings.Contains(out, i18n.T("welcome_title")) {
		t.Error("a config that never held a connection did not get the welcome screen")
	}
}

func TestDeletingTheLastConnectionGetsTheEmptyHome(t *testing.T) {
	out := modelWithoutTunnels(t, true).viewList()

	if strings.Contains(out, i18n.T("welcome_title")) {
		t.Error("an onboarded config was welcomed as a first run")
	}
	if !strings.Contains(out, i18n.T("add_tunnel_prompt")) {
		t.Error("the empty home did not offer a way to add a connection")
	}
}
