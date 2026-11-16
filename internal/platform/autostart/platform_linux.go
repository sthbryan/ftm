//go:build linux

package autostart

func newPlatformManager() Manager {
	return linuxManager{}
}
