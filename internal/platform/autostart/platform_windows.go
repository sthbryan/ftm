//go:build windows

package autostart

func newPlatformManager() Manager {
	if packaged() {
		return msixManager{}
	}
	return registryManager{}
}
