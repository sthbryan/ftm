//go:build !(darwin || dragonfly || freebsd || linux || netbsd || openbsd || windows)

package instance

import "os"

// Platforms without flock or LockFileEx run without the single-instance lock.
func tryLock(*os.File) error { return nil }

func unlock(*os.File) error { return nil }
