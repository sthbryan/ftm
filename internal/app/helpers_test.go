package app

import (
	"path/filepath"
	"testing"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/process"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/store"
	"github.com/sthbryan/ftm/internal/web"
)

// testStore wraps cfg in a store that saves under a temp dir.
func testStore(t *testing.T, cfg *config.Config) *store.Store {
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

// testApp wires an App around st the way New does, without the instance lock
// or any background work.
func testApp(t *testing.T, st *store.Store) *App {
	t.Helper()

	manager := process.NewManager()
	bus := events.New()
	t.Cleanup(bus.Close)
	manager.SetEvents(bus)

	a := &App{
		Service: service.New(st, manager, bus),
		Bus:     bus,
		Manager: manager,
	}
	a.WebServer = web.NewServer(manager, a.Service, bus)
	return a
}
