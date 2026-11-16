//go:build windows

package provider

import (
	"os/exec"
	"strconv"
	"syscall"
)

// createNoWindow is CREATE_NO_WINDOW, which syscall does not export.
const createNoWindow = 0x08000000

func configureProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// CREATE_NO_WINDOW: the desktop app has no console, so without it every
	// tunnel binary would open a console window of its own.
	cmd.SysProcAttr.CreationFlags |= syscall.CREATE_NEW_PROCESS_GROUP | createNoWindow
	cmd.SysProcAttr.HideWindow = true
}

func taskkill(cmd *exec.Cmd, force bool) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	args := []string{"/T", "/PID", strconv.Itoa(cmd.Process.Pid)}
	if force {
		args = append([]string{"/F"}, args...)
	}

	kill := exec.Command("taskkill", args...)
	kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = kill.Run()
}

func terminateGroup(cmd *exec.Cmd) {
	taskkill(cmd, false)
}

func killGroup(cmd *exec.Cmd) {
	taskkill(cmd, true)
}

// hideWindow keeps a console window from flashing up for short-lived helper
// commands such as `--version`.
func hideWindow(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
}
