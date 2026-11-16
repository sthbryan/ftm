package cli

import (
	"os"
	"os/exec"
	"syscall"
)

// removeSelf deletes the running binary. Windows keeps a running .exe
// locked, so it is moved aside and a detached cmd deletes it once this
// process has exited.
func removeSelf(path string) error {
	if err := os.Remove(path); err == nil {
		return nil
	}
	old := path + ".old"
	_ = os.Remove(old)
	if err := os.Rename(path, old); err != nil {
		return err
	}
	cmd := exec.Command("cmd.exe", "/C", "ping -n 3 127.0.0.1 >NUL & del /F /Q \""+old+"\"")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x00000008 | 0x00000200}
	return cmd.Start()
}
