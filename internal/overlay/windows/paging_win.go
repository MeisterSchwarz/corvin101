//go:build windows

package windows

import (
	"corvin101/internal/overlay/widget"
	"fmt"
	"unsafe"
)

func pageControls(width, height int32) (previous, next, label widget.Rect) {
	bounds := panelBounds(width, height)
	side := max(int32(24), height/30)
	left := bounds.Left + max(12, width/80)
	top := bounds.Bottom - height/24 - side/2
	previous = widget.Rect{Left: left, Top: top, Right: left + side, Bottom: top + side}
	label = widget.Rect{Left: previous.Right + 8, Top: top, Right: previous.Right + 8 + max(100, width/12), Bottom: top + side}
	next = widget.Rect{Left: label.Right + 4, Top: top, Right: label.Right + 4 + side, Bottom: top + side}
	return
}

// Caller holds w.mu. The selected enemy is independent of page navigation.
func (w *Window) changePanelPageLocked(step int) {
	pages := max(1, (len(w.panelEnemies)+panelRows(w.height)-1)/panelRows(w.height))
	w.panelPage = max(0, min(w.panelPage+step, pages-1))
}

func (w *Window) wheelPanel(p widget.Point, delta int) bool {
	w.mu.Lock()
	list, _ := enemyPanelColumns(w.width, w.height)
	if !w.panelOpen || w.worldMenuOpen || w.panelLoading || !list.Contains(p) {
		w.panelWheelDelta = 0
		w.mu.Unlock()
		return false
	}
	w.panelWheelDelta += delta
	steps := w.panelWheelDelta / 120
	w.panelWheelDelta %= 120
	w.changePanelPageLocked(-steps)
	w.mu.Unlock()
	if steps != 0 {
		w.render()
	}
	return true
}

func (w *Window) drawPageControls(hdc uintptr, pixels unsafe.Pointer, width, height int32, page, pages int, cursor widget.Point) {
	previous, next, label := pageControls(width, height)
	fontSize := max(int32(11), height/60)
	for _, button := range []struct {
		bounds  widget.Rect
		text    string
		enabled bool
	}{{previous, "‹", page > 0}, {next, "›", page < pages-1}} {
		base := [3]int32{38, 31, 43}
		if button.enabled {
			base = [3]int32{65, 49, 47}
		}
		if button.enabled && button.bounds.Contains(cursor) {
			base = [3]int32{100, 74, 44}
		}
		panelGradient(pixels, width, height, button.bounds, base, 7, false)
		color := uint32(0xff514335)
		if button.enabled {
			color = 0xffa4834d
		}
		panelRule(pixels, width, height, button.bounds, color)
		textBounds := button.bounds
		textBounds.Left += button.bounds.Width() / 3
		drawPanelLine(hdc, pixels, width, height, textBounds, button.text, fontSize+6)
	}
	drawPanelLine(hdc, pixels, width, height, label, fmt.Sprintf("Seite %d/%d", page+1, pages), fontSize)
	hint := label
	hint.Left = next.Right + 16
	hint.Right = panelBounds(width, height).Right - 12
	drawPanelLine(hdc, pixels, width, height, hint, "Mausrad über der Liste · Tab schließen", fontSize)
}
