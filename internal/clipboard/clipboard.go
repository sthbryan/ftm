package clipboard

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
)

func Write(text string) error {
	switch runtime.GOOS {
	case "darwin":
		return writeMac(text)
	case "linux":
		return writeLinux(text)
	case "windows":
		return writeWindows(text)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func writeMac(text string) error {
	return writeVia("pbcopy", nil, text)
}

func writeLinux(text string) error {
	if err := writeVia("wl-copy", nil, text); err == nil {
		return nil
	}

	return writeVia("xclip", []string{"-selection", "clipboard"}, text)
}

func writeWindows(text string) error {
	cmd := exec.Command("powershell", "-command", "Set-Clipboard", "-Value", text)
	return cmd.Run()
}

// writeVia feeds text to a command's stdin. Every step reports its own error so
// a clipboard helper that exits early cannot look like a successful copy.
func writeVia(name string, args []string, text string) error {
	cmd := exec.Command(name, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	if _, err := io.WriteString(stdin, text); err != nil {
		_ = stdin.Close()
		return err
	}

	if err := stdin.Close(); err != nil {
		return err
	}

	return cmd.Wait()
}

func WriteAll(lines []string) error {
	var text string
	for i, line := range lines {
		if i > 0 {
			text += "\n"
		}
		text += line
	}
	return Write(text)
}
