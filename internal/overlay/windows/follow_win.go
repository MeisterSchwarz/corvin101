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

	procIsWindow = user32.NewProc(
		"IsWindow",
	)

	enumWindowsCallback = syscall.NewCallback(
		enumWizardWindow,
	)
)

// wizardWindowLock beschreibt genau den Wizard101-Client,
// an den dieses Overlay aktuell gebunden ist.
//
// Wir speichern sowohl HWND als auch Process-ID.
// Dadurch können wir erkennen, wenn ein altes HWND später
// von Windows wiederverwendet werden sollte.
type wizardWindowLock struct {
	hwnd      windows.Handle
	processID uint32
}

// findWindowContext wird über lParam an EnumWindows übergeben.
type findWindowContext struct {
	found     windows.Handle
	processID uint32
}

// followGameWindow hält das Corvin-Overlay über genau einem
// Wizard101-Fenster.
//
// Sobald ein Wizard101-Fenster gefunden wurde, bleibt Corvin
// an dieses Fenster gebunden.
//
// Weitere Wizard101-Clients werden ignoriert.
//
// Der Lock wird erst aufgehoben, wenn das ursprüngliche Fenster
// nicht mehr existiert oder nicht mehr zum ursprünglichen
// Wizard101-Prozess gehört.
//
// Minimieren löst den Lock ausdrücklich NICHT.
func (w *Window) followGameWindow(ctx context.Context) {
	ticker := time.NewTicker(followInterval)
	defer ticker.Stop()

	var locked wizardWindowLock

	w.syncGameWindow(&locked)

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			w.syncGameWindow(&locked)
		}
	}
}

// syncGameWindow ermittelt das aktuell gelockte Wizard101-Fenster
// und passt Position, Größe und Z-Order des Overlays an.
func (w *Window) syncGameWindow(
	locked *wizardWindowLock,
) {
	w.mu.RLock()
	overlayWindow := w.hwnd
	w.mu.RUnlock()

	if overlayWindow == 0 {
		return
	}

	gameWindow := resolveWizardWindow(locked)

	// Kein Wizard101 verfügbar.
	if gameWindow == 0 {
		hideOverlay(overlayWindow)
		return
	}

	// Minimieren versteckt nur das Overlay.
	// Der Lock auf diesen Wizard bleibt bestehen.
	minimized, _, _ := procIsIconic.Call(
		uintptr(gameWindow),
	)

	if minimized != 0 {
		hideOverlay(overlayWindow)
		return
	}

	var client rect

	result, _, _ := procGetClientRect.Call(
		uintptr(gameWindow),
		uintptr(
			unsafe.Pointer(&client),
		),
	)

	if result == 0 {
		hideOverlay(overlayWindow)
		return
	}

	topLeft := point{
		X: client.Left,
		Y: client.Top,
	}

	result, _, _ = procClientToScreen.Call(
		uintptr(gameWindow),
		uintptr(
			unsafe.Pointer(&topLeft),
		),
	)

	if result == 0 {
		hideOverlay(overlayWindow)
		return
	}

	width := client.Right - client.Left
	height := client.Bottom - client.Top

	if width <= 0 || height <= 0 {
		hideOverlay(overlayWindow)
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
	// Das Overlay wird weiterhin relativ zum gelockten Wizard
	// einsortiert und NICHT global TOPMOST gemacht.
	insertAfter := findWindowDirectlyAbove(
		gameWindow,
		overlayWindow,
	)

	procSetWindowPos.Call(
		uintptr(overlayWindow),
		uintptr(insertAfter),
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

// resolveWizardWindow liefert ausschließlich das aktuell
// gelockte Wizard101-Fenster.
//
// Falls noch kein Lock existiert, wird einmal nach dem ersten
// passenden Wizard101-Fenster gesucht und dieses gespeichert.
//
// Falls der gelockte Client geschlossen wurde, wird der Lock
// entfernt. Danach darf ein anderer Wizard101-Client übernommen
// werden.
func resolveWizardWindow(
	locked *wizardWindowLock,
) windows.Handle {
	if locked == nil {
		return 0
	}

	// Es existiert bereits ein Lock.
	if locked.hwnd != 0 {
		if wizardWindowStillValid(*locked) {
			return locked.hwnd
		}

		// Das ursprüngliche Fenster existiert nicht mehr
		// oder gehört nicht mehr zum ursprünglichen Prozess.
		*locked = wizardWindowLock{}
	}

	// Noch kein Wizard gelockt:
	// ersten passenden Client suchen.
	found := findWizardWindow()

	if found.hwnd == 0 ||
		found.processID == 0 {
		return 0
	}

	locked.hwnd = found.hwnd
	locked.processID = found.processID

	return locked.hwnd
}

// wizardWindowStillValid prüft, ob das gelockte HWND weiterhin
// existiert und noch zum ursprünglichen Prozess gehört.
//
// Wichtig:
// IsWindowVisible wird hier absichtlich NICHT geprüft.
// Ein minimierter oder temporär unsichtbarer Wizard soll den
// Lock nicht verlieren.
func wizardWindowStillValid(
	locked wizardWindowLock,
) bool {
	if locked.hwnd == 0 ||
		locked.processID == 0 {
		return false
	}

	exists, _, _ := procIsWindow.Call(
		uintptr(locked.hwnd),
	)

	if exists == 0 {
		return false
	}

	var currentProcessID uint32

	procGetWindowThreadProcessId.Call(
		uintptr(locked.hwnd),
		uintptr(
			unsafe.Pointer(
				&currentProcessID,
			),
		),
	)

	if currentProcessID == 0 {
		return false
	}

	// HWND gehört noch zum selben Prozess.
	if currentProcessID != locked.processID {
		return false
	}

	// Zusätzlich prüfen wir den Prozessnamen.
	//
	// Damit bleibt der Lock wirklich auf einen Wizard101-Client
	// beschränkt.
	processName := processNameByPID(
		currentProcessID,
	)

	return isWizardProcess(processName)
}

// findWindowDirectlyAbove sucht das nächste Fenster oberhalb
// des gelockten Wizard101-Fensters.
//
// Unser eigenes Overlay wird übersprungen.
func findWindowDirectlyAbove(
	gameWindow windows.Handle,
	overlayWindow windows.Handle,
) windows.Handle {
	current := gameWindow

	for {
		previous, _, _ := procGetWindow.Call(
			uintptr(current),
			gwHwndPrev,
		)

		if previous == 0 {
			return gameWindow
		}

		previousWindow := windows.Handle(
			previous,
		)

		if previousWindow == overlayWindow {
			current = previousWindow
			continue
		}

		return previousWindow
	}
}

func hideOverlay(
	overlayWindow windows.Handle,
) {
	procShowWindow.Call(
		uintptr(overlayWindow),
		swHide,
	)
}

// findWizardWindow sucht nach dem ersten sichtbaren
// Wizard101-Top-Level-Fenster.
//
// Diese Suche wird nur ausgeführt, wenn aktuell KEIN Wizard
// gelockt ist.
func findWizardWindow() wizardWindowLock {
	search := findWindowContext{}

	procEnumWindows.Call(
		enumWindowsCallback,
		uintptr(
			unsafe.Pointer(&search),
		),
	)

	return wizardWindowLock{
		hwnd:      search.found,
		processID: search.processID,
	}
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

	search := (*findWindowContext)(
		unsafe.Pointer(lParam),
	)

	visible, _, _ := procIsWindowVisible.Call(
		hwnd,
	)

	if visible == 0 {
		return 1
	}

	var processID uint32

	procGetWindowThreadProcessId.Call(
		hwnd,
		uintptr(
			unsafe.Pointer(&processID),
		),
	)

	if processID == 0 {
		return 1
	}

	processName := processNameByPID(
		processID,
	)

	if !isWizardProcess(processName) {
		return 1
	}

	search.found = windows.Handle(
		hwnd,
	)
	search.processID = processID

	// Erster Wizard gefunden.
	// Suche sofort beenden.
	return 0
}

// processNameByPID ermittelt den Dateinamen der ausführbaren
// Datei eines Prozesses.
func processNameByPID(
	processID uint32,
) string {
	handle, err := windows.OpenProcess(
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

	buffer := make(
		[]uint16,
		1024,
	)

	size := uint32(
		len(buffer),
	)

	result, _, _ := procQueryFullProcessImageNameW.Call(
		uintptr(handle),
		0,
		uintptr(
			unsafe.Pointer(&buffer[0]),
		),
		uintptr(
			unsafe.Pointer(&size),
		),
	)

	if result == 0 {
		return ""
	}

	path := windows.UTF16ToString(
		buffer[:size],
	)

	path = strings.ReplaceAll(
		path,
		"/",
		"\\",
	)

	index := strings.LastIndex(
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
