//go:build windows

package windows

import (
	"corvin101/internal/data/enemies"
	"corvin101/internal/overlay/widget"
	"unsafe"
)

// Caller holds w.mu.
func (w *Window) selectedHeadLocked(enemy enemies.WorldEnemy) *enemies.EncounterHead {
	if len(enemy.Heads) == 0 {
		return nil
	}
	if key := w.encounterHeads[enemy.Key]; key != "" {
		for i := range enemy.Heads {
			if enemy.Heads[i].Key == key || enemy.Heads[i].VariantOf == key {
				return &enemy.Heads[i]
			}
		}
	}
	for i := range enemy.Heads {
		head := enemy.Heads[i]
		if (enemies.WorldEnemy{Key: head.Key, EnemyInfo: head.EnemyInfo}).InCombat(w.panelCombatIDs) {
			return &enemy.Heads[i]
		}
	}
	return &enemy.Heads[0]
}
func headButtonRects(width, height int32, heads []enemies.EncounterHead) []widget.Rect {
	_, bounds := enemyPanelColumns(width, height)
	bounds = panelInset(bounds, max(10, height/60))
	fontSize := max(int32(12), min(width, height)/50)
	rowHeight := max(int32(30), fontSize*2)
	iconSize := rowHeight - 4
	left, top := bounds.Left, bounds.Top+fontSize*5
	var result []widget.Rect
	for _, head := range heads {
		size := max(int32(1), int32(len(head.Schools)))*(iconSize+4) + 10
		result = append(result, widget.Rect{Left: left, Right: left + size, Top: top, Bottom: top + rowHeight})
		left += size + max(int32(14), fontSize)
	}
	return result
}
func drawHeadButtons(hdc uintptr, pixels unsafe.Pointer, width, height int32, heads []enemies.EncounterHead, selected string, cursor widget.Point) {
	buttons := headButtonRects(width, height, heads)
	for i, head := range heads {
		b := buttons[i]
		size := b.Height() - 4
		left := b.Left + 5
		for _, school := range head.Schools {
			drawSchoolIcon(pixels, width, height, school, left+size/2, b.Top+2, size)
			left += size + 4
		}
		if head.Key == selected {
			fillPanelRect(pixels, width, height, widget.Rect{Left: b.Left + 3, Right: b.Right - 3, Top: b.Bottom - 1, Bottom: b.Bottom}, 0xffb49a60)
		}
		if i+1 < len(heads) {
			separator := b
			separator.Left = b.Right + 4
			separator.Right = buttons[i+1].Left - 3
			drawPanelLine(hdc, pixels, width, height, separator, "|", max(int32(14), b.Height()/2))
		}
	}
}
func (w *Window) headButtonAtLocked(p widget.Point) string {
	if !w.panelOpen || w.panelLoading || w.worldMenuOpen || w.detailCheats || w.detailDrops {
		return ""
	}
	for _, enemy := range w.modeEntriesLocked() {
		if enemy.Key == w.selectedEnemy {
			for i, b := range headButtonRects(w.width, w.height, enemy.Heads) {
				if b.Contains(p) {
					return enemy.Heads[i].Key
				}
			}
		}
	}
	return ""
}
