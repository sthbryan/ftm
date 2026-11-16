package tui

import (
	"path/filepath"
	"testing"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/store"
	"github.com/sthbryan/ftm/internal/supervisor"
	"github.com/sthbryan/ftm/internal/transport/httpapi"
)

// testStore wraps cfg in a store that saves under a temp dir.
func testStore(t *testing.T, cfg *core.Config) *store.Store {
	t.Helper()

	root := t.TempDir()
	cfgDir := filepath.Join(root, "config")
	return store.New(store.Dirs{
		Config:  cfgDir,
		Bin:     filepath.Join(cfgDir, "bin"),
		Runtime: filepath.Join(cfgDir, "run"),
		Legacy:  filepath.Join(root, "legacy"),
	}, cfg)
}

// testApp wires an App around st the way cmd/ftm does, without the instance
// lock or any background work.
func testApp(t *testing.T, st *store.Store) *Local {
	t.Helper()

	manager := supervisor.NewManager()
	bus := events.New()
	t.Cleanup(bus.Close)
	manager.SetEvents(bus)

	svc := service.New(st, manager, bus)
	return NewLocal(svc, bus, httpapi.NewServer(svc, bus))
}
