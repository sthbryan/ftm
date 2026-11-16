// Package open hands a URL or a folder to the desktop's default handler.
package open

import (
	"os/exec"
	"runtime"
)

// URL opens url in the default browser.
func URL(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return startAndReap(exec.Command("open", url))
	case "windows":
		return startAndReap(exec.Command("rundll32", "url.dll,FileProtocolHandler", url))
	default:
		return startAndReap(exec.Command("xdg-open", url))
	}
}

// Dir opens path in the file manager.
func Dir(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return startAndReap(exec.Command("open", path))
	case "windows":
		return startAndReap(exec.Command("explorer", path))
	default:
		return startAndReap(exec.Command("xdg-open", path))
	}
}

var onReaped = func(*exec.Cmd) {}

// startAndReap starts cmd and waits for it in the background, so a launcher
// that exits never stays behind as a zombie.
func startAndReap(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		_ = cmd.Wait()
		onReaped(cmd)
	}()

	return nil
}
