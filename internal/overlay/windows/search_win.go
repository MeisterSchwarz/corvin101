//go:build windows

package windows

import (
	"corvin101/internal/data/enemies"
	"corvin101/internal/overlay/theme"
	"corvin101/internal/overlay/widget"
	"golang.org/x/sys/windows"
	"strings"
	"unicode"
	"unsafe"
)

func searchHeight(height int32) int32 { return max(28, height/30) }
func searchBounds(width, height int32) widget.Rect {
	list, _ := enemyPanelColumns(width, height)
	return widget.Rect{Left: list.Left, Top: list.Top - searchHeight(height) - 6, Right: list.Right, Bottom: list.Top - 6}
}
func searchClearBounds(width, height int32) widget.Rect {
	b := searchBounds(width, height)
	b.Left = b.Right - b.Height()
	return b
}

func filterPanelEnemies(source []enemies.WorldEnemy, query string) []enemies.WorldEnemy {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return source
	}
	result := make([]enemies.WorldEnemy, 0, len(source))
	for _, enemy := range source {
		match := strings.Contains(strings.ToLower(enemy.Name), query)
		if match {
			result = append(result, enemy)
		}
	}
	return result
}

// Caller holds w.mu; rendering, clicks and pagination share this exact list.
func (w *Window) panelEntriesLocked() []enemies.WorldEnemy {
	return filterPanelEnemies(orderedPanelEnemies(w.panelEnemies, w.panelCombatIDs, w.currentPanelLocationLocked()), w.searchQuery)
}

func (w *Window) drawSearch(hdc uintptr, pixels unsafe.Pointer, width, height int32) {
	w.mu.RLock()
	query, active := w.searchQuery, w.searchActive
	w.mu.RUnlock()
	bounds := searchBounds(width, height)
	panelGradient(pixels, width, height, bounds, [3]int32{31, 26, 39}, 4, false)
	color := uint32(0xff756042)
	if active {
		color = 0xffd5ae64
	}
	panelRule(pixels, width, height, bounds, color)
	line := bounds
	line.Left += 9
	line.Right -= bounds.Height()
	text := query
	if text == "" && !active {
		text = "Gegner suchen …"
	}
	if active {
		text += "|"
	}
	drawPanelTextColor(hdc, pixels, width, height, line, text, max(12, height/60), theme.UIFontFamily, theme.RGB{R: 210, G: 207, B: 201})
	if query != "" {
		clear := searchClearBounds(width, height)
		clear.Left += clear.Width() / 3
		drawPanelLine(hdc, pixels, width, height, clear, "×", max(16, height/45))
	}
}

func (w *Window) clickSearch(p widget.Point) bool {
	w.mu.Lock()
	if !w.panelOpen || w.worldMenuOpen || w.panelLoading || !searchBounds(w.width, w.height).Contains(p) {
		w.mu.Unlock()
		return false
	}
	if searchClearBounds(w.width, w.height).Contains(p) {
		w.searchQuery = ""
		w.panelPage = 0
	}
	w.searchActive = true
	hwnd := w.hwnd
	w.mu.Unlock()
	if hwnd != 0 {
		user32.NewProc("PostMessageW").Call(uintptr(hwnd), wmSearchFocus, 1, 0)
	}
	w.render()
	return true
}

func (w *Window) deactivateSearch(restore bool) {
	w.mu.Lock()
	active := w.searchActive
	w.searchActive = false
	hwnd := w.hwnd
	w.mu.Unlock()
	if !active {
		return
	}
	w.setPanelMouseInput()
	if restore && hwnd != 0 {
		user32.NewProc("PostMessageW").Call(uintptr(hwnd), wmSearchFocus, 0, 0)
	}
	w.render()
}

func (w *Window) applySearchFocus(activate bool) {
	w.mu.RLock()
	hwnd, game, active := w.hwnd, w.gameWindow, w.searchActive
	w.mu.RUnlock()
	w.applyPanelMouseInput()
	if activate && active {
		user32.NewProc("SetForegroundWindow").Call(uintptr(hwnd))
		user32.NewProc("SetFocus").Call(uintptr(hwnd))
	} else if !activate {
		foreground, _, _ := procGetForegroundWindow.Call()
		if game != 0 && foreground == uintptr(hwnd) {
			user32.NewProc("SetForegroundWindow").Call(uintptr(game))
		}
	}
}

func (w *Window) searchCharacter(character rune) {
	w.mu.RLock()
	active := w.searchActive && w.panelOpen
	w.mu.RUnlock()
	if !active {
		return
	}
	if character == 13 || character == 27 {
		w.deactivateSearch(true)
		return
	}
	var pasted string
	if character == 22 {
		pasted = w.searchClipboard()
	}
	w.mu.Lock()
	switch {
	case character == 8:
		runes := []rune(w.searchQuery)
		if len(runes) > 0 {
			w.searchQuery = string(runes[:len(runes)-1])
		}
	case character == 22:
		for _, r := range pasted {
			if unicode.IsPrint(r) && len([]rune(w.searchQuery)) < 96 {
				w.searchQuery += string(r)
			}
		}
	case unicode.IsPrint(character) && len([]rune(w.searchQuery)) < 96:
		w.searchQuery += string(character)
	default:
		w.mu.Unlock()
		return
	}
	w.panelPage = 0
	w.panelWheelDelta = 0
	entries := w.panelEntriesLocked()
	visible := false
	for _, entry := range entries {
		if entry.Key == w.selectedEnemy {
			visible = true
			break
		}
	}
	if !visible {
		w.selectedEnemy = ""
		w.detailDrops = false
	}
	w.mu.Unlock()
	w.render()
}

func (w *Window) searchClipboard() string {
	w.mu.RLock()
	hwnd := w.hwnd
	w.mu.RUnlock()
	ok, _, _ := user32.NewProc("OpenClipboard").Call(uintptr(hwnd))
	if ok == 0 {
		return ""
	}
	defer user32.NewProc("CloseClipboard").Call()
	handle, _, _ := user32.NewProc("GetClipboardData").Call(13)
	if handle == 0 {
		return ""
	}
	pointer, _, _ := kernel32.NewProc("GlobalLock").Call(handle)
	if pointer == 0 {
		return ""
	}
	defer kernel32.NewProc("GlobalUnlock").Call(handle)
	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(pointer)))
}
