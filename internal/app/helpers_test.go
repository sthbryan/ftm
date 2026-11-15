package app

import (
	"path/filepath"
	"testing"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/store"
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
