//go:build windows

package windows

import (
	"corvin101/internal/data/enemies"
	"corvin101/internal/overlay/widget"
	"slices"
	"sort"
	"unsafe"
)

func modeOf(mode string) string {
	if mode == "challenge" {
		return mode
	}
	return "standard"
}
func enemyFamily(e enemies.WorldEnemy) string {
	if e.VariantOf != "" {
		return e.VariantOf
	}
	return e.Key
}
func (w *Window) hasChallengeLocked() bool {
	if w.selectedEnemy == "" {
		return false
	}
	for _, e := range w.panelEnemies {
		if enemyFamily(e) == w.selectedEnemy && e.Mode == "challenge" {
			return true
		}
	}
	return false
}

// Stable family keys keep a single catalog row while its detail data changes.
func (w *Window) modeEntriesLocked() []enemies.WorldEnemy {
	groups := map[string][]enemies.WorldEnemy{}
	for _, e := range w.panelEnemies {
		key := enemyFamily(e)
		groups[key] = append(groups[key], e)
	}
	var result []enemies.WorldEnemy
	for family, variants := range groups {
		sort.Slice(variants, func(i, j int) bool { return variants[i].Key < variants[j].Key })
		chosen := 0
		for i, e := range variants {
			if modeOf(e.Mode) == "standard" {
				chosen = i
				break
			}
		}
		mode, manual := w.enemyModes[family]
		if manual {
			for i, e := range variants {
				if modeOf(e.Mode) == mode {
					chosen = i
					break
				}
			}
		} else {
			found := false
			for i, e := range variants {
				if e.InCombat(w.panelCombatIDs) {
					chosen = i
					found = true
					break
				}
			}
			if !found {
				for i, e := range variants {
					if slices.Contains(e.Locations, w.panelZone) {
						chosen = i
						break
					}
				}
			}
		}
		entry := variants[chosen]
		entry.VariantKey = entry.Key
		entry.Key = family
		entry.GameIDs = slices.Clone(entry.GameIDs)
		for _, e := range variants {
			entry.GameIDs = append(entry.GameIDs, e.Key)
			entry.GameIDs = append(entry.GameIDs, e.GameIDs...)
		}
		result = append(result, entry)
	}
	return result
}
func (w *Window) autoPanelModeLocked() {
	w.enemyModes = nil
	w.encounterHeads = nil
	for _, e := range w.panelEnemies {
		if e.Key == w.selectedEnemy {
			w.selectedEnemy = enemyFamily(e)
			break
		}
	}
	if w.selectedEnemy == "" {
		for _, e := range w.panelEntriesLocked() {
			if e.InCombat(w.panelCombatIDs) {
				w.selectedEnemy = e.Key
				break
			}
		}
	}
}
func (w *Window) setPanelModeLocked(mode string) {
	if !w.hasChallengeLocked() {
		return
	}
	for _, entry := range w.modeEntriesLocked() {
		if entry.Key == w.selectedEnemy {
			if head := w.selectedHeadLocked(entry); head != nil {
				key := head.Key
				if head.VariantOf != "" {
					key = head.VariantOf
				}
				if w.encounterHeads == nil {
					w.encounterHeads = map[string]string{}
				}
				w.encounterHeads[w.selectedEnemy] = key
			}
			break
		}
	}
	if w.enemyModes == nil {
		w.enemyModes = map[string]string{}
	}
	w.enemyModes[w.selectedEnemy] = modeOf(mode)
	w.cheatScroll = 0
	w.cheatScrollMax = 0
	w.cheatWheelDelta = 0
}
func modeButtonRect(width, height int32, hasCheats bool) widget.Rect {
	index := 0
	if hasCheats {
		index = 2
	}
	first := detailTabRect(width, height, index, hasCheats)
	size := first.Width()
	return widget.Rect{Left: first.Left - 8 - size, Right: first.Left - 8, Top: first.Top, Bottom: first.Bottom}
}
func (w *Window) modeButtonAtLocked(p widget.Point) int {
	if !w.panelOpen || w.worldMenuOpen || w.panelLoading || !w.hasChallengeLocked() {
		return -1
	}
	if modeButtonRect(w.width, w.height, w.hasSelectedCheatsLocked()).Contains(p) {
		for _, e := range w.modeEntriesLocked() {
			if e.Key == w.selectedEnemy {
				if modeOf(e.Mode) == "challenge" {
					return 0
				}
				return 1
			}
		}
	}
	return -1
}
func (w *Window) drawModeButtons(hdc uintptr, pixels unsafe.Pointer, width, height int32) {
	w.mu.RLock()
	show := w.hasChallengeLocked()
	hasCheats := w.hasSelectedCheatsLocked()
	cursor := w.cursor
	mode := "standard"
	for _, e := range w.modeEntriesLocked() {
		if e.Key == w.selectedEnemy {
			mode = modeOf(e.Mode)
			break
		}
	}
	w.mu.RUnlock()
	if !show {
		return
	}
	button := modeButtonRect(width, height, hasCheats)
	size := button.Width() * 90 / 100
	if button.Contains(cursor) {
		size = button.Width()
	}
	drawIconImage(pixels, width, height, challengeModeIcon(mode == "challenge"), (button.Left+button.Right)/2, button.Top+(button.Height()-size)/2, size)
}
