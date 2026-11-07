//go:build windows

package system

import (
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sys/windows"
)

const singleInstanceMutexName = "Global\\corvin101_SingleInstance"

var (
	instanceMu sync.Mutex
	mutex      windows.Handle
)

// Keep a named kernel object open for the entire process lifetime. No mutex
// ownership is needed, so Go can move goroutines between Windows threads.
func acquireInstanceMutex(mutexName string) (windows.Handle, error) {
	name, err := windows.UTF16PtrFromString(mutexName)
	if err != nil {
		return 0, err
	}
	handle, err := windows.CreateMutex(nil, false, name)
	if err != nil {
		// CreateMutex returns an open handle even when the object already exists.
		if handle != 0 {
			windows.CloseHandle(handle)
		}
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
			return 0, nil
		}
		return 0, fmt.Errorf("create single-instance mutex: %w", err)
	}
	return handle, nil
}

func Lock() (bool, error) {
	instanceMu.Lock()
	defer instanceMu.Unlock()
	if mutex != 0 {
		return false, nil
	}
	handle, err := acquireInstanceMutex(singleInstanceMutexName)
	if err != nil {
		return false, err
	}
	if handle == 0 {
		return false, nil
	}
	mutex = handle
	return true, nil
}

func Unlock() {
	instanceMu.Lock()
	defer instanceMu.Unlock()
	if mutex != 0 {
		windows.CloseHandle(mutex)
		mutex = 0
	}
}
