package providers

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/sthbryan/ftm/internal/config"
)

// BinDir is where ftm keeps the provider binaries it downloads. It lives next
// to config.yaml, so on Windows it follows %AppData% like the rest of the
// app's data.
func BinDir() string {
	return filepath.Join(config.ConfigDir(), "bin")
}

// LegacyBinDir is the directory every provider used before BinDir existed. It
// is still read so installs made by older releases keep working.
func LegacyBinDir() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".config", config.AppName, "bin")
}

// BundledPath returns where ftm's own copy of the named binary lives. It is the
// BinDir path unless only a copy in the legacy directory exists.
func BundledPath(filename string) string {
	return bundledPath(BinDir(), LegacyBinDir(), filename)
}

func bundledPath(binDir, legacyDir, filename string) string {
	current := filepath.Join(binDir, filename)
	if _, err := os.Stat(current); err == nil || binDir == legacyDir {
		return current
	}

	legacy := filepath.Join(legacyDir, filename)
	if _, err := os.Stat(legacy); err == nil {
		return legacy
	}

	return current
}

// IsBundled reports whether path is a copy ftm downloaded itself, as opposed to
// one the user installed. ftm never replaces a binary it does not own.
func IsBundled(path string) bool {
	if path == "" {
		return false
	}
	dir := filepath.Clean(filepath.Dir(path))
	return samePath(dir, BinDir()) || samePath(dir, LegacyBinDir())
}

func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if a == b {
		return true
	}
	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)
	return errA == nil && errB == nil && ra == rb
}

var migrateOnce sync.Once

// MigrateLegacyBinDir moves binaries from the legacy directory into BinDir. It
// only does work where the two differ (Windows), never overwrites a newer copy
// and leaves a file in place when it cannot be moved, since BundledPath still
// finds it there.
func MigrateLegacyBinDir() {
	migrateOnce.Do(func() {
		if err := migrateBinDir(LegacyBinDir(), BinDir()); err != nil {
			log.Printf("providers: could not migrate %s: %v", LegacyBinDir(), err)
		}
	})
}

func migrateBinDir(legacyDir, binDir string) error {
	if samePath(legacyDir, binDir) {
		return nil
	}

	entries, err := os.ReadDir(legacyDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}

		src := filepath.Join(legacyDir, entry.Name())
		dst := filepath.Join(binDir, entry.Name())
		if _, err := os.Stat(dst); err == nil {
			continue
		}

		if err := os.Rename(src, dst); err == nil {
			continue
		}

		// A rename across volumes fails; copy instead and keep the original if
		// even that does not work.
		if err := copyFile(src, dst); err != nil {
			log.Printf("providers: could not migrate %s: %v", src, err)
			continue
		}
		_ = os.Remove(src)
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	tmp := dst + ".migrating"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}

	return os.Rename(tmp, dst)
}
