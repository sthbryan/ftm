//go:build darwin

package autostart

import (
	"os"
	"path/filepath"
)

type darwinManager struct{}

func (darwinManager) Supported() bool { return true }

func agentPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", appLabel+".plist"), nil
}

func (darwinManager) Enabled() (bool, error) {
	path, err := agentPath()
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (m darwinManager) Enable() error {
	path, err := agentPath()
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

	return os.WriteFile(path, []byte(launchAgent(target)), 0644)
}

func (darwinManager) Disable() error {
	path, err := agentPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (m darwinManager) registeredPath() (string, error) {
	path, err := agentPath()
	if err != nil {
		return "", err
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return launchAgentProgram(string(contents)), nil
}

func (m darwinManager) Repair() error {
	return repairPath(m)
}
