package store_test

import (
	"path/filepath"
	"testing"

	"github.com/sthbryan/ftm/internal/provider"
	"github.com/sthbryan/ftm/internal/store"
)

// internal/provider resolves its bin dir from package dirs; this keeps that
// path pinned to the one Dirs value.
func TestProviderBinDirIsDirsBin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AppData", filepath.Join(home, "appdata"))
	t.Setenv("XDG_CONFIG_HOME", "")

	if got, want := provider.BinDir(), store.DefaultDirs().Bin; got != want {
		t.Fatalf("provider.BinDir() = %q, want store.Dirs.Bin %q", got, want)
	}
}
