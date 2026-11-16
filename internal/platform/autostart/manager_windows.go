//go:build windows

package autostart

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

const (
	runKey      = `Software\Microsoft\Windows\CurrentVersion\Run`
	approvedKey = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
	runValue    = appName
)

type registryManager struct{}

func (registryManager) Supported() bool { return true }

func (m registryManager) Enabled() (bool, error) {
	command, err := m.runCommand()
	if err != nil || command == "" {
		return false, err
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, approvedKey, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return true, nil
		}
		return false, err
	}
	defer key.Close()

	approval, _, err := key.GetBinaryValue(runValue)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return true, nil
		}
		return false, err
	}

	return !startupApprovalDisabled(approval), nil
}

func (registryManager) runCommand() (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	defer key.Close()

	command, _, err := key.GetStringValue(runValue)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	return command, nil
}

func (registryManager) Enable() error {
	target, err := execPath()
	if err != nil {
		return err
	}

	key, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if err := key.SetStringValue(runValue, quoteWindowsCommand(target)); err != nil {
		return err
	}

	return clearApproval()
}

func clearApproval() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, approvedKey, registry.SET_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}
	defer key.Close()

	if err := key.DeleteValue(runValue); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}

	return nil
}

func (registryManager) Disable() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}
	defer key.Close()

	if err := key.DeleteValue(runValue); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}

	return clearApproval()
}

func (m registryManager) registeredPath() (string, error) {
	command, err := m.runCommand()
	if err != nil {
		return "", err
	}

	return unquoteWindowsCommand(command), nil
}

func (m registryManager) Repair() error {
	return repairPath(m)
}
