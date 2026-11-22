//go:build windows

package windows

import (
	"corvin101/assets"
	"corvin101/internal/data/enemies"
	"corvin101/internal/overlay/widget"
	"testing"
)

func TestResistanceIconCentersVisibleContent(t *testing.T) {
	f, err := assets.Items.Open("items/resistance.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	icon, err := decodeIcon(f)
	if err != nil {
		t.Fatal(err)
	}
	icon = normalizeToggleIcon(icon)
	left, top, right, bottom := icon.width, icon.height, 0, 0
	for y := 0; y < icon.height; y++ {
		for x := 0; x < icon.width; x++ {
			if icon.pixels[y*icon.width+x]>>24 > 8 {
				left = min(left, x)
				right = max(right, x+1)
				top = min(top, y)
				bottom = max(bottom, y+1)
			}
		}
	}
	dx, dy := left-(icon.width-right), top-(icon.height-bottom)
	if dx*dx > 1 || dy*dy > 1 {
		t.Fatalf("visible silhouette off center: %d,%d - %d,%d on %dx%d", left, top, right, bottom, icon.width, icon.height)
	}
}

func TestCheatSheetScrollDoesNotChangeEnemyPage(t *testing.T) {
	w := &Window{panelOpen: true, detailCheats: true, width: 1280, height: 720, cheatScrollMax: 200, panelPage: 2}
	_, details := enemyPanelColumns(w.width, w.height)
	p := widget.Point{X: details.Left + 20, Y: details.Top + 100}
	if !w.wheelPanel(p, -120) || w.cheatScroll <= 0 || w.panelPage != 2 {
		t.Fatal("cheat scroll changed list or failed")
	}
	w.wheelPanel(p, -12000)
	if w.cheatScroll != 200 {
		t.Fatal("scroll exceeded content")
	}
	w.wheelPanel(p, 12000)
	if w.cheatScroll != 0 {
		t.Fatal("scroll exceeded start")
	}
}

func TestCheatButtonOnlyForSelectedEnemyWithNotes(t *testing.T) {
	w := &Window{panelOpen: true, width: 1280, height: 720, selectedEnemy: "plain", panelEnemies: []enemies.WorldEnemy{{Key: "plain"}, {Key: "boss", EnemyInfo: enemies.EnemyInfo{Cheats: []enemies.CheatMechanic{{ID: "blade", Text: "Keine Klingen."}}}}}}
	if w.hasSelectedCheatsLocked() {
		t.Fatal("button shown without cheats")
	}
	w.selectedEnemy = "boss"
	if !w.hasSelectedCheatsLocked() {
		t.Fatal("cheat button missing")
	}
	r := cheatButtonRect(w.width, w.height)
	w.clickPanel(widget.Point{X: r.Left + 2, Y: r.Top + 2})
	if !w.detailCheats || w.detailDrops {
		t.Fatal("cheats did not open")
	}
	w.clickPanel(widget.Point{X: r.Left + 2, Y: r.Top + 2})
	if !w.detailCheats {
		t.Fatal("active cheat tab unexpectedly toggled")
	}
	r = detailTabRect(w.width, w.height, 0, true)
	w.clickPanel(widget.Point{X: r.Left + 2, Y: r.Top + 2})
	if w.detailCheats || w.detailDrops {
		t.Fatal("stats tab did not open")
	}
	r = detailTabRect(w.width, w.height, 1, true)
	w.clickPanel(widget.Point{X: r.Left + 2, Y: r.Top + 2})
	if !w.detailDrops || w.detailCheats {
		t.Fatal("drops tab did not open")
	}
}

func TestStrategyOnlyEnemyShowsSheetButton(t *testing.T) {
	w := &Window{panelOpen: true, width: 1280, height: 720, selectedEnemy: "boss", panelEnemies: []enemies.WorldEnemy{{Key: "boss", EnemyInfo: enemies.EnemyInfo{Strategy: []enemies.CheatNote{{ID: "strategy", Text: "Hinweis"}}}}}}
	if !w.hasSelectedCheatsLocked() {
		t.Fatal("strategy-only sheet is inaccessible")
	}
}
