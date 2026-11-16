//go:build darwin

package notify

func newPlatformNotifier() (Notifier, SoundPlayer) {
	return newDarwinNotifier(), newDarwinSoundPlayer()
}
