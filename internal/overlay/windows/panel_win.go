//go:build windows

package windows

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"unsafe"

	"corvin101/internal/data/enemies"
	"corvin101/internal/data/zones"
	"corvin101/internal/overlay/theme"
	"corvin101/internal/overlay/widget"
	"golang.org/x/sys/windows"
)

func (w *Window) updatePanelWorld(zoneKey string) {
	world := zones.WorldKey(zoneKey)
	w.mu.RLock()
	previous := zones.WorldKey(w.panelZone)
	w.mu.RUnlock()
	w.mu.Lock()
	w.panelZone = zoneKey
	w.mu.Unlock()
	if previous == world {
		return
	}
	w.selectPanelWorld(world)
}

func (w *Window) selectPanelWorld(world string) {
	w.mu.Lock()
	w.worldMenuOpen = false
	if world == w.panelWorld && !w.panelLoadFailed {
		w.mu.Unlock()
		return
	}
	w.panelWorld, w.panelPage = world, 0
	w.panelLoadFailed = false
	w.panelEnemies = nil
	w.panelStatus = "Warte auf die aktuelle Welt ..."
	if world == "" || w.panelRepository == nil {
		w.mu.Unlock()
		return
	}
	w.panelStatus = "Gegner werden geladen ..."
	w.mu.Unlock()
	go func() {
		entries, err := w.panelRepository.ListWorld(context.Background(), world)
		w.mu.Lock()
		if w.panelWorld != world {
			w.mu.Unlock()
			return
		}
		w.panelEnemies = entries
		w.panelLoadFailed = err != nil
		w.panelStatus = "Keine Gegner für diese Welt registriert."
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				w.panelStatus = "Für diese Welt ist noch keine Gegnerdatei vorhanden."
			} else {
				w.panelStatus = "Gegnerdaten konnten nicht geladen werden."
			}
			log.Printf("[overlay] load enemies for %s: %v", world, err)
		}
		w.mu.Unlock()
		w.render()
	}()
}

func panelRows(height int32) int {
	return max(1, int((height*7/10-height/6)/max(44, height/20)))
}

func (w *Window) drawEnemyPanel(hdc uintptr, pixels unsafe.Pointer, width, height int32) {
	w.mu.RLock()
	world, entries, status, page := w.panelWorld, w.panelEnemies, w.panelStatus, w.panelPage
	w.mu.RUnlock()
	if world == "" {
		world = "Unbekannte Welt"
		status = "Warte auf die aktuelle Welt ..."
	}
	bounds := panelBounds(width, height)
	buffer := unsafe.Slice((*uint32)(pixels), int(width*height))
	border := max(int32(2), min(width, height)/360)
	for y := bounds.Top; y < bounds.Bottom; y++ {
		for x := bounds.Left; x < bounds.Right; x++ {
			color := uint32(0xFF201D30)
			if x < bounds.Left+border || x >= bounds.Right-border || y < bounds.Top+border || y >= bounds.Bottom-border {
				color = 0xFFDBB965
			}
			buffer[int(y)*int(width)+int(x)] = color
		}
	}
	scale := float64(min(width, height)) / 1080
	header := bounds
	header.Bottom = header.Top + height/12
	titleSize := max(int32(18), int32(36*scale))
	titlePadding := max(int32(8), int32(10*scale))
	title := bounds
	title.Left += border + titlePadding
	title.Right -= border + titlePadding
	title.Top += border + titlePadding
	title.Bottom = title.Top + titleSize + 6
	w.mu.RLock()
	menuOpen := w.worldMenuOpen
	for _, option := range w.panelWorlds {
		if option.Key == world {
			world = option.Name
			break
		}
	}
	w.mu.RUnlock()
	label := fmt.Sprintf("%s (%d)", world, len(entries))
	// Measure with the same font used by drawPanelLine so the arrow follows the count.
	textWidth := panelTextWidth(hdc, label, titleSize)
	title.Right = min(title.Right, title.Left+textWidth+titleSize+16)
	arrow := "▾"
	if menuOpen {
		arrow = "▴"
	}
	drawPanelLine(hdc, pixels, width, height, title, label+"  "+arrow, titleSize)
	w.mu.Lock()
	w.worldHeader = title
	w.mu.Unlock()
	if menuOpen {
		w.drawWorldMenu(hdc, pixels, width, height, bounds, header.Bottom)
		return
	}
	rows := panelRows(height)
	pages := max(1, (len(entries)+rows-1)/rows)
	page = min(page, pages-1)
	top := header.Bottom
	rowHeight := max(int32(44), height/20)
	padding := max(int32(10), width/80)
	if len(entries) == 0 {
		body := bounds
		body.Top = header.Bottom
		body.Bottom -= height / 12
		drawOutlinedText(hdc, pixels, width, height, body, status, max(14, int32(24*scale)), scale, 0)
	}
	for i := page * rows; i < min(len(entries), (page+1)*rows); i++ {
		row := widget.Rect{Left: bounds.Left + padding, Top: top, Right: bounds.Right - padding, Bottom: top + rowHeight/2}
		drawPanelLine(hdc, pixels, width, height, row, entries[i].Name, max(12, int32(21*scale)))
		row.Top = row.Bottom
		row.Bottom = top + rowHeight
		drawPanelAffinities(hdc, pixels, width, height, row, entries[i].Affinities, max(11, int32(16*scale)))
		top += rowHeight
	}
	footer := bounds
	footer.Top = bounds.Bottom - height/12
	text := fmt.Sprintf("Seite %d/%d · Bild ↑/↓ blättern · Tab schließen", page+1, pages)
	drawPanelLine(hdc, pixels, width, height, footer, text, max(11, int32(18*scale)))
}

func drawPanelAffinities(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, affinities map[enemies.School]int, fontSize int32) {
	if len(affinities) == 0 {
		drawPanelLine(hdc, pixels, width, height, bounds, "Keine Affinitäten hinterlegt", fontSize)
		return
	}
	keys := make([]string, 0, len(affinities))
	for school := range affinities {
		keys = append(keys, string(school))
	}
	sort.Strings(keys)
	cellWidth := min(fontSize*6, bounds.Width()/int32(len(keys)))
	iconSize := min(bounds.Height(), fontSize+6, cellWidth/2)
	gap := max(int32(2), fontSize/4)
	for index, key := range keys {
		school := enemies.School(key)
		left := bounds.Left + int32(index)*cellWidth
		valueBounds := bounds
		valueBounds.Left = left + iconSize + gap
		valueBounds.Right = left + cellWidth - gap
		drawPanelLine(hdc, pixels, width, height, valueBounds, fmt.Sprintf("%+d%%", affinities[school]), fontSize)
		drawSchoolIcon(pixels, width, height, school, left+iconSize/2, bounds.Top+(bounds.Height()-iconSize)/2, iconSize)
	}
}

func drawPanelLine(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, text string, fontSize int32) {
	name, _ := windows.UTF16PtrFromString(theme.UIFontFamily)
	font, _, _ := procCreateFontW.Call(uintptr(-fontSize), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(name)))
	if font == 0 {
		return
	}
	defer procDeleteObject.Call(font)
	old, _, _ := procSelectObject.Call(hdc, font)
	defer procSelectObject.Call(hdc, old)
	procSetBkMode.Call(hdc, transparentBackground)
	procSetTextColor.Call(hdc, rgb(245, 220, 160))
	value, err := windows.UTF16FromString(text)
	if err != nil {
		return
	}
	area := rect{Left: bounds.Left, Top: bounds.Top, Right: bounds.Right, Bottom: bounds.Bottom}
	procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&value[0])), uintptr(len(value)-1), uintptr(unsafe.Pointer(&area)), dtVCenter|dtSingleLine|0x8000|0x800)
	ensureTextAlpha(pixels, width, height, bounds)
}
