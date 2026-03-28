//go:build darwin

package notifications

import (
	"fmt"
	"os/exec"
)

type darwinNotifier struct {
	useAlerter bool
}

func newDarwinNotifier() Notifier {
	if err := exec.Command("which", "alerter").Run(); err == nil {
		return &darwinNotifier{useAlerter: true}
	}
	if err := exec.Command("which", "osascript").Run(); err != nil {
		return nil
	}
	return &darwinNotifier{useAlerter: false}
}

func (n *darwinNotifier) IsAvailable() bool { return true }

func (n *darwinNotifier) Notify(title, body string) error {
	if n.useAlerter {
		return exec.Command("alerter",
			"-title", title,
			"-message", body,
		).Run()
	}

	script := fmt.Sprintf(`display notification "%s" with title "%s"`, escapeAppleScript(body), escapeAppleScript(title))
	return exec.Command("osascript", "-e", script).Run()
}

func escapeAppleScript(s string) string {
	result := ""
	for _, c := range s {
		switch c {
		case '\\':
			result += "\\\\"
		case '"':
			result += "\\\""
		default:
			result += string(c)
		}
	}
	return result
}

type darwinSoundPlayer struct {
	sounds map[SoundType]string
}

func newDarwinSoundPlayer() SoundPlayer {
	if err := exec.Command("which", "afplay").Run(); err != nil {
		return nil
	}
	return &darwinSoundPlayer{
		sounds: map[SoundType]string{
			SoundStartup: "/System/Library/Sounds/Glass.aiff",
			SoundSuccess: "/System/Library/Sounds/Tink.aiff",
			SoundError:   "/System/Library/Sounds/Basso.aiff",
			SoundWarning: "/System/Library/Sounds/Ping.aiff",
			SoundAlert:   "/System/Library/Sounds/Blow.aiff",
			SoundInfo:    "/System/Library/Sounds/Funk.aiff",
		},
	}
}

func (s *darwinSoundPlayer) IsAvailable() bool { return true }

func (s *darwinSoundPlayer) PlaySound(t SoundType) error {
	path, ok := s.sounds[t]
	if !ok {
		return nil
	}
	return exec.Command("afplay", path).Run()
}
