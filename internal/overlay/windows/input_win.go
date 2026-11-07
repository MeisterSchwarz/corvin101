//go:build windows

package windows

import (
	"context"
	"corvin101/internal/data/zones"
	"time"
	"unsafe"

	"corvin101/internal/overlay/widget"
)

const (
	hoverPollInterval = 50 * time.Millisecond
)

var (
	procGetAsyncKeyState    = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
)

func (w *Window) trackCursor(
	ctx context.Context,
) {
	ticker :=
		time.NewTicker(
			hoverPollInterval,
		)

	defer ticker.Stop()

	w.updateCursor()
	// Track the physical key edge so holding Tab only toggles once.
	keyState, _, _ := procGetAsyncKeyState.Call(0x09)
	tabWasDown := keyState&0x8000 != 0
	pageWasDown := map[uintptr]bool{}

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			keyState, _, _ := procGetAsyncKeyState.Call(0x09)
			tabDown := keyState&0x8000 != 0
			changed := false
			if tabDown && !tabWasDown {
				opened := false
				foreground, _, _ := procGetForegroundWindow.Call()
				w.mu.Lock()
				if w.gameWindow != 0 && foreground == uintptr(w.gameWindow) {
					w.panelOpen = !w.panelOpen
					opened = w.panelOpen
					w.worldMenuOpen = false
					if opened {
						w.panelPage = 0
					}
					changed = true
				}
				w.mu.Unlock()
				if changed {
					w.setPanelMouseInput(opened)
					if opened {
						w.mu.RLock()
						zone := w.panelZone
						w.mu.RUnlock()
						w.selectPanelWorld(zones.WorldKey(zone))
					}
				}
			}
			tabWasDown = tabDown
			for _, key := range []uintptr{0x21, 0x22} {
				state, _, _ := procGetAsyncKeyState.Call(key)
				down := state&0x8000 != 0
				if down && !pageWasDown[key] {
					foreground, _, _ := procGetForegroundWindow.Call()
					w.mu.Lock()
					if w.panelOpen && w.gameWindow != 0 && foreground == uintptr(w.gameWindow) {
						pages := max(1, (len(w.panelEnemies)+panelRows(w.height)-1)/panelRows(w.height))
						if key == 0x21 {
							w.panelPage--
						} else {
							w.panelPage++
						}
						w.panelPage = max(0, min(w.panelPage, pages-1))
						changed = true
					}
					w.mu.Unlock()
				}
				pageWasDown[key] = down
			}
			if w.updateCursor() || changed {
				w.render()
			}
		}
	}
}

func (w *Window) updateCursor() bool {
	w.mu.RLock()

	hwnd := w.hwnd

	oldCursor := w.cursor
	oldKnown := w.cursorKnown

	w.mu.RUnlock()

	if hwnd == 0 {
		return false
	}

	cursor := point{}

	result, _, _ :=
		procGetCursorPos.Call(
			uintptr(
				unsafe.Pointer(
					&cursor,
				),
			),
		)

	if result == 0 {
		return false
	}

	result, _, _ =
		procScreenToClient.Call(
			uintptr(hwnd),
			uintptr(
				unsafe.Pointer(
					&cursor,
				),
			),
		)

	if result == 0 {
		return false
	}

	newCursor := widget.Point{
		X: cursor.X,
		Y: cursor.Y,
	}

	changed :=
		!oldKnown ||
			oldCursor != newCursor

	if !changed {
		return false
	}

	w.mu.Lock()

	w.cursor = newCursor
	w.cursorKnown = true

	w.mu.Unlock()

	return true
}
