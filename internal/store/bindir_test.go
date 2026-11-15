package store_test

import (
	"path/filepath"
	"testing"

	"github.com/sthbryan/ftm/internal/providers"
	"github.com/sthbryan/ftm/internal/store"
)

// internal/providers still asks config.ConfigDir() for its bin dir; this keeps
// that path pinned to the one Dirs value.
func TestProviderBinDirIsDirsBin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AppData", filepath.Join(home, "appdata"))
	t.Setenv("XDG_CONFIG_HOME", "")

	if got, want := providers.BinDir(), store.DefaultDirs().Bin; got != want {
		t.Fatalf("providers.BinDir() = %q, want store.Dirs.Bin %q", got, want)
	}
}
