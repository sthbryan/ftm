//go:build darwin || linux

package clipboard

import (
	"io"
	"os/exec"
)

// writeVia feeds text to a clipboard helper's stdin. Every step reports its own
// error so a helper that exits early cannot look like a successful copy.
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
