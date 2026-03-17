package clipboard

import (
	"fmt"
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
	cmd := exec.Command("pbcopy")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	if _, err := stdin.Write([]byte(text)); err != nil {
		return err
	}

	stdin.Close()
	return cmd.Wait()
}

func writeLinux(text string) error {
	cmd := exec.Command("wl-copy")
	stdin, err := cmd.StdinPipe()
	if err == nil {
		if err := cmd.Start(); err == nil {
			stdin.Write([]byte(text))
			stdin.Close()
			if err := cmd.Wait(); err == nil {
				return nil
			}
		}
	}

	cmd = exec.Command("xclip", "-selection", "clipboard")
	stdin, err = cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	if _, err := stdin.Write([]byte(text)); err != nil {
		return err
	}

	stdin.Close()
	return cmd.Wait()
}

func writeWindows(text string) error {
	cmd := exec.Command("powershell", "-command", "Set-Clipboard", "-Value", text)
	return cmd.Run()
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

func Read() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return readMac()
	case "linux":
		return readLinux()
	case "windows":
		return readWindows()
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func readMac() (string, error) {
	cmd := exec.Command("pbpaste")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func readLinux() (string, error) {
	cmd := exec.Command("wl-paste")
	out, err := cmd.Output()
	if err == nil {
		return string(out), nil
	}

	cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
	out, err = cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func readWindows() (string, error) {
	cmd := exec.Command("powershell", "-command", "Get-Clipboard")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
