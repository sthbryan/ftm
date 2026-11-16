package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sthbryan/ftm/internal/i18n"
	"github.com/sthbryan/ftm/internal/store/dirs"
)

func isolateHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AppData", filepath.Join(home, "appdata"))
	t.Setenv("XDG_CONFIG_HOME", "")

	return home
}

// The language is read from a config another ftm may own, so it is peeked at
// where an older release left it.
func TestPeekConfigReadsTheSavedLanguage(t *testing.T) {
	home := isolateHome(t)
	dir := filepath.Join(home, ".config", dirs.AppName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("version: 2\nlanguage: es\ntunnels: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := peekConfig()
	if err != nil {
		t.Fatalf("peekConfig() failed: %v", err)
	}
	if cfg.Language != i18n.LangES {
		t.Fatalf("Language = %q, want %q", cfg.Language, i18n.LangES)
	}
}
