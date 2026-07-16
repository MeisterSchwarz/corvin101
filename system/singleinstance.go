//go:build windows

package system

import "golang.org/x/sys/windows"

var mutex windows.Handle

func Lock() bool {
	name, _ := windows.UTF16PtrFromString("Global\\ravendex_SingleInstance")

	h, err := windows.CreateMutex(nil, true, name)
	if err != nil {
		return false
	}

	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(h)
		return false
	}

	mutex = h
	return true
}

func Unlock() {
	if mutex != 0 {
		windows.CloseHandle(mutex)
		mutex = 0
	}
}
