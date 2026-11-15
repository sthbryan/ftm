package web

import (
	"path/filepath"
	"testing"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/events"
	"github.com/sthbryan/ftm/internal/process"
	"github.com/sthbryan/ftm/internal/service"
	"github.com/sthbryan/ftm/internal/store"
)

// newTestServer wires a server the way the app does, with the config saved
// under a temp dir. manager may be nil for handlers that never touch it.
func newTestServer(t *testing.T, manager *process.Manager, cfg *config.Config) *Server {
	t.Helper()

	root := t.TempDir()
	st := store.New(store.Dirs{
		Config: filepath.Join(root, "config"),
		Legacy: filepath.Join(root, "legacy"),
	}, cfg)

	bus := events.New()
	t.Cleanup(bus.Close)

	var sup service.Supervisor = nopSupervisor{}
	if manager != nil {
		sup = manager
		manager.SetEvents(bus)
	}
	return NewServer(manager, service.New(st, sup, bus), bus)
}

type nopSupervisor struct{}

func (nopSupervisor) Start(config.TunnelConfig) error                { return nil }
func (nopSupervisor) Stop(string) error                              { return nil }
func (nopSupervisor) StopAll()                                       {}
func (nopSupervisor) CheckInstallation(config.Provider) (bool, bool) { return false, false }
func (nopSupervisor) InstallProvider(config.Provider) error          { return nil }
