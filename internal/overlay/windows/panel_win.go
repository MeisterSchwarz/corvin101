//go:build windows

package windows

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"time"
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
	if w.panelZone != zoneKey {
		w.panelPage = 0
	}
	w.panelZone = zoneKey
	w.mu.Unlock()
	if previous == world {
		return
	}
	w.selectPanelWorld(world)
}

func (w *Window) selectPanelWorld(world string) {
	w.deactivateSearch(true)
	w.mu.Lock()
	w.worldMenuOpen = false
	if world == w.panelWorld && !w.panelLoadFailed {
		w.mu.Unlock()
		return
	}
	w.panelWorld, w.panelPage = world, 0
	w.panelLoadGeneration++
	generation := w.panelLoadGeneration
	w.panelLoading = false
	w.panelWheelDelta = 0
	w.panelCombatIDs = combatEnemyIDs(w.lastSnapshot, world)
	w.selectedEnemy = ""
	w.detailDrops = false
	w.searchQuery = ""
	w.panelLoadFailed = false
	w.panelEnemies = nil
	w.panelStatus = "Warte auf die aktuelle Welt ..."
	if world == "" || w.panelRepository == nil {
		w.mu.Unlock()
		return
	}
	if entries, cached := w.panelRepository.CachedWorld(world); cached {
		w.panelEnemies = entries
		w.panelStatus = "Keine Gegner für diese Welt registriert."
		w.mu.Unlock()
		w.render()
		return
	}
	w.panelStatus = "Gegner werden geladen ..."
	w.panelLoading = true
	w.panelLoadStarted = time.Now()
	w.mu.Unlock()
	go func() {
		entries, err := w.panelRepository.ListWorld(context.Background(), world)
		w.mu.Lock()
		if w.panelWorld != world || w.panelLoadGeneration != generation {
			w.mu.Unlock()
			return
		}
		w.panelEnemies = entries
		w.panelLoading = false
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
	return max(1, int((height*7/10-height/6-searchHeight(height)-6+height/40)/max(44, height/20)))
}

func (w *Window) drawEnemyPanel(hdc uintptr, pixels unsafe.Pointer, width, height int32) {
	w.mu.RLock()
	world, entries, status, page := w.panelWorld, w.panelEnemies, w.panelStatus, w.panelPage
	total := len(w.panelEnemies)
	location := w.currentPanelLocationLocked()
	entries = w.panelEntriesLocked()
	selected, cursor := w.selectedEnemy, w.cursor
	combatIDs := w.panelCombatIDs
	loading, started := w.panelLoading, w.panelLoadStarted
	w.mu.RUnlock()
	if world == "" {
		world = "Unbekannte Welt"
		status = "Warte auf die aktuelle Welt ..."
	}
	bounds := panelBounds(width, height)
	border := max(int32(2), min(width, height)/360)
	drawBestiaryFrame(pixels, width, height, bounds)
	scale := float64(min(width, height)) / 1080
	header := bounds
	header.Bottom = header.Top + height/12
	titleSize := max(int32(18), int32(36*scale))
	titlePadding := max(int32(14), int32(18*scale))
	title := bounds
	title.Left += border + titlePadding
	title.Right -= border + titlePadding
	title.Top += border + 4
	title.Bottom = header.Bottom - 4
	w.mu.RLock()
	menuOpen := w.worldMenuOpen
	for _, option := range w.panelWorlds {
		if option.Key == world {
			world = option.Name
			break
		}
	}
	w.mu.RUnlock()
	label := fmt.Sprintf("%s (%d)", world, total)
	// Measure with the same font used by drawPanelLine so the arrow follows the count.
	textWidth := panelFontTextWidth(hdc, label, titleSize, theme.DisplayFontFamily)
	title.Right = min(title.Right, title.Left+textWidth+titleSize+16)
	textBounds := title
	textBounds.Right -= titleSize + 12
	drawBestiaryText(hdc, pixels, width, height, textBounds, label, titleSize, true)
	drawWorldArrow(pixels, width, height, textBounds.Right+10, (title.Top+title.Bottom)/2-titleSize/8, max(8, titleSize/3), menuOpen)
	w.mu.Lock()
	w.worldHeader = title
	w.mu.Unlock()
	if menuOpen {
		w.drawWorldMenu(hdc, pixels, width, height, bounds, header.Bottom)
		return
	}
	if loading {
		drawEnemyLoading(hdc, pixels, width, height, bounds, started)
		return
	}
	w.drawSearch(hdc, pixels, width, height)
	rows := panelRows(height)
	pages := max(1, (len(entries)+rows-1)/rows)
	page = min(page, pages-1)
	list, details := enemyPanelColumns(width, height)
	card := details
	panelGradient(pixels, width, height, card, [3]int32{34, 29, 42}, 9, true)
	panelRule(pixels, width, height, card, 0xff69533c)
	details = panelInset(details, max(10, height/60))
	if len(entries) == 0 {
		body := list
		body.Bottom = body.Top + max(32, height/20)
		if total > 0 {
			status = "Keine Treffer"
		}
		drawPanelLine(hdc, pixels, width, height, body, status, max(12, height/60))
	}
	now := time.Now()
	for i := page * rows; i < min(len(entries), (page+1)*rows); i++ {
		row := enemyPanelRow(list, height, i-page*rows)
		if entries[i].InCombat(combatIDs) {
			drawCombatBestiaryRow(pixels, width, height, row, entries[i].Key == selected, row.Contains(cursor), now)
		} else if enemyInLocation(entries[i], location) {
			drawLocationBestiaryRow(pixels, width, height, row, entries[i].Key == selected, row.Contains(cursor), now)
		} else {
			drawBestiaryRow(pixels, width, height, row, entries[i].Key == selected, row.Contains(cursor))
		}
		row.Left += 10
		name := entries[i].Name
		if entries[i].InCombat(combatIDs) {
			name = "⚔ " + name
		}
		drawEnemyListName(hdc, pixels, width, height, row, name, entries[i].Schools, max(12, int32(18*scale)))
	}
	separator := details
	separator.Left -= 8
	separator.Right = separator.Left + 1
	fillPanelRect(pixels, width, height, separator, 0xFF65536A)
	w.drawEnemyDetails(hdc, pixels, width, height, details, entries, selected)
	w.drawPageControls(hdc, pixels, width, height, page, pages, cursor)
}

func drawEnemyListName(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, name string, schools []enemies.School, fontSize int32) {
	if len(schools) == 0 {
		drawBestiaryText(hdc, pixels, width, height, bounds, name, fontSize, false)
		return
	}
	iconSize := min(bounds.Height()-8, max(int32(20), fontSize+6))
	gap := int32(4)
	iconWidth := int32(len(schools)) * (iconSize + gap)
	textBounds := bounds
	textBounds.Right = max(bounds.Left, bounds.Right-iconWidth-8)
	drawBestiaryText(hdc, pixels, width, height, textBounds, name, fontSize, false)
	left := min(textBounds.Right, bounds.Left+panelFontTextWidth(hdc, name, fontSize, theme.DisplayFontFamily)) + 8
	for _, school := range schools {
		if left+iconSize > bounds.Right {
			break
		}
		drawSchoolIcon(pixels, width, height, school, left+iconSize/2, bounds.Top+(bounds.Height()-iconSize)/2, iconSize)
		left += iconSize + gap
	}
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
	drawPanelText(hdc, pixels, width, height, bounds, text, fontSize, theme.UIFontFamily)
}

func drawPanelText(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, text string, fontSize int32, family string) {
	drawPanelTextColor(hdc, pixels, width, height, bounds, text, fontSize, family, theme.RGB{R: 245, G: 220, B: 160})
}

func drawPanelTextColor(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, text string, fontSize int32, family string, color theme.RGB) {
	name, _ := windows.UTF16PtrFromString(family)
	font, _, _ := procCreateFontW.Call(uintptr(-fontSize), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(name)))
	if font == 0 {
		return
	}
	defer procDeleteObject.Call(font)
	old, _, _ := procSelectObject.Call(hdc, font)
	defer procSelectObject.Call(hdc, old)
	procSetBkMode.Call(hdc, transparentBackground)
	procSetTextColor.Call(hdc, rgb(color.R, color.G, color.B))
	value, err := windows.UTF16FromString(text)
	if err != nil {
		return
	}
	area := rect{Left: bounds.Left, Top: bounds.Top, Right: bounds.Right, Bottom: bounds.Bottom}
	procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&value[0])), uintptr(len(value)-1), uintptr(unsafe.Pointer(&area)), dtVCenter|dtSingleLine|0x8000|0x800)
	ensureTextAlpha(pixels, width, height, bounds)
}
