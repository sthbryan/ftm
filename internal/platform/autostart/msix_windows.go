//go:build windows

package autostart

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	startupTaskID    = "FtmAutostart"
	startupTaskClass = "Windows.ApplicationModel.StartupTask"

	appModelErrorNoPackage = 15700

	roInitMultithreaded = 1
	sFalse              = 1
	rpcChangedMode      = 0x80010106

	asyncStarted = 0
	asyncError   = 3

	stateDisabledByUser = 1
	stateEnabled        = 2

	slotQueryInterface = 0
	slotRelease        = 2

	slotAsyncInfoStatus  = 7
	slotAsyncGetResults  = 8
	slotStaticsGetAsync  = 7
	slotTaskRequestEnabl = 6
	slotTaskDisable      = 7
	slotTaskState        = 8

	awaitTimeout  = 15 * time.Second
	awaitInterval = 5 * time.Millisecond
)

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	combase  = windows.NewLazySystemDLL("combase.dll")

	procGetCurrentPackageFullName = kernel32.NewProc("GetCurrentPackageFullName")
	procRoInitialize              = combase.NewProc("RoInitialize")
	procRoUninitialize            = combase.NewProc("RoUninitialize")
	procRoGetActivationFactory    = combase.NewProc("RoGetActivationFactory")
	procWindowsCreateString       = combase.NewProc("WindowsCreateString")
	procWindowsDeleteString       = combase.NewProc("WindowsDeleteString")

	iidStartupTaskStatics = windows.GUID{
		Data1: 0xEE5B60BD,
		Data2: 0xA148,
		Data3: 0x41A7,
		Data4: [8]byte{0xB2, 0x6E, 0xE8, 0xB8, 0x8A, 0x1E, 0x62, 0xF8},
	}

	iidAsyncInfo = windows.GUID{
		Data1: 0x00000036,
		Data2: 0x0000,
		Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}
)

func packaged() bool {
	length := uint32(0)
	ret, _, _ := procGetCurrentPackageFullName.Call(uintptr(unsafe.Pointer(&length)), 0)
	return ret != appModelErrorNoPackage
}

type msixManager struct{}

func (msixManager) Supported() bool { return true }

func (msixManager) Repair() error { return nil }

func failed(hr uintptr) bool {
	return int32(hr) < 0
}

func hresult(op string, hr uintptr) error {
	return fmt.Errorf("%s failed: hresult 0x%08x", op, uint32(hr))
}

func vtableCall(this uintptr, slot uintptr, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(this))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + slot*unsafe.Sizeof(uintptr(0))))

	all := make([]uintptr, 0, len(args)+1)
	all = append(all, this)
	all = append(all, args...)

	hr, _, _ := syscall.SyscallN(fn, all...)
	return hr
}

func release(this uintptr) {
	if this != 0 {
		vtableCall(this, slotRelease)
	}
}

func createString(value string) (uintptr, error) {
	encoded, err := windows.UTF16FromString(value)
	if err != nil {
		return 0, err
	}

	var handle uintptr
	hr, _, _ := procWindowsCreateString.Call(
		uintptr(unsafe.Pointer(&encoded[0])),
		uintptr(len(encoded)-1),
		uintptr(unsafe.Pointer(&handle)),
	)
	if failed(hr) {
		return 0, hresult("WindowsCreateString", hr)
	}

	return handle, nil
}

func deleteString(handle uintptr) {
	if handle != 0 {
		procWindowsDeleteString.Call(handle)
	}
}

func initializeRuntime() (func(), error) {
	runtime.LockOSThread()

	hr, _, _ := procRoInitialize.Call(roInitMultithreaded)
	if failed(hr) && uint32(hr) != rpcChangedMode {
		runtime.UnlockOSThread()
		return nil, hresult("RoInitialize", hr)
	}

	initialized := !failed(hr) && hr != sFalse

	return func() {
		if initialized {
			procRoUninitialize.Call()
		}
		runtime.UnlockOSThread()
	}, nil
}

func startupTaskStatics() (uintptr, error) {
	class, err := createString(startupTaskClass)
	if err != nil {
		return 0, err
	}
	defer deleteString(class)

	var factory uintptr
	hr, _, _ := procRoGetActivationFactory.Call(
		class,
		uintptr(unsafe.Pointer(&iidStartupTaskStatics)),
		uintptr(unsafe.Pointer(&factory)),
	)
	if failed(hr) {
		return 0, hresult("RoGetActivationFactory", hr)
	}

	return factory, nil
}

func awaitOperation(operation uintptr, result unsafe.Pointer) error {
	var info uintptr
	if hr := vtableCall(operation, slotQueryInterface,
		uintptr(unsafe.Pointer(&iidAsyncInfo)),
		uintptr(unsafe.Pointer(&info)),
	); failed(hr) {
		return hresult("QueryInterface(IAsyncInfo)", hr)
	}
	defer release(info)

	deadline := time.Now().Add(awaitTimeout)
	for {
		var status int32
		if hr := vtableCall(info, slotAsyncInfoStatus, uintptr(unsafe.Pointer(&status))); failed(hr) {
			return hresult("IAsyncInfo.Status", hr)
		}

		if status != asyncStarted {
			if status == asyncError {
				return fmt.Errorf("startup task operation reported status %d", status)
			}
			break
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("startup task operation timed out")
		}

		time.Sleep(awaitInterval)
	}

	if hr := vtableCall(operation, slotAsyncGetResults, uintptr(result)); failed(hr) {
		return hresult("IAsyncOperation.GetResults", hr)
	}

	return nil
}

func withStartupTask(fn func(task uintptr) error) error {
	uninitialize, err := initializeRuntime()
	if err != nil {
		return err
	}
	defer uninitialize()

	statics, err := startupTaskStatics()
	if err != nil {
		return err
	}
	defer release(statics)

	taskID, err := createString(startupTaskID)
	if err != nil {
		return err
	}
	defer deleteString(taskID)

	var operation uintptr
	if hr := vtableCall(statics, slotStaticsGetAsync, taskID, uintptr(unsafe.Pointer(&operation))); failed(hr) {
		return hresult("IStartupTaskStatics.GetAsync", hr)
	}
	defer release(operation)

	var task uintptr
	if err := awaitOperation(operation, unsafe.Pointer(&task)); err != nil {
		return err
	}
	defer release(task)

	return fn(task)
}

func (msixManager) Enabled() (bool, error) {
	enabled := false

	err := withStartupTask(func(task uintptr) error {
		var state int32
		if hr := vtableCall(task, slotTaskState, uintptr(unsafe.Pointer(&state))); failed(hr) {
			return hresult("IStartupTask.State", hr)
		}

		enabled = state == stateEnabled
		return nil
	})

	return enabled, err
}

func (msixManager) Enable() error {
	return withStartupTask(func(task uintptr) error {
		var operation uintptr
		if hr := vtableCall(task, slotTaskRequestEnabl, uintptr(unsafe.Pointer(&operation))); failed(hr) {
			return hresult("IStartupTask.RequestEnableAsync", hr)
		}
		defer release(operation)

		var state int32
		if err := awaitOperation(operation, unsafe.Pointer(&state)); err != nil {
			return err
		}

		switch state {
		case stateEnabled:
			return nil
		case stateDisabledByUser:
			return ErrDisabledByUser
		default:
			return fmt.Errorf("startup task stayed disabled (state %d)", state)
		}
	})
}

func (msixManager) Disable() error {
	return withStartupTask(func(task uintptr) error {
		if hr := vtableCall(task, slotTaskDisable); failed(hr) {
			return hresult("IStartupTask.Disable", hr)
		}
		return nil
	})
}
