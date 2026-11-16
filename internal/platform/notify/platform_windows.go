//go:build windows

package notify

func newPlatformNotifier() (Notifier, SoundPlayer) {
	return newWindowsNotifier(), newWindowsSoundPlayer()
}
