//go:build windows

package windows

import (
	"corvin101/internal/data/enemies"
	"corvin101/internal/game/events"
	"corvin101/internal/game/state"
	"corvin101/internal/overlay/widget"
	"reflect"
	"testing"
)

func TestCombatEnemyOrderingAndEnd(t *testing.T) {
	source := []enemies.WorldEnemy{
		{Key: "z", EnemyInfo: enemies.EnemyInfo{Name: "Zulu"}},
		{Key: "b", EnemyInfo: enemies.EnemyInfo{Name: "Beta"}},
		{Key: "a", EnemyInfo: enemies.EnemyInfo{Name: "Alpha"}},
	}
	snapshot := state.Snapshot{Game: state.GameSnapshot{ZoneKey: "Avalon/Hub"}, Combat: state.CombatSnapshot{Active: true, Participants: []state.ParticipantSnapshot{
		{Kind: events.ParticipantLocalPlayer, Team: 1},
		{Kind: events.ParticipantMob, Team: 2, MobID: "z"},
		{Kind: events.ParticipantMob, Team: 2, MobID: "b"},
		{Kind: events.ParticipantMob, Team: 2, MobID: "unregistered"},
		{Kind: events.ParticipantMob, Team: 1, MobID: "a"},
	}}}
	w := &Window{panelWorld: "Avalon", panelPage: 3, selectedEnemy: "a"}
	w.updatePanelCombat(snapshot)
	keys := func(entries []enemies.WorldEnemy) []string {
		result := []string{}
		for _, e := range entries {
			result = append(result, e.Key)
		}
		return result
	}
	if got := keys(orderedPanelEnemies(source, w.panelCombatIDs)); !reflect.DeepEqual(got, []string{"b", "z", "a"}) {
		t.Fatalf("battle order: %v", got)
	}
	if w.panelPage != 0 || w.selectedEnemy != "a" {
		t.Fatal("priority update lost selection or did not reset page")
	}
	w.panelPage = 1
	snapshot.Combat.Phase = events.PhasePlanning
	w.updatePanelCombat(snapshot)
	if w.panelPage != 1 {
		t.Fatal("unchanged combat opponents reset page")
	}
	if len(combatEnemyIDs(snapshot, "Novus")) != 0 {
		t.Fatal("different world received combat priority")
	}
	snapshot.Combat.Active = false
	w.updatePanelCombat(snapshot)
	if got := keys(orderedPanelEnemies(source, w.panelCombatIDs)); !reflect.DeepEqual(got, []string{"a", "b", "z"}) {
		t.Fatalf("post-battle order: %v", got)
	}
	if source[0].Key != "z" {
		t.Fatal("sorting mutated repository list")
	}
}

func TestEnemyClickUsesDisplayedOrder(t *testing.T) {
	w := &Window{panelOpen: true, width: 1280, height: 720, panelEnemies: []enemies.WorldEnemy{
		{Key: "a", EnemyInfo: enemies.EnemyInfo{Name: "Alpha"}},
		{Key: "z", EnemyInfo: enemies.EnemyInfo{Name: "Zulu"}},
	}, panelCombatIDs: map[string]bool{"z": true}}
	list, details := enemyPanelColumns(w.width, w.height)
	row := enemyPanelRow(list, w.height, 0)
	w.clickPanel(widget.Point{X: row.Left + 10, Y: row.Top + 10})
	if w.selectedEnemy != "z" {
		t.Fatalf("wrong selection: %s", w.selectedEnemy)
	}
	w.clickPanel(widget.Point{X: details.Left + 10, Y: details.Top + 10})
	if w.selectedEnemy != "z" {
		t.Fatal("detail click changed selection")
	}
}

func TestBattleThenLocationThenRestAlphabetically(t *testing.T) {
	zone := 202
	source := []enemies.WorldEnemy{
		{Key: "rest-z", EnemyInfo: enemies.EnemyInfo{Name: "Zulu"}},
		{Key: "local-z", EnemyInfo: enemies.EnemyInfo{Name: "Zulu", Locations: []int{202}}},
		{Key: "battle-z", EnemyInfo: enemies.EnemyInfo{Name: "Zulu", Locations: []int{202}}},
		{Key: "rest-a", EnemyInfo: enemies.EnemyInfo{Name: "Alpha"}},
		{Key: "local-a", EnemyInfo: enemies.EnemyInfo{Name: "Alpha", Locations: []int{201, 202}}},
		{Key: "battle-a", EnemyInfo: enemies.EnemyInfo{Name: "Alpha", GameIDs: []string{"alias"}, Locations: []int{999}}},
	}
	ordered := orderedPanelEnemies(source, map[string]bool{"battle-z": true, "alias": true}, &zone)
	want := []string{"battle-a", "battle-z", "local-a", "local-z", "rest-a", "rest-z"}
	for i, key := range want {
		if ordered[i].Key != key {
			t.Fatalf("position %d: got %s, want %s", i, ordered[i].Key, key)
		}
	}
	if enemyInLocation(source[0], &zone) || enemyInLocation(source[1], nil) {
		t.Fatal("unknown location was prioritized")
	}
}
