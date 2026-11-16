//go:build windows

package notify

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// appUserModelID matches the identity in cmd/ftm-desktop/winres/winres.json, so a toast
// is attributed to ftm instead of to the PowerShell host process.
const appUserModelID = "com.justcallmebryan.ftm"

var (
	shell32               = windows.NewLazySystemDLL("shell32.dll")
	procSetAppUserModelID = shell32.NewProc("SetCurrentProcessExplicitAppUserModelID")
)

func init() {
	id, err := windows.UTF16PtrFromString(appUserModelID)
	if err != nil {
		return
	}

	// Best effort: without it the toast still fires, it is just attributed to
	// the host process instead of the app.
	_, _, _ = procSetAppUserModelID.Call(uintptr(unsafe.Pointer(id)))
}

// runPowerShell runs a script without flashing a console window. Every
// notification used to open one.
// createNoWindow is CREATE_NO_WINDOW: powershell gets no console at all, so
// nothing can flash before -WindowStyle Hidden applies.
const createNoWindow = 0x08000000

func runPowerShell(script string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return cmd.Run()
}

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
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('%s').Show($toast)
`, escapePowerShell(title), escapePowerShell(body), appUserModelID)

	return runPowerShell(script)
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
	return runPowerShell(script)
}
