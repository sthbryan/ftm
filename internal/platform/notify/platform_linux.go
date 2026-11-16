//go:build linux

package notify

func newPlatformNotifier() (Notifier, SoundPlayer) {
	return newLinuxNotifier(), newLinuxSoundPlayer()
}
