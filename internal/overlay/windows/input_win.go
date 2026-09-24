//go:build windows

package windows

import (
	"context"
	"time"
	"unsafe"

	"corvin101/internal/overlay/widget"
)

const (
	hoverPollInterval = 50 * time.Millisecond
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

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if w.updateCursor() {
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
