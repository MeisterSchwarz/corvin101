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
	list.Right = bounds.Left + bounds.Width()*30/100
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
	w.mu.RLock()
	dropsView, cursor := w.detailDrops, w.cursor
	w.mu.RUnlock()
	title := line
	title.Right = detailToggleRect(width, height, dropsView).Left - 12
	drawEnemyDetailTitle(hdc, pixels, width, height, title, enemy.Name, enemy.Schools, fontSize+6)
	debugSize := max(int32(10), fontSize-4)
	debugLine := bounds
	debugLine.Top = bounds.Bottom - debugSize*2
	debugLine.Left = max(bounds.Left, bounds.Right-panelTextWidth(hdc, enemy.Key, debugSize))
	drawPanelLine(hdc, pixels, width, height, debugLine, enemy.Key, debugSize)
	// Reserve the bottom corner for the temporary debug identifier.
	bounds.Bottom = debugLine.Top - 6
	line.Top = line.Bottom
	line.Bottom += fontSize * 2
	if len(enemy.LocationNames) > 0 {
		location := line
		location.Right = title.Right
		drawPanelTextColor(hdc, pixels, width, height, location, strings.Join(enemy.LocationNames, ", "), max(14, fontSize), theme.UIFontFamily, theme.RGB{R: 210, G: 207, B: 201})
	}
	if dropsView {
		drawDetailToggle(hdc, pixels, width, height, true, cursor)
		drawEnemyDrops(hdc, pixels, width, height, bounds, line.Bottom+fontSize, fontSize, enemy.EnemyInfo)
		return
	}
	line.Top = line.Bottom + fontSize
	line.Bottom = line.Top + fontSize*2
	drawPanelText(hdc, pixels, width, height, line, "Schaden (+/-)", fontSize+2, theme.DisplayFontFamily)
	rule := line
	rule.Top = rule.Bottom - 1
	fillPanelRect(pixels, width, height, rule, 0xff6f563a)
	drawDetailToggle(hdc, pixels, width, height, false, cursor)
	drawDamageColumns(hdc, pixels, width, height, bounds, line.Bottom+6, fontSize, enemy.Affinities)
}

func drawEnemyDetailTitle(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, name string, schools []enemies.School, fontSize int32) {
	iconSize := min(bounds.Height()-2, fontSize+8)
	gap := int32(5)
	text := bounds
	if len(schools) > 0 {
		text.Right = max(text.Left, bounds.Right-int32(len(schools))*(iconSize+gap)-8)
	}
	drawBestiaryText(hdc, pixels, width, height, text, name, fontSize, true)
	left := min(text.Right, bounds.Left+panelFontTextWidth(hdc, name, fontSize, theme.DisplayFontFamily)) + 8
	for _, school := range schools {
		if left+iconSize > bounds.Right {
			break
		}
		drawSchoolIcon(pixels, width, height, school, left+iconSize/2, bounds.Top+(bounds.Height()-iconSize)/2, iconSize)
		left += iconSize + gap
	}
}
