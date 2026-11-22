//go:build windows

package windows

import (
	"corvin101/internal/data/enemies"
	"corvin101/internal/overlay/theme"
	"corvin101/internal/overlay/widget"
	"fmt"
	"golang.org/x/sys/windows"
	"unsafe"
)

func cheatButtonRect(width, height int32) widget.Rect {
	return detailTabRect(width, height, 2, true)
}

// Caller holds w.mu.
func (w *Window) hasSelectedCheatsLocked() bool {
	if !w.panelOpen || w.worldMenuOpen || w.panelLoading {
		return false
	}
	for _, enemy := range w.panelEnemies {
		if enemyFamily(enemy) == w.selectedEnemy && enemy.HasCheatSheet() {
			return true
		}
	}
	return false
}

func drawEnemyCheats(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, top, fontSize int32, mechanics []enemies.CheatMechanic, scroll int32, strategy []enemies.CheatNote) (int32, []strategyEnemyLink) {
	var links []strategyEnemyLink
	heading := bounds
	heading.Top = top
	heading.Bottom = top + fontSize*2
	title := "Cheats"
	if len(mechanics) == 0 {
		title = "Strategie"
	}
	drawPanelText(hdc, pixels, width, height, heading, title, fontSize+2, theme.DisplayFontFamily)
	fillPanelRect(pixels, width, height, widget.Rect{Left: bounds.Left, Right: bounds.Right, Top: heading.Bottom, Bottom: heading.Bottom + 1}, 0xff6f563a)
	top = heading.Bottom + 14
	if len(mechanics) == 0 && len(strategy) == 0 {
		drawWrappedCheatText(hdc, pixels, width, height, bounds, top, fontSize, "Für diese Variante sind noch keine Cheats oder Strategien hinterlegt.", theme.RGB{R: 199, G: 195, B: 190})
		return 0, nil
	}
	viewportTop := top
	bounds.Top = top
	top -= scroll
	for _, mechanic := range mechanics {
		start := top
		textBounds := bounds
		textBounds.Left += 14
		if mechanic.Schedule != nil {
			label := "Einmaliger Rundenplan"
			if mechanic.Schedule.Repeat > 0 {
				label = fmt.Sprintf("Zyklus · alle %d Runden", mechanic.Schedule.Repeat)
			}
			top = drawWrappedCheatText(hdc, pixels, width, height, textBounds, top, max(int32(11), fontSize-2), label, theme.RGB{R: 211, G: 181, B: 113}) + 5
		}
		var mechanicLinks []strategyEnemyLink
		top, mechanicLinks = drawStrategyParagraph(hdc, pixels, width, height, textBounds, top, fontSize, enemies.CheatNote{ID: mechanic.ID, Text: mechanic.Text, Enemies: mechanic.Enemies}, theme.RGB{R: 245, G: 220, B: 160})
		links = append(links, mechanicLinks...)
		if mechanic.Schedule != nil {
			for _, event := range mechanic.Schedule.Events {
				timing := "Anfang"
				if event.Timing == "end" {
					timing = "Ende"
				}
				top = drawWrappedCheatText(hdc, pixels, width, height, textBounds, top+6, max(int32(11), fontSize-1), fmt.Sprintf("Runde %d · %s: %s", event.Round, timing, event.Text), theme.RGB{R: 225, G: 207, B: 166})
			}
		}
		if len(mechanic.Notes) > 0 {
			top += fontSize / 2
		}
		for _, note := range mechanic.Notes {
			top = drawWrappedCheatText(hdc, pixels, width, height, textBounds, top, max(int32(11), fontSize-2), note.Text, theme.RGB{R: 199, G: 195, B: 190}) + 5
		}
		fillPanelRect(pixels, width, height, widget.Rect{Left: bounds.Left + 2, Right: bounds.Left + 4, Top: max(bounds.Top, start), Bottom: min(bounds.Bottom, top)}, 0xff786344)
		top += fontSize + 6
	}
	if len(strategy) > 0 {
		if len(mechanics) > 0 {
			top = drawWrappedCheatText(hdc, pixels, width, height, bounds, top, fontSize+2, "Strategie", theme.RGB{R: 245, G: 220, B: 160}, theme.DisplayFontFamily) + 10
		}
		textBounds := bounds
		textBounds.Left += 14
		for _, paragraph := range strategy {
			var paragraphLinks []strategyEnemyLink
			top, paragraphLinks = drawStrategyParagraph(hdc, pixels, width, height, textBounds, top, fontSize, paragraph)
			top += fontSize
			links = append(links, paragraphLinks...)
		}
	}
	contentBottom := top - fontSize - 6
	if len(strategy) > 0 {
		contentBottom = top - fontSize
	}
	maxScroll := max(int32(0), contentBottom+scroll-bounds.Bottom)
	if maxScroll > 0 {
		x := bounds.Right - 2
		fillPanelRect(pixels, width, height, widget.Rect{Left: x, Right: x + 2, Top: viewportTop, Bottom: bounds.Bottom}, 0xff403744)
		viewHeight := bounds.Bottom - viewportTop
		thumb := min(viewHeight, max(int32(20), viewHeight*viewHeight/(viewHeight+maxScroll)))
		y := viewportTop + min(scroll, maxScroll)*(viewHeight-thumb)/maxScroll
		fillPanelRect(pixels, width, height, widget.Rect{Left: x, Right: x + 2, Top: y, Bottom: y + thumb}, 0xffa38a54)
	}
	return maxScroll, links
}

// Main text and notes share a left edge; wrapped lines never hang past a bullet.
func drawWrappedCheatText(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, top, fontSize int32, text string, color theme.RGB, families ...string) int32 {
	familyName := theme.UIFontFamily
	if len(families) > 0 {
		familyName = families[0]
	}
	family, _ := windows.UTF16PtrFromString(familyName)
	font, _, _ := procCreateFontW.Call(uintptr(-fontSize), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(family)))
	if font == 0 {
		return top
	}
	defer procDeleteObject.Call(font)
	old, _, _ := procSelectObject.Call(hdc, font)
	defer procSelectObject.Call(hdc, old)
	procSetBkMode.Call(hdc, transparentBackground)
	procSetTextColor.Call(hdc, rgb(color.R, color.G, color.B))
	value, err := windows.UTF16FromString(text)
	if err != nil {
		return top
	}
	area := rect{Left: bounds.Left, Right: bounds.Right - 4, Top: top, Bottom: bounds.Bottom}
	measured := area
	procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&value[0])), uintptr(len(value)-1), uintptr(unsafe.Pointer(&measured)), 0x10|0x400|0x800)
	if measured.Bottom <= bounds.Top || top >= bounds.Bottom {
		return measured.Bottom
	}
	area.Bottom = min(bounds.Bottom, measured.Bottom)
	saved, _, _ := gdi32.NewProc("SaveDC").Call(hdc)
	if saved == 0 {
		return measured.Bottom
	}
	defer gdi32.NewProc("RestoreDC").Call(hdc, saved)
	gdi32.NewProc("IntersectClipRect").Call(hdc, uintptr(bounds.Left), uintptr(bounds.Top), uintptr(bounds.Right), uintptr(bounds.Bottom))
	procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&value[0])), uintptr(len(value)-1), uintptr(unsafe.Pointer(&area)), 0x10|0x800|0x2000)
	ensureTextAlpha(pixels, width, height, widget.Rect{Left: area.Left, Right: area.Right, Top: max(bounds.Top, area.Top), Bottom: area.Bottom})
	return measured.Bottom
}
