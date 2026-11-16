//go:build linux

package notify

import (
	"os/exec"
)

type linuxNotifier struct{}

func newLinuxNotifier() Notifier {
	cmd := exec.Command("which", "notify-send")
	if err := cmd.Run(); err != nil {
		return nil
	}
	return &linuxNotifier{}
}

func (n *linuxNotifier) IsAvailable() bool {
	return true
}

func (n *linuxNotifier) Notify(title, body string) error {
	return exec.Command("notify-send", "-u", "normal", "-a", "ftm", title, body).Run()
}

type linuxSoundPlayer struct {
	player string
}

func newLinuxSoundPlayer() SoundPlayer {
	players := []string{"paplay", "aplay", "play"}
	for _, p := range players {
		if exec.Command("which", p).Run() == nil {
			return &linuxSoundPlayer{player: p}
		}
	}
	return nil
}

func (s *linuxSoundPlayer) IsAvailable() bool {
	return s != nil && s.player != ""
}

func (s *linuxSoundPlayer) PlaySound(t SoundType) error {
	sounds := map[SoundType]string{
		SoundStartup: "/usr/share/sounds/ubuntu/stereo/desktop-login.ogg",
		SoundSuccess: "/usr/share/sounds/ubuntu/stereo/button-toggle-on.ogg",
		SoundError:   "/usr/share/sounds/ubuntu/stereo/dialog-error.ogg",
		SoundWarning: "/usr/share/sounds/ubuntu/stereo/button-toggle-off.ogg",
		SoundAlert:   "/usr/share/sounds/ubuntu/stereo/alert.ogg",
		SoundInfo:    "/usr/share/sounds/ubuntu/stereo/button-pressed.ogg",
	}
	path, ok := sounds[t]
	if !ok {
		return nil
	}
	return exec.Command(s.player, path).Run()
}
