//go:build !windows

package cli

import "os"

func removeSelf(path string) error { return os.Remove(path) }
