//go:build windows

package windows

import (
	"corvin101/internal/data/enemies"
	"corvin101/internal/overlay/widget"
	"testing"
)

func TestSearchFiltersNamesOnlyAndKeepsPriority(t *testing.T) {
	zone := 202
	entries := []enemies.WorldEnemy{
		{Key: "rest", EnemyInfo: enemies.EnemyInfo{Name: "Krok Alpha"}},
		{Key: "local", EnemyInfo: enemies.EnemyInfo{Name: "Krok Zulu", Locations: []int{202}}},
		{Key: "boss", EnemyInfo: enemies.EnemyInfo{Name: "Krok Boss", GameIDs: []string{"boss_alias"}}},
	}
	filtered := filterPanelEnemies(orderedPanelEnemies(entries, map[string]bool{"boss_alias": true}, &zone), "KROK")
	if len(filtered) != 3 || filtered[0].Key != "boss" || filtered[1].Key != "local" || filtered[2].Key != "rest" {
		t.Fatalf("priority: %+v", filtered)
	}
	if got := filterPanelEnemies(entries, " BOSS_ALIAS "); len(got) != 0 {
		t.Fatalf("alias ID matched: %+v", got)
	}
	if got := filterPanelEnemies(entries, "rest"); len(got) != 0 {
		t.Fatalf("game ID matched: %+v", got)
	}
	if got := filterPanelEnemies(entries, "missing"); len(got) != 0 {
		t.Fatal("unexpected matches")
	}
}

func TestSearchRequiresActivationAndResetsPagination(t *testing.T) {
	w := &Window{panelOpen: true, width: 1280, height: 720, panelPage: 5, panelEnemies: []enemies.WorldEnemy{
		{Key: "a", EnemyInfo: enemies.EnemyInfo{Name: "Alpha"}},
		{Key: "b", EnemyInfo: enemies.EnemyInfo{Name: "Beta"}},
	}}
	w.searchCharacter('A')
	if w.searchQuery != "" {
		t.Fatal("inactive search consumed typing")
	}
	bounds := searchBounds(w.width, w.height)
	if !w.clickSearch(widget.Point{X: bounds.Left + 5, Y: bounds.Top + 5}) {
		t.Fatal("search click not recognized")
	}
	w.searchCharacter('B')
	if w.searchQuery != "B" || w.panelPage != 0 || len(w.panelEntriesLocked()) != 1 {
		t.Fatal("query did not reset/filter")
	}
	w.searchCharacter(8)
	if w.searchQuery != "" || len(w.panelEntriesLocked()) != 2 {
		t.Fatal("backspace failed")
	}
	w.searchCharacter('a')
	w.searchCharacter(13)
	if w.searchActive || w.searchQuery != "a" {
		t.Fatal("enter did not release focus")
	}
}

func TestSearchAndListDoNotOverlap(t *testing.T) {
	for _, height := range []int32{600, 720, 1080, 1440} {
		list, _ := enemyPanelColumns(1920, height)
		search := searchBounds(1920, height)
		if search.Bottom >= list.Top {
			t.Fatal("search overlaps first row")
		}
		last := enemyPanelRow(list, height, panelRows(height)-1)
		if last.Bottom > list.Bottom {
			t.Fatalf("last row overflows at %d: %+v %+v", height, last, list)
		}
	}
}
