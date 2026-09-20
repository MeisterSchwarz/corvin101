//go:build windows

package windows

import (
	"context"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	followInterval = 100 * time.Millisecond

	processQueryLimitedInformation = 0x1000

	hwndTopmost = ^uintptr(0)
)

var (
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")

	enumWindowsCallback = syscall.NewCallback(enumWizardWindow)
)

// findWindowContext wird über lParam an EnumWindows übergeben.
//
// Dadurch können wir einen einzigen statischen Callback verwenden,
// aber trotzdem das Ergebnis eines einzelnen Suchvorgangs speichern.
type findWindowContext struct {
	found windows.Handle
}

// followGameWindow hält das Corvin-Overlay über dem
// Client-Bereich des Wizard101-Fensters.
func (w *Window) followGameWindow(ctx context.Context) {
	ticker := time.NewTicker(followInterval)
	defer ticker.Stop()

	w.syncGameWindow()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			w.syncGameWindow()
		}
	}
}

// syncGameWindow sucht das Wizard101-Fenster und passt das
// Overlay an dessen Client-Bereich an.
func (w *Window) syncGameWindow() {
	gameWindow := findWizardWindow()

	w.mu.RLock()
	overlayWindow := w.hwnd
	w.mu.RUnlock()

	if overlayWindow == 0 {
		return
	}

	if gameWindow == 0 {
		procShowWindow.Call(
			uintptr(overlayWindow),
			swHide,
		)

		return
	}

	var client rect

	result, _, _ :=
		procGetClientRect.Call(
			uintptr(gameWindow),
			uintptr(
				unsafe.Pointer(
					&client,
				),
			),
		)

	if result == 0 {
		return
	}

	topLeft := point{
		X: client.Left,
		Y: client.Top,
	}

	result, _, _ =
		procClientToScreen.Call(
			uintptr(gameWindow),
			uintptr(
				unsafe.Pointer(
					&topLeft,
				),
			),
		)

	if result == 0 {
		return
	}

	width :=
		client.Right -
			client.Left

	height :=
		client.Bottom -
			client.Top

	if width <= 0 ||
		height <= 0 {

		return
	}

	w.mu.Lock()

	sizeChanged :=
		w.width != width ||
			w.height != height

	w.width = width
	w.height = height

	w.mu.Unlock()

	procSetWindowPos.Call(
		uintptr(overlayWindow),
		hwndTopmost,
		uintptr(topLeft.X),
		uintptr(topLeft.Y),
		uintptr(width),
		uintptr(height),
		swpNoActivate|
			swpShowWindow,
	)

	if sizeChanged {
		w.render()
	}
}

// findWizardWindow sucht nach einem sichtbaren Top-Level-Fenster,
// das zu einem bekannten Wizard101-Prozess gehört.
//
// Wichtig:
// enumWindowsCallback wird NICHT hier erzeugt.
// syscall.NewCallback darf nicht bei jedem Poll aufgerufen werden.
func findWizardWindow() windows.Handle {
	search := findWindowContext{}

	procEnumWindows.Call(
		enumWindowsCallback,
		uintptr(
			unsafe.Pointer(
				&search,
			),
		),
	)

	return search.found
}

// enumWizardWindow ist der einmalig registrierte Callback
// für EnumWindows.
func enumWizardWindow(
	hwnd uintptr,
	lParam uintptr,
) uintptr {
	if lParam == 0 {
		return 0
	}

	search :=
		(*findWindowContext)(
			unsafe.Pointer(
				lParam,
			),
		)

	visible, _, _ :=
		procIsWindowVisible.Call(
			hwnd,
		)

	if visible == 0 {
		return 1
	}

	var processID uint32

	procGetWindowThreadProcessId.Call(
		hwnd,
		uintptr(
			unsafe.Pointer(
				&processID,
			),
		),
	)

	if processID == 0 {
		return 1
	}

	processName :=
		processNameByPID(
			processID,
		)

	if !isWizardProcess(
		processName,
	) {
		return 1
	}

	search.found =
		windows.Handle(hwnd)

	// EnumWindows abbrechen, da wir Wizard101 gefunden haben.
	return 0
}

// processNameByPID ermittelt den Dateinamen der ausführbaren
// Datei eines Prozesses.
func processNameByPID(
	processID uint32,
) string {
	handle, err :=
		windows.OpenProcess(
			processQueryLimitedInformation,
			false,
			processID,
		)

	if err != nil {
		return ""
	}

	defer windows.CloseHandle(
		handle,
	)

	buffer :=
		make([]uint16, 1024)

	size :=
		uint32(
			len(buffer),
		)

	result, _, _ :=
		procQueryFullProcessImageNameW.Call(
			uintptr(handle),
			0,
			uintptr(
				unsafe.Pointer(
					&buffer[0],
				),
			),
			uintptr(
				unsafe.Pointer(
					&size,
				),
			),
		)

	if result == 0 {
		return ""
	}

	path :=
		windows.UTF16ToString(
			buffer[:size],
		)

	path =
		strings.ReplaceAll(
			path,
			"/",
			"\\",
		)

	index :=
		strings.LastIndex(
			path,
			"\\",
		)

	if index >= 0 {
		path =
			path[index+1:]
	}

	return path
}

// isWizardProcess enthält die bekannten Namen des
// Wizard101-Clients.
func isWizardProcess(
	name string,
) bool {
	return strings.EqualFold(
		name,
		"WizardGraphicalClient.exe",
	) ||
		strings.EqualFold(
			name,
			"Wizard101.exe",
		)
}
