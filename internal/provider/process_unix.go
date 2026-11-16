//go:build !windows

package provider

import (
	"os/exec"
	"syscall"
)

func configureProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func signalGroup(cmd *exec.Cmd, sig syscall.Signal) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		_ = cmd.Process.Signal(sig)
		return
	}

	_ = syscall.Kill(-pgid, sig)
}

func terminateGroup(cmd *exec.Cmd) {
	signalGroup(cmd, syscall.SIGTERM)
}

func killGroup(cmd *exec.Cmd) {
	signalGroup(cmd, syscall.SIGKILL)
}

func hideWindow(*exec.Cmd) {}
