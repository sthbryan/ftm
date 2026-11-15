package app

import (
	"strings"
	"testing"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/web"
)

func modelWithoutTunnels(t *testing.T, onboarded bool) *Model {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Onboarded = onboarded
	st := testStore(t, cfg)

	return &Model{
		App:    &App{Store: st, WebServer: web.NewServer(nil, st)},
		Keys:   DefaultKeys,
		State:  viewList,
		Width:  100,
		Height: 30,
	}
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
