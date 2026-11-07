//go:build windows

package windows

import (
	"maps"
	"sort"
	"strings"
	"unsafe"

	"corvin101/internal/data/enemies"
	"corvin101/internal/data/zones"
	"corvin101/internal/game/events"
	"corvin101/internal/game/state"
	"corvin101/internal/overlay/theme"
	"corvin101/internal/overlay/widget"
)

func combatEnemyIDs(snapshot state.Snapshot, world string) map[string]bool {
	ids := make(map[string]bool)
	if !snapshot.Combat.Active || world != zones.WorldKey(snapshot.Game.ZoneKey) {
		return ids
	}
	localTeam, teamKnown := 0, false
	for _, p := range snapshot.Combat.Participants {
		if p.Kind == events.ParticipantLocalPlayer {
			localTeam, teamKnown = p.Team, true
			break
		}
	}
	for _, p := range snapshot.Combat.Participants {
		if p.Kind == events.ParticipantMob && p.MobID != "" && (!teamKnown || p.Team != localTeam) {
			ids[p.MobID] = true
		}
	}
	return ids
}

func (w *Window) updatePanelCombat(snapshot state.Snapshot) {
	w.mu.Lock()
	defer w.mu.Unlock()
	ids := combatEnemyIDs(snapshot, w.panelWorld)
	if !maps.Equal(ids, w.panelCombatIDs) {
		w.panelPage = 0
	}
	w.panelCombatIDs = ids
}

func orderedPanelEnemies(source []enemies.WorldEnemy, combat map[string]bool) []enemies.WorldEnemy {
	result := append([]enemies.WorldEnemy(nil), source...)
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if combat[a.Key] != combat[b.Key] {
			return combat[a.Key]
		}
		an, bn := strings.ToLower(a.Name), strings.ToLower(b.Name)
		if an == bn {
			return a.Key < b.Key
		}
		return an < bn
	})
	return result
}

func enemyPanelColumns(width, height int32) (widget.Rect, widget.Rect) {
	bounds := panelBounds(width, height)
	padding := max(int32(10), width/80)
	list := bounds
	list.Left += padding
	list.Right = bounds.Left + bounds.Width()*46/100
	list.Top += height / 12
	list.Bottom -= height / 12
	details := list
	details.Left = list.Right + padding
	details.Right = bounds.Right - padding
	return list, details
}

func enemyPanelRow(list widget.Rect, height int32, index int) widget.Rect {
	rowHeight := max(int32(44), height/20)
	list.Top += int32(index) * rowHeight
	list.Bottom = list.Top + rowHeight
	return list
}

func fillPanelRect(pixels unsafe.Pointer, width, height int32, bounds widget.Rect, color uint32) {
	buffer := unsafe.Slice((*uint32)(pixels), int(width*height))
	for y := max(0, bounds.Top); y < min(height, bounds.Bottom); y++ {
		for x := max(0, bounds.Left); x < min(width, bounds.Right); x++ {
			buffer[int(y)*int(width)+int(x)] = color
		}
	}
}

func (w *Window) drawEnemyDetails(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, entries []enemies.WorldEnemy, selected string) {
	fontSize := max(int32(12), min(width, height)/50)
	var enemy *enemies.WorldEnemy
	for i := range entries {
		if entries[i].Key == selected {
			enemy = &entries[i]
			break
		}
	}
	line := bounds
	line.Bottom = line.Top + fontSize*2
	if enemy == nil {
		drawPanelText(hdc, pixels, width, height, line, "Bestiarium", fontSize+8, theme.DisplayFontFamily)
		line.Top = line.Bottom + fontSize
		line.Bottom = line.Top + fontSize*2
		drawPanelLine(hdc, pixels, width, height, line, "Wähle links einen Gegner aus.", fontSize)
		return
	}
	drawPanelText(hdc, pixels, width, height, line, enemy.Name, fontSize+6, theme.DisplayFontFamily)
	line.Top = line.Bottom
	line.Bottom += fontSize * 2
	drawPanelLine(hdc, pixels, width, height, line, enemy.Key, max(11, fontSize-3))
	line.Top = line.Bottom + fontSize
	line.Bottom = line.Top + fontSize*2
	drawPanelLine(hdc, pixels, width, height, line, "Affinitäten", fontSize)
	rule := line
	rule.Top = rule.Bottom - 1
	fillPanelRect(pixels, width, height, rule, 0xff6f563a)
	top := line.Bottom
	keys := make([]string, 0, len(enemy.Affinities))
	for school := range enemy.Affinities {
		keys = append(keys, string(school))
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		line.Top = top
		line.Bottom = top + fontSize*2
		drawPanelLine(hdc, pixels, width, height, line, "Keine Affinitäten hinterlegt", fontSize)
	}
	columns := max(1, min(3, int(bounds.Width()/(fontSize*7))))
	cellWidth := bounds.Width() / int32(columns)
	rowHeight := max(int32(28), fontSize*2)
	for i, key := range keys {
		cell := widget.Rect{Left: bounds.Left + int32(i%columns)*cellWidth, Top: top + int32(i/columns)*rowHeight, Right: bounds.Left + int32(i%columns+1)*cellWidth, Bottom: top + int32(i/columns+1)*rowHeight}
		if cell.Bottom > bounds.Bottom {
			break
		}
		school := enemies.School(key)
		drawPanelAffinities(hdc, pixels, width, height, cell, map[enemies.School]int{school: enemy.Affinities[school]}, fontSize)
	}
}
