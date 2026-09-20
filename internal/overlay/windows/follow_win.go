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
)

// followGameWindow hält das Corvin-Overlay über dem
// Client-Bereich des Wizard101-Fensters.
//
// Position und Größe werden regelmäßig aktualisiert, damit das
// Overlay auch bei Verschieben oder Skalieren des Spielfensters
// korrekt sitzt.
func (w *Window) followGameWindow(ctx context.Context) {
	ticker := time.NewTicker(followInterval)
	defer ticker.Stop()

	// Direkt einmal synchronisieren, damit wir nicht erst auf
	// den ersten Tick warten müssen.
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

	// Wizard101-Fenster aktuell nicht gefunden.
	if gameWindow == 0 {
		procShowWindow.Call(
			uintptr(overlayWindow),
			swHide,
		)

		return
	}

	var client rect

	result, _, _ := procGetClientRect.Call(
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

	// GetClientRect liefert Koordinaten relativ zum
	// Wizard101-Fenster. Für SetWindowPos benötigen wir
	// Bildschirmkoordinaten.
	topLeft := point{
		X: client.Left,
		Y: client.Top,
	}

	result, _, _ = procClientToScreen.Call(
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

	width := client.Right - client.Left
	height := client.Bottom - client.Top

	if width <= 0 || height <= 0 {
		return
	}

	// Merken, ob sich die Größe geändert hat.
	//
	// Das ist wichtig, weil unsere Widget-Positionen und
	// Skalierung von der aktuellen Client-Größe abhängen.
	w.mu.Lock()

	sizeChanged :=
		w.width != width ||
			w.height != height

	w.width = width
	w.height = height

	w.mu.Unlock()

	// Overlay exakt über dem Client-Bereich von Wizard101
	// positionieren.
	//
	// SWP_NOACTIVATE verhindert, dass Corvin den Fokus von
	// Wizard101 übernimmt.
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

	// Bei einer Größenänderung müssen wir neu rendern,
	// weil sich Position und Scale der Widgets ändern.
	if sizeChanged {
		w.render()
	}
}

// findWizardWindow sucht nach einem sichtbaren Top-Level-Fenster,
// das zu einem bekannten Wizard101-Prozess gehört.
func findWizardWindow() windows.Handle {
	var found windows.Handle

	callback := syscall.NewCallback(
		func(
			hwnd uintptr,
			lParam uintptr,
		) uintptr {
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

			found =
				windows.Handle(hwnd)

			// Enumeration abbrechen.
			return 0
		},
	)

	procEnumWindows.Call(
		callback,
		0,
	)

	return found
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

	size := uint32(
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

	// Sicherheitshalber beide Slash-Varianten vereinheitlichen.
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
		path = path[index+1:]
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
