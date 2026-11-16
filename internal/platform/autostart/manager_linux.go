//go:build linux

package autostart

import (
	"os"
	"path/filepath"
)

const desktopEntryFile = "ftm-desktop.desktop"

type linuxManager struct{}

func (linuxManager) Supported() bool { return true }

func entryPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "autostart", desktopEntryFile), nil
}

func readEntry() (string, bool, error) {
	path, err := entryPath()
	if err != nil {
		return "", false, err
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}

	return string(contents), true, nil
}

func (linuxManager) Enabled() (bool, error) {
	contents, found, err := readEntry()
	if err != nil || !found {
		return false, err
	}

	return desktopEntryEnabled(contents), nil
}

func (linuxManager) Enable() error {
	path, err := entryPath()
	if err != nil {
		return err
	}

	target, err := execPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(desktopEntry(target)), 0644)
}

func (linuxManager) Disable() error {
	path, err := entryPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (linuxManager) registeredPath() (string, error) {
	contents, found, err := readEntry()
	if err != nil || !found {
		return "", err
	}

	return desktopEntryExec(contents), nil
}

func (m linuxManager) Repair() error {
	return repairPath(m)
}
