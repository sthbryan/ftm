//go:build windows

package clipboard

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002

	// Another process can hold the clipboard for a moment while it copies.
	openClipboardWait = time.Second
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procCloseClipboard   = user32.NewProc("CloseClipboard")

	procGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")
	procGlobalFree   = kernel32.NewProc("GlobalFree")
	procLstrcpyW     = kernel32.NewProc("lstrcpyW")
)

// Write puts text on the clipboard through the Win32 API. Shelling out to
// PowerShell cost a few hundred milliseconds, flashed a console window, and
// passed the text as a command-line argument, which broke on quotes, on
// newlines and near the argument length limit.
func Write(text string) error {
	utf16, err := windows.UTF16FromString(text)
	if err != nil {
		return fmt.Errorf("clipboard text is not valid UTF-16: %w", err)
	}

	// The clipboard is owned by the thread that opened it, and a goroutine can
	// resume on a different thread between the open and the close.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := openClipboard(); err != nil {
		return err
	}
	defer func() {
		_, _, _ = procCloseClipboard.Call()
	}()

	if err := callOK("EmptyClipboard", procEmptyClipboard); err != nil {
		return err
	}

	// The clipboard takes ownership of this block on success, so it is only
	// freed on the paths that do not hand it over.
	handle, _, allocErr := procGlobalAlloc.Call(gmemMoveable, uintptr(len(utf16)*2))
	if handle == 0 {
		return fmt.Errorf("GlobalAlloc: %w", lastErr(allocErr))
	}

	block, _, lockErr := procGlobalLock.Call(handle)
	if block == 0 {
		_, _, _ = procGlobalFree.Call(handle)
		return fmt.Errorf("GlobalLock: %w", lastErr(lockErr))
	}

	// lstrcpyW copies the terminating NUL too, and lets the source pointer stay
	// a real pointer instead of round-tripping through a uintptr.
	_, _, _ = procLstrcpyW.Call(block, uintptr(unsafe.Pointer(&utf16[0])))
	_, _, _ = procGlobalUnlock.Call(handle)

	if r, _, setErr := procSetClipboardData.Call(cfUnicodeText, handle); r == 0 {
		_, _, _ = procGlobalFree.Call(handle)
		return fmt.Errorf("SetClipboardData: %w", lastErr(setErr))
	}

	return nil
}

func openClipboard() error {
	deadline := time.Now().Add(openClipboardWait)

	for {
		r, _, err := procOpenClipboard.Call(0)
		if r != 0 {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("OpenClipboard: %w", lastErr(err))
		}

		time.Sleep(time.Millisecond)
	}
}

func callOK(name string, proc *windows.LazyProc) error {
	r, _, err := proc.Call()
	if r == 0 {
		return fmt.Errorf("%s: %w", name, lastErr(err))
	}
	return nil
}

// lastErr turns the always-present syscall.Errno(0) that LazyProc.Call returns
// on success into something readable.
func lastErr(err error) error {
	var errno syscall.Errno
	if errors.As(err, &errno) && errno == 0 {
		return errors.New("the call failed without reporting an error code")
	}
	return err
}
