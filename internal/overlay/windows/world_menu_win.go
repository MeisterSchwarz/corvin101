//go:build windows

package windows

import (
	"context"
	"slices"
	"unsafe"

	"corvin101/internal/data/zones"
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
	worlds = slices.DeleteFunc(worlds, func(world zones.WorldInfo) bool {
		return world.Key == "Housing" || world.Key == "Minigames"
	})
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
		drawBestiaryRow(pixels, width, height, cell, world.Key == selected, cell.Contains(cursor))
		label := world.Name
		if world.Key == selected {
			label = "✓ " + label
		}
		cell.Left += 6
		drawPanelLine(hdc, pixels, width, height, cell, label, fontSize)
	}
}

func (w *Window) clickPanel(p widget.Point) {
	if w.clickSearch(p) {
		return
	}
	w.deactivateSearch(true)
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
	if index := w.modeButtonAtLocked(p); index >= 0 {
		mode := "standard"
		if index == 1 {
			mode = "challenge"
		}
		w.setPanelModeLocked(mode)
		w.mu.Unlock()
		w.render()
		return
	}
	if link := w.strategyLinkAtLocked(p); link != nil {
		family, mode := link.Family, link.Mode
		if link.Head != "" {
			if w.encounterHeads == nil {
				w.encounterHeads = map[string]string{}
			}
			w.encounterHeads[family] = link.Head
		}
		w.selectedEnemy = family
		if w.enemyModes == nil {
			w.enemyModes = map[string]string{}
		}
		w.enemyModes[family] = mode
		w.searchQuery = ""
		w.detailCheats = false
		w.detailDrops = false
		w.cheatScroll = 0
		w.panelPage = 0
		for i, e := range w.panelEntriesLocked() {
			if e.Key == family {
				w.panelPage = i / panelRows(w.height)
				break
			}
		}
		w.mu.Unlock()
		w.render()
		return
	}
	if head := w.headButtonAtLocked(p); head != "" {
		if w.encounterHeads == nil {
			w.encounterHeads = map[string]string{}
		}
		w.encounterHeads[w.selectedEnemy] = head
		for _, entry := range w.modeEntriesLocked() {
			if entry.Key == w.selectedEnemy {
				for _, member := range entry.Heads {
					if member.Key == head && member.VariantOf != "" {
						w.encounterHeads[w.selectedEnemy] = member.VariantOf
					}
				}
			}
		}
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
	} else {
		if w.panelLoading {
			w.mu.Unlock()
			return
		}
		previous, next, _ := pageControls(w.width, w.height)
		if tab := w.detailTabAtLocked(p); tab >= 0 {
			w.detailDrops = tab == 1
			w.detailCheats = tab == 2
			w.mu.Unlock()
			w.render()
			return
		}
		if category := w.dropCategoryAtLocked(p); category != "" {
			w.dropCategory = category
			w.mu.Unlock()
			w.render()
			return
		}
		if previous.Contains(p) || next.Contains(p) {
			step := 1
			if previous.Contains(p) {
				step = -1
			}
			w.changePanelPageLocked(step)
			w.mu.Unlock()
			w.render()
			return
		}
		entries := w.panelEntriesLocked()
		rows := panelRows(w.height)
		page := min(w.panelPage, max(0, (len(entries)-1)/rows))
		list, _ := enemyPanelColumns(w.width, w.height)
		for i := page * rows; i < min(len(entries), (page+1)*rows); i++ {
			if enemyPanelRow(list, w.height, i-page*rows).Contains(p) {
				if w.selectedEnemy != entries[i].Key {
					w.detailDrops = false
					w.detailCheats = false
					w.cheatScroll = 0
					w.cheatScrollMax = 0
					w.cheatWheelDelta = 0
					w.dropCategory = ""
				}
				w.selectedEnemy = entries[i].Key
				break
			}
		}
	}
	w.mu.Unlock()
	if selected != "" {
		w.selectPanelWorld(selected)
	}
	w.render()
}

// Only the open panel receives mouse clicks. Keep the game focused.
func (w *Window) setPanelMouseInput() {
	w.mu.RLock()
	hwnd := w.hwnd
	w.mu.RUnlock()
	if hwnd != 0 {
		user32.NewProc("PostMessageW").Call(uintptr(hwnd), wmPanelInput, 0, 0)
	}
}

func (w *Window) applyPanelMouseInput() {
	w.mu.RLock()
	hwnd := w.hwnd
	open := w.panelOpen
	searchActive := w.searchActive
	w.mu.RUnlock()
	style := uintptr(wsExToolWindow | wsExLayered | wsExNoActivate)
	if searchActive {
		style &^= wsExNoActivate
	}
	if !open {
		style |= wsExTransparent
	}
	index := int32(-20) // GWL_EXSTYLE
	user32.NewProc("SetWindowLongPtrW").Call(uintptr(hwnd), uintptr(index), style)
}

func panelTextWidth(hdc uintptr, text string, fontSize int32) int32 {
	return panelFontTextWidth(hdc, text, fontSize, theme.UIFontFamily)
}

func panelFontTextWidth(hdc uintptr, text string, fontSize int32, family string) int32 {
	name, _ := windows.UTF16PtrFromString(family)
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
