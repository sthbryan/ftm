package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/sthbryan/ftm/internal/core"
	"github.com/sthbryan/ftm/internal/i18n"
)

func testDirs(t *testing.T) Dirs {
	t.Helper()
	root := t.TempDir()
	cfg := filepath.Join(root, "config")
	return Dirs{
		Config:  cfg,
		Bin:     filepath.Join(cfg, "bin"),
		Runtime: filepath.Join(cfg, "run"),
		Legacy:  filepath.Join(root, "legacy"),
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestOpenCreatesDefaultConfigWhenMissing(t *testing.T) {
	d := testDirs(t)

	s, err := Open(d)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	cfg := s.Snapshot()
	if cfg.NotificationsStatus != core.NotificationPending || cfg.Language != core.LanguageAuto {
		t.Errorf("fresh config = %+v, want the defaults", cfg)
	}
	if cfg.Onboarded {
		t.Error("Onboarded = true on a fresh config, want the welcome screen")
	}
	if _, err := os.Stat(d.ConfigFile()); err != nil {
		t.Fatalf("Open did not persist a default config: %v", err)
	}
}

func TestOpenPersistsMigration(t *testing.T) {
	d := testDirs(t)
	writeFile(t, d.ConfigFile(), "version: 1\nlanguage: en\nnotification_sound: true\ntunnels: []\n")

	if _, err := Open(d); err != nil {
		t.Fatalf("Open: %v", err)
	}

	data, err := os.ReadFile(d.ConfigFile())
	if err != nil {
		t.Fatal(err)
	}
	reloaded, _, err := core.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), fmt.Sprintf("version: %d", core.ConfigVersion)) || reloaded.Language != core.LanguageAuto {
		t.Errorf("config on disk = %q, want the migration persisted", data)
	}
}

func TestOpenRepairsDuplicateIDs(t *testing.T) {
	d := testDirs(t)
	writeFile(t, d.ConfigFile(), "version: 4\nlanguage: auto\nonboarded: true\ntunnels:\n"+
		"    - id: tunnel-1786488003\n      name: Curse of Strahd\n      provider: cloudflared\n      local_port: 30000\n"+
		"    - id: tunnel-1786488003\n      name: Mesa dos\n      provider: pinggy\n      local_port: 30001\n")

	s, err := Open(d)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	tunnels := s.Tunnels()
	if tunnels[0].ID != "tunnel-1786488003" || tunnels[1].ID != "mesa-dos" {
		t.Fatalf("ids = %q, %q, want the duplicate renamed", tunnels[0].ID, tunnels[1].ID)
	}

	again, err := Open(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := again.Tunnels()[1].ID; got != "mesa-dos" {
		t.Errorf("id = %q on reload, want the repair persisted", got)
	}
}

func TestOpenRejectsMalformedYAML(t *testing.T) {
	d := testDirs(t)
	writeFile(t, d.ConfigFile(), "tunnels: [oops")

	if _, err := Open(d); err == nil {
		t.Fatal("Open accepted malformed YAML, want an error")
	}
}

func TestOpenMovesTheLegacyConfig(t *testing.T) {
	d := testDirs(t)
	writeFile(t, d.LegacyConfigFile(), "version: 4\nlanguage: es\ntunnels: []\n")

	s, err := Open(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Snapshot().Language; got != "es" {
		t.Errorf("Language = %q, want the legacy config carried over", got)
	}
}

func TestOpenLeavesTheCurrentConfigOverTheLegacyOne(t *testing.T) {
	d := testDirs(t)
	writeFile(t, d.LegacyConfigFile(), "version: 4\nlanguage: es\ntunnels: []\n")
	writeFile(t, d.ConfigFile(), "version: 4\nlanguage: pt\ntunnels: []\n")

	s, err := Open(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Snapshot().Language; got != "pt" {
		t.Errorf("Language = %q, want the current config kept", got)
	}
}

func TestUpdateRoundTripsThroughDisk(t *testing.T) {
	d := testDirs(t)
	s, err := Open(d)
	if err != nil {
		t.Fatal(err)
	}

	want := []core.TunnelConfig{
		{ID: "a", Name: "Foundry VTT", Provider: core.ProviderCloudflared, LocalPort: 30000},
		{ID: "b", Name: "Partida de prueba", Provider: core.ProviderPinggy, LocalPort: 30001},
	}
	err = s.Update(func(c *core.Config) error {
		c.WebPort = 40507
		c.NotificationSound = false
		c.Tunnels = want
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(d)
	if err != nil {
		t.Fatal(err)
	}
	got := reopened.Snapshot()
	if got.WebPort != 40507 || got.NotificationSound {
		t.Errorf("reloaded = port %d sound %v, want 40507 false", got.WebPort, got.NotificationSound)
	}
	for i := range want {
		if got.Tunnels[i] != want[i] {
			t.Errorf("Tunnels[%d] = %+v, want %+v", i, got.Tunnels[i], want[i])
		}
	}

	entries, _ := os.ReadDir(d.Config)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("Update left a temporary file behind: %s", e.Name())
		}
	}
}

func TestUpdateKeepsTheOldConfigWhenTheSaveFails(t *testing.T) {
	d := testDirs(t)
	s := New(d, core.DefaultConfig())

	// A directory where the config belongs makes the final rename fail.
	if err := os.MkdirAll(d.ConfigFile(), 0o755); err != nil {
		t.Fatal(err)
	}

	err := s.Update(func(c *core.Config) error {
		c.Language = "es"
		return nil
	})
	if err == nil {
		t.Fatal("Update succeeded with a directory in the way, want an error")
	}
	if got := s.Snapshot().Language; got != core.LanguageAuto {
		t.Errorf("Language = %q after a failed save, want it unchanged", got)
	}

	entries, _ := os.ReadDir(d.Config)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("a failed save left a temporary file behind: %s", e.Name())
		}
	}
}

func TestUpdateReturnsTheCallbackErrorAndChangesNothing(t *testing.T) {
	d := testDirs(t)
	s := New(d, core.DefaultConfig())
	boom := errors.New("boom")

	err := s.Update(func(c *core.Config) error {
		c.Language = "es"
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Update = %v, want the callback's error", err)
	}
	if got := s.Snapshot().Language; got != core.LanguageAuto {
		t.Errorf("Language = %q, want it unchanged", got)
	}
	if _, err := os.Stat(d.ConfigFile()); !os.IsNotExist(err) {
		t.Errorf("a refused update wrote the config (stat err %v)", err)
	}
}

func TestSnapshotIsACopy(t *testing.T) {
	s := New(testDirs(t), core.DefaultConfig())
	if err := s.Update(func(c *core.Config) error {
		c.AddTunnel(core.TunnelConfig{ID: "a", Name: "A"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	snap := s.Snapshot()
	snap.Tunnels[0].Name = "changed"
	snap.ProviderExpirationMinutes["pinggy"] = 1
	snap.ExpirationThresholds[0] = 99

	again := s.Snapshot()
	if again.Tunnels[0].Name != "A" || again.ProviderExpirationMinutes["pinggy"] != 60 || again.ExpirationThresholds[0] != 30 {
		t.Errorf("changing a snapshot changed the store: %+v", again)
	}

	tunnel, ok := s.Tunnel("a")
	if !ok {
		t.Fatal("Tunnel(a) not found")
	}
	tunnel.Name = "changed"
	if got, _ := s.Tunnel("a"); got.Name != "A" {
		t.Error("changing a returned tunnel changed the store")
	}
}

func TestConcurrentWritersAndReaders(t *testing.T) {
	d := testDirs(t)
	s, err := Open(d)
	if err != nil {
		t.Fatal(err)
	}

	const writers, each = 8, 10
	var wg sync.WaitGroup
	for w := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range each {
				err := s.Update(func(c *core.Config) error {
					c.AddTunnel(core.TunnelConfig{ID: fmt.Sprintf("w%d-%d", w, i), Name: "Mesa"})
					c.WebPort = 40500 + w
					return nil
				})
				if err != nil {
					t.Error(err)
				}
			}
		}()
	}
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range each * 2 {
				for _, tunnel := range s.Snapshot().Tunnels {
					_ = tunnel.Name
				}
				_, _ = s.Tunnel("w0-0")
			}
		}()
	}
	wg.Wait()

	if got := len(s.Tunnels()); got != writers*each {
		t.Fatalf("Tunnels = %d, want %d: an update was lost", got, writers*each)
	}

	reopened, err := Open(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(reopened.Tunnels()); got != writers*each {
		t.Fatalf("Tunnels on disk = %d, want %d", got, writers*each)
	}
}

func TestPeekReadsWithoutWriting(t *testing.T) {
	d := testDirs(t)

	cfg, err := Peek(d)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Language != core.LanguageAuto {
		t.Errorf("Language = %q, want the default", cfg.Language)
	}
	if _, err := os.Stat(d.Config); !os.IsNotExist(err) {
		t.Errorf("Peek created %s, want nothing written", d.Config)
	}

	writeFile(t, d.LegacyConfigFile(), "version: 1\nlanguage: es\ntunnels: []\n")
	cfg, err = Peek(d)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Language != "es" {
		t.Errorf("Language = %q, want the legacy config read", cfg.Language)
	}
	if _, err := os.Stat(d.ConfigFile()); !os.IsNotExist(err) {
		t.Error("Peek migrated the legacy config, want nothing written")
	}
}

// i18n cannot import core, so the "auto" language it resolves is pinned here.
func TestLanguageAutoMatchesI18n(t *testing.T) {
	if core.LanguageAuto != i18n.LangAuto {
		t.Fatalf("core.LanguageAuto = %q, i18n.LangAuto = %q", core.LanguageAuto, i18n.LangAuto)
	}
}
