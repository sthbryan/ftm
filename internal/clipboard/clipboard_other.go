//go:build !darwin && !linux && !windows

package clipboard

import (
	"fmt"
	"runtime"
)

func Write(string) error {
	return fmt.Errorf("clipboard is not supported on %s", runtime.GOOS)
}
