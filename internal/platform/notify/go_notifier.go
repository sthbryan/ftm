package notify

import (
	"log"
)

func newGoNotification() (Notifier, SoundPlayer) {
	return newTerminalBell(), newTerminalBell()
}

type terminalBell struct{}

func newTerminalBell() *terminalBell {
	return &terminalBell{}
}

func (t *terminalBell) IsAvailable() bool {
	return true
}

func (t *terminalBell) Notify(title, body string) error {
	log.Printf("[bell] %s: %s", title, body)
	return nil
}

func (t *terminalBell) PlaySound(t_ SoundType) error {
	switch t_ {
	case SoundSuccess:
		log.Print("\a[success]")
	case SoundError:
		log.Print("\a[error]")
	case SoundWarning:
		log.Print("\a[warning]")
	case SoundAlert:
		log.Print("\a[alert!]")
	case SoundStartup:
		log.Print("\a[startup]")
	default:
		log.Print("\a")
	}
	return nil
}
