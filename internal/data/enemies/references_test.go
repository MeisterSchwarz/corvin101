package enemies

import (
	"context"
	"reflect"
	"testing"
)

func TestEnemyReferencesMergeCatalogButKeepVariantDrops(t *testing.T) {
	repo := NewRepository("de", NewRemoteLoader(testFiles{
		"core/enemies/Arcanum.yaml":     "main:\n  schools: [fire]\n  locations: [203]\n  affinities:\n    ice: 15\n    fire: -26\nalias:\n  ref: main\n  locations: [202, 203]\n",
		"i18n/de/enemies/Arcanum.json":  `{"main":"Gemeinsamer Gegner","alias":"Anderer Übersetzungstext"}`,
		"core/drops/Arcanum.yaml":       "main:\n  treasurecards:\n    - item: shared\nalias:\n  treasurecards:\n    - item: shared\n    - item: extra\n",
		"core/items/treasurecards.json": `{"shared":{},"extra":{}}`,
	}))
	ctx := context.Background()
	entries, err := repo.ListWorld(ctx, "Arcanum")
	if err != nil || len(entries) != 1 {
		t.Fatalf("catalog: %+v, %v", entries, err)
	}
	entry := entries[0]
	if entry.Key != "main" || entry.Name != "Gemeinsamer Gegner" || !reflect.DeepEqual(entry.Locations, []int{202, 203}) || len(entry.GameIDs) != 2 {
		t.Fatalf("merge: %+v", entry)
	}
	if !entry.InCombat(map[string]bool{"alias": true}) || !entry.InCombat(map[string]bool{"main": true}) || entry.InCombat(map[string]bool{"other": true}) {
		t.Fatal("combat identity matching failed")
	}
	if len(entry.DropCategories) != 1 || len(entry.DropCategories[0].Items) != 2 {
		t.Fatalf("drops not deduplicated: %+v", entry.DropCategories)
	}
	for _, id := range []string{"main", "alias"} {
		info, found, err := repo.Resolve(ctx, "Arcanum/Hub", id)
		if err != nil || !found || info.Name != "Gemeinsamer Gegner" || len(info.Schools) != 1 || info.Schools[0] != SchoolFire || info.Affinities[SchoolIce] != 15 {
			t.Fatalf("resolve %s: %+v %v", id, info, err)
		}
		if id == "main" && len(info.DropCategories[0].Items) != 1 {
			t.Fatal("alias drops leaked into actual main variant")
		}
	}
	entries[0].GameIDs[0] = "changed"
	again, _ := repo.CachedWorld("Arcanum")
	if again[0].GameIDs[0] == "changed" {
		t.Fatal("caller mutated cached identities")
	}
}

func TestInvalidEnemyReferences(t *testing.T) {
	for name, core := range map[string]coreEnemyFile{
		"missing":  {"a": {Ref: "missing"}},
		"self":     {"a": {Ref: "a"}},
		"chain":    {"a": {}, "b": {Ref: "a"}, "c": {Ref: "b"}},
		"cycle":    {"a": {Ref: "b"}, "b": {Ref: "a"}},
		"override": {"a": {}, "b": {Ref: "a", Schools: []School{SchoolFire}}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateEnemyReferences(core); err == nil {
				t.Fatal("invalid reference accepted")
			}
		})
	}
}
