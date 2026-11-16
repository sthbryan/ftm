package notify

import (
	"fmt"
	"log"
	"sync"

	"github.com/sthbryan/ftm/internal/i18n"
)

type Notifier interface {
	Notify(title, body string) error
	IsAvailable() bool
}

type SoundPlayer interface {
	PlaySound(t SoundType) error
	IsAvailable() bool
}

type SoundType int

const (
	SoundStartup SoundType = iota
	SoundSuccess
	SoundError
	SoundWarning
	SoundAlert
	SoundInfo
)

func (s SoundType) String() string {
	switch s {
	case SoundStartup:
		return "startup"
	case SoundSuccess:
		return "success"
	case SoundError:
		return "error"
	case SoundWarning:
		return "warning"
	case SoundAlert:
		return "alert"
	case SoundInfo:
		return "info"
	default:
		return "unknown"
	}
}

var (
	notifier             Notifier
	soundPlayer          SoundPlayer
	available            bool
	soundEnabled         bool
	notificationsEnabled bool
	mu                   sync.RWMutex
)

func Init() {
	n, s := newPlatformNotifier()
	if n != nil && n.IsAvailable() {
		notifier = n
		soundPlayer = s
		available = true
		log.Printf("[notifications] Using platform-native")
		return
	}

	n, s = newGoNotification()
	if n != nil && n.IsAvailable() {
		notifier = n
		soundPlayer = s
		available = true
		log.Printf("[notifications] Using fallback")
		return
	}

	available = false
	log.Printf("[notifications] Running in silent mode")
}

func SetSoundEnabled(enabled bool) {
	mu.Lock()
	soundEnabled = enabled
	mu.Unlock()
}

func SetNotificationsEnabled(enabled bool) {
	mu.Lock()
	notificationsEnabled = enabled
	mu.Unlock()
}

func Notify(title, body string) error {
	mu.RLock()
	enabled := notificationsEnabled
	mu.RUnlock()

	if !enabled {
		return nil
	}

	if notifier == nil || !available {
		log.Printf("[notification] %s: %s", title, body)
		return nil
	}

	go func() {
		if err := notifier.Notify(title, body); err != nil {
			log.Printf("[notification] %s: %v", title, err)
		}
	}()

	return nil
}

func Notifyf(title, format string, args ...interface{}) error {
	return Notify(title, fmt.Sprintf(format, args...))
}

func PlaySound(t SoundType) error {
	mu.RLock()
	enabled := soundEnabled
	mu.RUnlock()

	if !enabled {
		return nil
	}

	if soundPlayer == nil || !available {
		return nil
	}

	go func() {
		if err := soundPlayer.PlaySound(t); err != nil {
			log.Printf("[notification] sound: %v", err)
		}
	}()

	return nil
}

func IsAvailable() bool {
	return available
}

func NotifyTunnelOnline(name, url string) {
	_ = Notify("Tunnel Active", fmt.Sprintf("%s - %s", name, url))
	_ = PlaySound(SoundSuccess)
}

func NotifyTunnelError(name, errMsg string) {
	_ = Notify("Tunnel Error", fmt.Sprintf("%s: %s", name, errMsg))
	_ = PlaySound(SoundError)
}

func NotifyTunnelTimeout(name string) {
	_ = Notify("Tunnel Timeout", fmt.Sprintf("%s could not connect", name))
	_ = PlaySound(SoundError)
}

func NotifyTunnelStopped(name string) {
	_ = Notify("Tunnel Stopped", fmt.Sprintf("%s has been stopped", name))
	_ = PlaySound(SoundInfo)
}

func NotifyNewVisitor(name string, visitors int) {
	_ = Notify(i18n.T("notify_visitor_title"), i18n.TF("notify_visitor_body", name, visitors))
}

func NotifyTunnelExpiring(name string, minutes int) {
	if minutes == 1 {
		_ = Notify("Last Minute!", fmt.Sprintf("%s: 1 minute remaining", name))
		_ = PlaySound(SoundAlert)
	} else {
		_ = Notify("Tunnel Expiring", fmt.Sprintf("%s: %d minutes remaining", name, minutes))
		_ = PlaySound(SoundWarning)
	}
}

func NotifyTunnelExpired(name string) {
	_ = Notify("Tunnel Expired", fmt.Sprintf("%s session has ended", name))
	_ = PlaySound(SoundError)
}

func NotifyWelcome() {
	_ = Notify("Welcome!", "Foundry Tunnel Manager started")
	_ = PlaySound(SoundStartup)
}
