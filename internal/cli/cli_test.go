package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/i18n"
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

func writeConfig(t *testing.T, home, body string) {
	t.Helper()

	dir := filepath.Join(home, ".config", config.AppName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestInitAppliesConfiguredLanguage(t *testing.T) {
	home := isolateHome(t)
	writeConfig(t, home, "version: 2\nlanguage: es\ntunnels: []\n")

	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	if got := i18n.CurrentLanguage(); got != i18n.LangES {
		t.Fatalf("CurrentLanguage() = %q, want %q", got, i18n.LangES)
	}
}

func TestInitFallsBackWhenConfigIsUnreadable(t *testing.T) {
	home := isolateHome(t)
	writeConfig(t, home, "tunnels: [oops")

	if err := Init(); err != nil {
		t.Fatalf("Init() returned an error for a broken config: %v, want a fallback", err)
	}

	got := i18n.CurrentLanguage()
	if got != i18n.LangEN && got != i18n.LangES {
		t.Fatalf("CurrentLanguage() = %q, want a supported language", got)
	}
}

func TestInitLoadsTranslations(t *testing.T) {
	isolateHome(t)

	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	if got := i18n.T("connections"); got == "connections" {
		t.Fatal("translations were not loaded: T(connections) echoed the key")
	}
}
