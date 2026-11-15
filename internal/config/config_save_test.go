package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveReplacesTheFileAndLeavesNoTemporaryFiles(t *testing.T) {
	isolateHome(t)

	first := DefaultConfig()
	first.Language = "es"
	if err := first.Save(); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	second := DefaultConfig()
	second.Language = "pt"
	second.WebPort = 40511
	if err := second.Save(); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Language != "pt" || loaded.WebPort != 40511 {
		t.Fatalf("Load() = language %q port %d, want the latest save", loaded.Language, loaded.WebPort)
	}

	entries, err := os.ReadDir(filepath.Dir(ConfigPath()))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ConfigFile+".tmp") {
			t.Errorf("Save left a temporary file behind: %s", entry.Name())
		}
	}
}

func TestSaveCleansUpWhenTheDestinationIsBlocked(t *testing.T) {
	isolateHome(t)

	dir := filepath.Dir(ConfigPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	// A directory where the config belongs makes the final rename fail, which is
	// the one step a partial write could otherwise leave half-applied.
	if err := os.Mkdir(ConfigPath(), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := DefaultConfig().Save(); err == nil {
		t.Fatal("Save() succeeded with a directory in the way, want an error")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ConfigFile+".tmp") {
			t.Errorf("a failed Save left a temporary file behind: %s", entry.Name())
		}
	}
}
