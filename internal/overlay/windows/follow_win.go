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

	// GetWindow constants.
	gwHwndPrev = 3

	// IsIconic prüft, ob ein Fenster minimiert ist.
)

var (
	procQueryFullProcessImageNameW = kernel32.NewProc(
		"QueryFullProcessImageNameW",
	)

	procGetWindow = user32.NewProc(
		"GetWindow",
	)

	procIsIconic = user32.NewProc(
		"IsIconic",
	)

	enumWindowsCallback = syscall.NewCallback(
		enumWizardWindow,
	)
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
//
// Wichtig:
//
// Das Overlay wird NICHT global TOPMOST gemacht.
//
// Stattdessen wird es relativ zum Wizard101-Fenster in der
// normalen Windows-Z-Order positioniert.
//
// Dadurch:
//
//   - bleibt das Overlay über Wizard101,
//   - bleibt es sichtbar, wenn auf einem anderen Monitor
//     ein anderes Fenster aktiv ist,
//   - liegt es aber nicht über Fenstern, die Wizard101
//     tatsächlich überdecken.
func (w *Window) followGameWindow(
	ctx context.Context,
) {
	ticker :=
		time.NewTicker(
			followInterval,
		)

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

// syncGameWindow sucht das Wizard101-Fenster und passt
// Position, Größe und Z-Order des Overlays an.
func (w *Window) syncGameWindow() {
	gameWindow :=
		findWizardWindow()

	w.mu.RLock()

	overlayWindow :=
		w.hwnd

	w.mu.RUnlock()

	if overlayWindow == 0 {
		return
	}

	// Wizard101 wurde nicht gefunden.
	if gameWindow == 0 {
		hideOverlay(
			overlayWindow,
		)

		return
	}

	// Ein minimiertes Wizard101 besitzt zwar weiterhin ein HWND,
	// soll aber natürlich kein sichtbares Overlay haben.
	minimized, _, _ :=
		procIsIconic.Call(
			uintptr(
				gameWindow,
			),
		)

	if minimized != 0 {
		hideOverlay(
			overlayWindow,
		)

		return
	}

	var client rect

	result, _, _ :=
		procGetClientRect.Call(
			uintptr(
				gameWindow,
			),
			uintptr(
				unsafe.Pointer(
					&client,
				),
			),
		)

	if result == 0 {
		hideOverlay(
			overlayWindow,
		)

		return
	}

	topLeft := point{
		X: client.Left,
		Y: client.Top,
	}

	result, _, _ =
		procClientToScreen.Call(
			uintptr(
				gameWindow,
			),
			uintptr(
				unsafe.Pointer(
					&topLeft,
				),
			),
		)

	if result == 0 {
		hideOverlay(
			overlayWindow,
		)

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

		hideOverlay(
			overlayWindow,
		)

		return
	}

	w.mu.Lock()

	sizeChanged :=
		w.width != width ||
			w.height != height

	w.width = width
	w.height = height

	w.mu.Unlock()

	// ------------------------------------------------------------
	// Z-Order
	// ------------------------------------------------------------
	//
	// SetWindowPos interpretiert hWndInsertAfter so:
	//
	// Das Overlay wird direkt HINTER diesem Fenster einsortiert.
	//
	// Wir wollen:
	//
	//     Fenster vor Wizard
	//     Overlay
	//     Wizard
	//
	// Deshalb suchen wir das Fenster, das unmittelbar VOR Wizard
	// liegt, und setzen unser Overlay dahinter.
	//
	// Falls Wizard bereits ganz oben in der normalen Z-Order liegt,
	// gibt es kein vorheriges Fenster. In diesem Fall verwenden wir
	// Wizard selbst als InsertAfter-Fenster und korrigieren danach
	// gegebenenfalls die Reihenfolge.
	insertAfter :=
		findWindowDirectlyAbove(
			gameWindow,
			overlayWindow,
		)

	procSetWindowPos.Call(
		uintptr(
			overlayWindow,
		),
		uintptr(
			insertAfter,
		),
		uintptr(
			topLeft.X,
		),
		uintptr(
			topLeft.Y,
		),
		uintptr(
			width,
		),
		uintptr(
			height,
		),
		swpNoActivate|
			swpShowWindow,
	)

	if sizeChanged {
		w.render()
	}
}

// findWindowDirectlyAbove sucht das nächste Fenster oberhalb
// von Wizard101.
//
// Unser eigenes Overlay wird dabei übersprungen.
//
// Das Ergebnis wird als hWndInsertAfter für SetWindowPos
// verwendet.
func findWindowDirectlyAbove(
	gameWindow windows.Handle,
	overlayWindow windows.Handle,
) windows.Handle {
	current :=
		gameWindow

	for {
		previous, _, _ :=
			procGetWindow.Call(
				uintptr(
					current,
				),
				gwHwndPrev,
			)

		if previous == 0 {
			// Wizard ist bereits ganz oben.
			//
			// In diesem Fall positionieren wir das Overlay
			// relativ zu Wizard selbst.
			return gameWindow
		}

		previousWindow :=
			windows.Handle(
				previous,
			)

		// Unser eigenes Overlay darf nicht als Referenz
		// verwendet werden.
		if previousWindow ==
			overlayWindow {

			current =
				previousWindow

			continue
		}

		return previousWindow
	}
}

func hideOverlay(
	overlayWindow windows.Handle,
) {
	procShowWindow.Call(
		uintptr(
			overlayWindow,
		),
		swHide,
	)
}

// findWizardWindow sucht nach einem sichtbaren Top-Level-Fenster,
// das zu einem bekannten Wizard101-Prozess gehört.
//
// Wichtig:
//
// enumWindowsCallback wird NICHT hier erzeugt.
// syscall.NewCallback darf nicht bei jedem Poll aufgerufen werden.
func findWizardWindow() windows.Handle {
	search :=
		findWindowContext{}

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
		windows.Handle(
			hwnd,
		)

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
		make(
			[]uint16,
			1024,
		)

	size :=
		uint32(
			len(
				buffer,
			),
		)

	result, _, _ :=
		procQueryFullProcessImageNameW.Call(
			uintptr(
				handle,
			),
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
