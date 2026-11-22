//go:build windows

package windows

import (
	"corvin101/internal/data/enemies"
	"corvin101/internal/overlay/widget"
	"testing"
)

func TestModeSwitchPreservesCurrentDetailView(t *testing.T) {
	for _, view := range []string{"stats", "drops", "cheats"} {
		t.Run(view, func(t *testing.T) {
			w := &Window{panelOpen: true, width: 1280, height: 720, selectedEnemy: "normal", detailDrops: view == "drops", detailCheats: view == "cheats", dropCategory: "reagents", panelEnemies: []enemies.WorldEnemy{
				{Key: "normal", EnemyInfo: enemies.EnemyInfo{Cheats: []enemies.CheatMechanic{{ID: "cheat"}}}},
				{Key: "hard", EnemyInfo: enemies.EnemyInfo{Mode: "challenge", VariantOf: "normal"}},
			}}
			for _, mode := range []string{"challenge", "standard"} {
				w.setPanelModeLocked(mode)
				if w.detailDrops != (view == "drops") || w.detailCheats != (view == "cheats") || w.dropCategory != "reagents" || w.selectedEnemy != "normal" {
					t.Fatal("mode switch reset detail view")
				}
				if !w.hasSelectedCheatsLocked() {
					t.Fatal("sheet became inaccessible on variant without data")
				}
			}
		})
	}
}

func TestChallengeIconClickTogglesOnlySelectedEnemy(t *testing.T) {
	w := &Window{panelOpen: true, width: 1280, height: 720, selectedEnemy: "normal", panelEnemies: []enemies.WorldEnemy{
		{Key: "normal"}, {Key: "hard", EnemyInfo: enemies.EnemyInfo{Mode: "challenge", VariantOf: "normal"}}, {Key: "plain"},
	}}
	r := modeButtonRect(w.width, w.height, false)
	p := widget.Point{X: r.Left + 5, Y: r.Top + 5}
	if w.modeButtonAtLocked(p) != 1 {
		t.Fatal("standard icon does not target challenge")
	}
	w.clickPanel(p)
	if w.enemyModes["normal"] != "challenge" || w.modeButtonAtLocked(p) != 0 {
		t.Fatal("challenge icon did not toggle")
	}
	w.clickPanel(p)
	if w.enemyModes["normal"] != "standard" {
		t.Fatal("gray icon did not return")
	}
	w.selectedEnemy = "plain"
	if w.modeButtonAtLocked(p) != -1 {
		t.Fatal("icon shown without challenge variant")
	}
}

func TestChallengeIconGrayVariantPreservesAlpha(t *testing.T) {
	red, gray := challengeModeIcon(true), challengeModeIcon(false)
	if red == nil || gray == nil || len(red.pixels) != len(gray.pixels) {
		t.Fatal("embedded icons unavailable")
	}
	changed := false
	for i, p := range red.pixels {
		g := gray.pixels[i]
		if p>>24 != g>>24 {
			t.Fatal("gray icon changed transparency")
		}
		if (g>>16)&255 != (g>>8)&255 || (g>>8)&255 != g&255 {
			t.Fatal("inactive icon is not neutral gray")
		}
		if p != g {
			changed = true
		}
	}
	if !changed {
		t.Fatal("red and gray variants identical")
	}
}

func TestPerEnemyModesKeepOneRowAndIndependentChoices(t *testing.T) {
	w := &Window{height: 720, panelWorld: "Ahmarra", panelZone: "Ahmarra/Room", selectedEnemy: "normal", panelCombatIDs: map[string]bool{"hard": true}, panelEnemies: []enemies.WorldEnemy{
		{Key: "normal", EnemyInfo: enemies.EnemyInfo{Name: "Boss"}},
		{Key: "hard", EnemyInfo: enemies.EnemyInfo{Name: "Boss", Mode: "challenge", VariantOf: "normal", Locations: []string{"Ahmarra/Room_Hard"}}},
		{Key: "other", EnemyInfo: enemies.EnemyInfo{Name: "Other"}},
		{Key: "other-hard", EnemyInfo: enemies.EnemyInfo{Name: "Other", Mode: "challenge", VariantOf: "other"}},
		{Key: "standard-only", EnemyInfo: enemies.EnemyInfo{Name: "Plain"}},
	}}
	find := func(key string) enemies.WorldEnemy {
		for _, e := range w.modeEntriesLocked() {
			if e.Key == key {
				return e
			}
		}
		t.Fatal("missing family", key)
		return enemies.WorldEnemy{}
	}
	if len(w.panelEntriesLocked()) != 3 || !w.hasChallengeLocked() {
		t.Fatal("duplicate catalog rows or missing switch")
	}
	if find("normal").VariantKey != "hard" {
		t.Fatal("combat challenge variant not automatic")
	}
	if find("other").VariantKey != "other" {
		t.Fatal("unrelated enemy switched mode")
	}
	w.setPanelModeLocked("standard")
	if find("normal").VariantKey != "normal" || !find("normal").InCombat(w.panelCombatIDs) {
		t.Fatal("manual standard failed or lost battle highlight")
	}
	w.selectedEnemy = "other"
	w.setPanelModeLocked("challenge")
	if find("other").VariantKey != "other-hard" || find("normal").VariantKey != "normal" {
		t.Fatal("per-enemy selections are not independent")
	}
	w.selectedEnemy = "normal"
	if find("normal").VariantKey != "normal" {
		t.Fatal("choice not retained when returning to enemy")
	}
	w.autoPanelModeLocked()
	if find("normal").VariantKey != "hard" {
		t.Fatal("reopening did not restore current combat variant")
	}
	w.selectedEnemy = "standard-only"
	if w.hasChallengeLocked() {
		t.Fatal("switch shown without counterpart")
	}
}
