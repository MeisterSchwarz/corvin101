//go:build windows

package windows

import (
	"corvin101/internal/data/enemies"
	"corvin101/internal/overlay/widget"
	"testing"
)

func TestEncounterModeSwitchPreservesMemberAndView(t *testing.T) {
	w := &Window{panelOpen: true, width: 1280, height: 720, selectedEnemy: "judges", detailCheats: true, panelEnemies: []enemies.WorldEnemy{
		{Key: "judges", EnemyInfo: enemies.EnemyInfo{Heads: []enemies.EncounterHead{{Key: "death"}, {Key: "life"}}}},
		{Key: "hard-judges", EnemyInfo: enemies.EnemyInfo{Mode: "challenge", VariantOf: "judges", Heads: []enemies.EncounterHead{{Key: "hard-death", EnemyInfo: enemies.EnemyInfo{VariantOf: "death"}}, {Key: "hard-life", EnemyInfo: enemies.EnemyInfo{VariantOf: "life"}}}}},
	}, encounterHeads: map[string]string{"judges": "life"}}
	for _, mode := range []string{"challenge", "standard"} {
		w.setPanelModeLocked(mode)
		entries := w.modeEntriesLocked()
		if len(entries) != 1 || !w.detailCheats {
			t.Fatal("group switch changed catalog or view")
		}
		head := w.selectedHeadLocked(entries[0])
		want := "life"
		if mode == "challenge" {
			want = "hard-life"
		}
		if head == nil || head.Key != want {
			t.Fatalf("member lost: %+v", head)
		}
	}
}

func TestEncounterSchoolButtonsSelectIndependentHeads(t *testing.T) {
	group := enemies.WorldEnemy{Key: "balaur", EnemyInfo: enemies.EnemyInfo{Heads: []enemies.EncounterHead{
		{Key: "balance", EnemyInfo: enemies.EnemyInfo{Schools: []enemies.School{enemies.SchoolBalance}}},
		{Key: "life-myth", EnemyInfo: enemies.EnemyInfo{Schools: []enemies.School{enemies.SchoolLife, enemies.SchoolMyth}}},
	}}}
	w := &Window{panelOpen: true, width: 1280, height: 720, selectedEnemy: "balaur", panelEnemies: []enemies.WorldEnemy{group}, panelCombatIDs: map[string]bool{"life-myth": true}}
	if w.selectedHeadLocked(group).Key != "life-myth" {
		t.Fatal("active head not selected")
	}
	rects := headButtonRects(w.width, w.height, group.Heads)
	if rects[0].Right >= rects[1].Left {
		t.Fatal("head buttons overlap separator")
	}
	w.clickPanel(widget.Point{X: rects[0].Left + 2, Y: rects[0].Top + 2})
	if w.selectedHeadLocked(group).Key != "balance" || w.selectedEnemy != "balaur" {
		t.Fatal("head click did not keep encounter selected")
	}
	w.detailDrops = true
	if w.headButtonAtLocked(widget.Point{X: rects[0].Left + 2, Y: rects[0].Top + 2}) != "" {
		t.Fatal("head selector active on shared drops page")
	}
}
