//go:build windows

package windows

import (
	"context"
	"unsafe"

	"corvin101/internal/overlay/theme"
	"corvin101/internal/overlay/widget"
	"golang.org/x/sys/windows"
)

func panelBounds(width, height int32) widget.Rect {
	return (widget.Placement{Anchor: widget.AnchorCenter, X: 0.5, Y: 0.5, Width: max(1, width*7/10), Height: max(1, height*7/10)}).Bounds(widget.Viewport{Width: width, Height: height})
}

func (w *Window) loadPanelWorlds(ctx context.Context) {
	if w.worldRepository == nil {
		return
	}
	w.mu.Lock()
	w.worldMenuStatus = "Welten werden geladen ..."
	w.mu.Unlock()
	worlds, err := w.worldRepository.ListWorlds(ctx)
	w.mu.Lock()
	w.panelWorlds = worlds
	w.worldMenuStatus = "Keine Welten vorhanden."
	if err != nil {
		w.worldMenuStatus = "Welten konnten nicht geladen werden."
	}
	w.mu.Unlock()
	w.render()
}

func worldCell(bounds widget.Rect, top int32, count, index int) widget.Rect {
	rows := max(1, (count+2)/3)
	padding := max(int32(8), bounds.Width()/80)
	cellWidth := (bounds.Width() - padding*2) / 3
	cellHeight := max(int32(1), min(int32(48), (bounds.Bottom-top-padding)/int32(rows)))
	left := bounds.Left + padding + int32(index/rows)*cellWidth
	y := top + int32(index%rows)*cellHeight
	return widget.Rect{Left: left, Top: y, Right: left + cellWidth - padding, Bottom: y + cellHeight}
}

func (w *Window) drawWorldMenu(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, top int32) {
	w.mu.RLock()
	worlds, selected, status, cursor := w.panelWorlds, w.panelWorld, w.worldMenuStatus, w.cursor
	w.mu.RUnlock()
	if len(worlds) == 0 {
		body := bounds
		body.Top = top
		drawPanelLine(hdc, pixels, width, height, body, status, max(12, height/50))
		return
	}
	fontSize := max(int32(12), min(width, height)/48)
	for i, world := range worlds {
		cell := worldCell(bounds, top, len(worlds), i)
		if world.Key == selected || cell.Contains(cursor) {
			buffer := unsafe.Slice((*uint32)(pixels), int(width*height))
			color := uint32(0xFF3C324B)
			if cell.Contains(cursor) {
				color = 0xFF51405F
			}
			for y := cell.Top; y < cell.Bottom; y++ {
				for x := cell.Left; x < cell.Right; x++ {
					buffer[int(y)*int(width)+int(x)] = color
				}
			}
		}
		label := world.Name
		if world.Key == selected {
			label = "✓ " + label
		}
		cell.Left += 6
		drawPanelLine(hdc, pixels, width, height, cell, label, fontSize)
	}
}

func (w *Window) clickPanel(p widget.Point) {
	w.mu.Lock()
	if !w.panelOpen {
		w.mu.Unlock()
		return
	}
	if w.worldHeader.Contains(p) {
		w.worldMenuOpen = !w.worldMenuOpen
		w.mu.Unlock()
		w.render()
		return
	}
	selected := ""
	if w.worldMenuOpen {
		bounds := panelBounds(w.width, w.height)
		for i, world := range w.panelWorlds {
			if worldCell(bounds, bounds.Top+w.height/12, len(w.panelWorlds), i).Contains(p) {
				selected = world.Key
				break
			}
		}
		w.worldMenuOpen = false
	}
	w.mu.Unlock()
	if selected != "" {
		w.selectPanelWorld(selected)
	}
	w.render()
}

// Only the open panel receives mouse clicks. Keep the game focused.
func (w *Window) setPanelMouseInput(open bool) {
	w.mu.RLock()
	hwnd := w.hwnd
	w.mu.RUnlock()
	style := uintptr(wsExToolWindow | wsExLayered | wsExNoActivate)
	if !open {
		style |= wsExTransparent
	}
	index := int32(-20) // GWL_EXSTYLE
	user32.NewProc("SetWindowLongPtrW").Call(uintptr(hwnd), uintptr(index), style)
}

func panelTextWidth(hdc uintptr, text string, fontSize int32) int32 {
	name, _ := windows.UTF16PtrFromString(theme.UIFontFamily)
	font, _, _ := procCreateFontW.Call(uintptr(-fontSize), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(name)))
	if font == 0 {
		return int32(len([]rune(text))) * fontSize / 2
	}
	defer procDeleteObject.Call(font)
	old, _, _ := procSelectObject.Call(hdc, font)
	defer procSelectObject.Call(hdc, old)
	value, err := windows.UTF16FromString(text)
	if err != nil {
		return 0
	}
	area := rect{}
	procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&value[0])), uintptr(len(value)-1), uintptr(unsafe.Pointer(&area)), dtSingleLine|0x400|0x800)
	return area.Right - area.Left
}
