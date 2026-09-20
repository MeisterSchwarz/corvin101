//go:build windows

package system

import "golang.org/x/sys/windows"

const singleInstanceMutexName = "Global\\corvin101_SingleInstance"

var mutex windows.Handle

func Lock() bool {
	name, err := windows.UTF16PtrFromString(singleInstanceMutexName)
	if err != nil {
		return false
	}

	handle, err := windows.CreateMutex(nil, true, name)
	if err != nil {
		return false
	}

	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(handle)
		return false
	}

	mutex = handle
	return true
}

func Unlock() {
	if mutex == 0 {
		return
	}

	windows.ReleaseMutex(mutex)
	windows.CloseHandle(mutex)
	mutex = 0
}
