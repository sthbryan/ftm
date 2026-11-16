//go:build !darwin && !linux && !windows

package autostart

func newPlatformManager() Manager {
	return unsupported{}
}
