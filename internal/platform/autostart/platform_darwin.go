//go:build darwin

package autostart

func newPlatformManager() Manager {
	return darwinManager{}
}
