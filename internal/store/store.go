// Package store owns config.yaml: it is the only code that reads or writes it,
// and every access goes through one mutex so the web handlers, the TUI and the
// background loops never race on it.
package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/sthbryan/ftm/internal/config"
	"github.com/sthbryan/ftm/internal/store/dirs"
)

// Dirs gives the config, bin and runtime directories. See package dirs.
type Dirs = dirs.Dirs

// DefaultDirs resolves the directories for the current user and platform.
func DefaultDirs() Dirs { return dirs.Default() }

// Store is a mutex-guarded config.Config backed by config.yaml.
type Store struct {
	mu   sync.RWMutex
	dirs Dirs
	cfg  *config.Config
}

// Open loads config.yaml from d, creating it with defaults when it is missing,
// moving a config left by an older build and persisting any migration.
func Open(d Dirs) (*Store, error) {
	migrateLegacy(d)

	cfg, changed, err := read(d)
	if err != nil {
		return nil, err
	}

	s := &Store{dirs: d, cfg: cfg}
	if changed {
		if err := write(d, cfg); err != nil {
			return nil, fmt.Errorf("failed to persist migrated config: %w", err)
		}
	}
	return s, nil
}

// Peek reads config.yaml without writing anything, for a process that must not
// own the config (a second instance, or a command that only needs the language).
func Peek(d Dirs) (config.Config, error) {
	path := d.ConfigFile()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = d.LegacyConfigFile()
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return *config.DefaultConfig(), nil
	}
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to read config: %w", err)
	}

	cfg, _, err := config.Parse(data)
	if err != nil {
		return config.Config{}, err
	}
	return *cfg, nil
}

// New wraps cfg without reading the disk. Writes still go to d. Meant for
// tests and for callers that already hold a config.
func New(d Dirs, cfg *config.Config) *Store {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &Store{dirs: d, cfg: cfg.Clone()}
}

// Dirs returns the directories this store was opened with.
func (s *Store) Dirs() Dirs { return s.dirs }

// Snapshot returns a deep copy of the current config. Changing it changes
// nothing; use Update for that.
func (s *Store) Snapshot() config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s.cfg.Clone()
}

// Tunnels returns a copy of the saved connections.
func (s *Store) Tunnels() []config.TunnelConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]config.TunnelConfig{}, s.cfg.Tunnels...)
}

// Tunnel returns a copy of the connection with id.
func (s *Store) Tunnel(id string) (config.TunnelConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if t := s.cfg.GetTunnel(id); t != nil {
		return *t, true
	}
	return config.TunnelConfig{}, false
}

// Update applies fn to a copy of the config and saves it. The change only
// becomes visible once it is on disk: if fn or the save fails, the config is
// left as it was and the error is returned. Updates are serialised, so fn sees
// every earlier update.
func (s *Store) Update(fn func(*config.Config) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.cfg.Clone()
	if err := fn(next); err != nil {
		return err
	}
	if err := write(s.dirs, next); err != nil {
		return err
	}
	s.cfg = next
	return nil
}

func read(d Dirs) (*config.Config, bool, error) {
	data, err := os.ReadFile(d.ConfigFile())
	if os.IsNotExist(err) {
		return config.DefaultConfig(), true, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to read config: %w", err)
	}
	return config.Parse(data)
}

// write replaces config.yaml through a sibling temp file and a rename. A crash,
// a full disk or a scanner holding the file then leaves the previous config
// intact instead of a half-written one that cannot be parsed on the next start.
func write(d Dirs, cfg *config.Config) error {
	if err := os.MkdirAll(d.Config, 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	data, err := cfg.Marshal()
	if err != nil {
		return err
	}

	path := d.ConfigFile()
	tmp, err := os.CreateTemp(d.Config, filepath.Base(path)+".tmp*")
	if err != nil {
		return fmt.Errorf("failed to create a temporary config: %w", err)
	}
	tmpPath := tmp.Name()

	fail := func(format string, err error) error {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf(format, err)
	}

	if _, err := tmp.Write(data); err != nil {
		return fail("failed to write config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fail("failed to flush config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close config: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to set config permissions: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to replace config: %w", err)
	}
	return nil
}

// migrateLegacy copies a config written by an older build, which always used
// ~/.config, to the location this build reads. It only runs when the
// destination is missing, so it can never overwrite newer state.
func migrateLegacy(d Dirs) {
	path := d.ConfigFile()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		return
	}

	legacy := d.LegacyConfigFile()
	if legacy == path {
		return
	}

	data, err := os.ReadFile(legacy)
	if err != nil {
		return
	}

	if err := os.MkdirAll(d.Config, 0o755); err != nil {
		return
	}

	_ = os.WriteFile(path, data, 0o644)
}
