//go:build windows

package notifications

import (
	"fmt"
	"os/exec"
)

type windowsNotifier struct{}

func newWindowsNotifier() Notifier {
	return &windowsNotifier{}
}

func (n *windowsNotifier) IsAvailable() bool {
	return true
}

func (n *windowsNotifier) Notify(title, body string) error {
	script := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
$template = '<toast><visual><binding template="ToastText02"><text id="1">%s</text><text id="2">%s</text></binding></visual></toast>'
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($template)
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('ftm').Show($toast)
`, escapePowerShell(title), escapePowerShell(body))
	return exec.Command("powershell", "-Command", script).Run()
}

func escapePowerShell(s string) string {
	result := ""
	for _, c := range s {
		switch c {
		case '"':
			result += "`\""
		case '`':
			result += "``"
		case '$':
			result += "`$"
		default:
			result += string(c)
		}
	}
	return result
}

type windowsSoundPlayer struct{}

func newWindowsSoundPlayer() SoundPlayer {
	return &windowsSoundPlayer{}
}

func (s *windowsSoundPlayer) IsAvailable() bool {
	return true
}

func (s *windowsSoundPlayer) PlaySound(t SoundType) error {
	sounds := map[SoundType]string{
		SoundStartup: ".Windows_NotifyCalendar",
		SoundSuccess: ".Windows_Notify-Calendar",
		SoundError:   ".Windows_GTalk",
		SoundWarning: ".Windows_Notify-Reminder",
		SoundAlert:   ".Windows_Notify-Messaging",
		SoundInfo:    ".Windows_Notify-Chat",
	}
	soundName, ok := sounds[t]
	if !ok {
		return nil
	}
	script := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; [System.Media.SystemSounds]::%s.Play()`, soundName)
	return exec.Command("powershell", "-Command", script).Run()
}
