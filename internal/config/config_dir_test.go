package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDirUsesAppDataOnWindows(t *testing.T) {
	got := configDir("windows", `C:\Users\bryan`, `C:\Users\bryan\AppData\Roaming`)
	want := filepath.Join(`C:\Users\bryan\AppData\Roaming`, AppName)

	if got != want {
		t.Errorf("configDir(windows) = %q, want %q", got, want)
	}
}

func TestConfigDirKeepsDotConfigOnUnix(t *testing.T) {
	for _, goos := range []string{"linux", "darwin"} {
		t.Run(goos, func(t *testing.T) {
			got := configDir(goos, "/home/bryan", "/home/bryan/.config")
			want := filepath.Join("/home/bryan", ".config", AppName)

			if got != want {
				t.Errorf("configDir(%s) = %q, want %q", goos, got, want)
			}
		})
	}
}

func TestConfigDirFallsBackWithoutAnAppDataDirectory(t *testing.T) {
	got := configDir("windows", `/home/bryan`, "")
	want := filepath.Join(`/home/bryan`, ".config", AppName)

	if got != want {
		t.Errorf("configDir without %%AppData%% = %q, want the legacy layout %q", got, want)
	}
}

func TestMigrateLegacyConfigCopiesTheOldFile(t *testing.T) {
	isolateHome(t)

	legacyDir := legacyConfigDir()
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "version: 4\nlanguage: es\ntunnels: []\n"
	if err := os.WriteFile(filepath.Join(legacyDir, ConfigFile), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "nested", ConfigFile)
	migrateLegacyConfig(dest)

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("migration did not write %s: %v", dest, err)
	}
	if string(got) != content {
		t.Errorf("migrated config = %q, want %q", got, content)
	}
}

func TestMigrateLegacyConfigLeavesTheCurrentFileAlone(t *testing.T) {
	isolateHome(t)

	legacyDir := legacyConfigDir()
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, ConfigFile), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	dest := filepath.Join(dir, ConfigFile)
	current := "version: 4\nlanguage: pt\ntunnels: []\n"
	if err := os.WriteFile(dest, []byte(current), 0o644); err != nil {
		t.Fatal(err)
	}

	migrateLegacyConfig(dest)

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != current {
		t.Errorf("migration overwrote the current config with %q", got)
	}
}
