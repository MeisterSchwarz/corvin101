package enemies

import (
	"context"
	"testing"
)

func TestEncounterModeVariantsAndReferencesStaySeparate(t *testing.T) {
	r := NewRepository("de", NewRemoteLoader(testFiles{
		"core/enemies/Ahmarra.yaml":    "one: {schools: [storm, death]}\ntwo: {schools: [storm, life]}\nhard-one: {mode: challenge, variant_of: one, schools: [storm, death]}\nhard-two: {mode: challenge, variant_of: two, schools: [storm, life]}\n",
		"core/encounters/Ahmarra.yaml": "judges: {enemies: [one, two]}\nhard-judges: {mode: challenge, variant_of: judges, enemies: [hard-one, hard-two]}\n",
		"core/cheats/Ahmarra.yaml":     "judges:\n  strategy:\n    - id: approach\n      enemies:\n        target: {enemy: two}\nhard-judges: {ref: judges}\n",
	}))
	entries, err := r.ListWorld(context.Background(), "Ahmarra")
	if err != nil || len(entries) != 2 {
		t.Fatalf("groups: %+v %v", entries, err)
	}
	for _, entry := range entries {
		if len(entry.Heads) != 2 || len(entry.Strategy) != 1 {
			t.Fatal("members/shared strategy missing")
		}
		ref := entry.Strategy[0].Enemies[0]
		if ref.FamilyID != "judges" || ref.HeadID != "two" {
			t.Fatalf("reference no longer targets grouped head: %+v", ref)
		}
		if entry.Mode == "challenge" {
			if entry.VariantOf != "judges" || ref.EnemyID != "hard-two" || entry.Heads[1].Key != "hard-two" {
				t.Fatal("challenge group mismatch")
			}
		} else if ref.EnemyID != "two" {
			t.Fatal("standard group mismatch")
		}
	}
}

func TestEncounterGroupsCatalogButKeepsIndependentCombatIdentities(t *testing.T) {
	r := NewRepository("de", NewRemoteLoader(testFiles{
		"core/enemies/Darkmoor.yaml":       "one: {schools: [balance], affinities: {fire: -10}}\ntwo: {schools: [life, myth], affinities: {fire: -20}}\nother: {}\n",
		"core/encounters/Darkmoor.yaml":    "balaur:\n  enemies: [one, two]\n",
		"i18n/de/encounters/Darkmoor.json": `{"balaur":"Balaur"}`,
		"core/cheats/Darkmoor.yaml":        "balaur:\n  strategy: [order]\n",
		"i18n/de/cheats/Darkmoor.json":     `{"balaur":{"order":"Balance zuerst."}}`,
		"core/drops/Darkmoor.yaml":         "balaur:\n  gold: [10, 15]\n",
	}))
	entries, err := r.ListWorld(context.Background(), "Darkmoor")
	if err != nil || len(entries) != 2 {
		t.Fatalf("catalog: %+v %v", entries, err)
	}
	var group WorldEnemy
	for _, e := range entries {
		if e.Key == "balaur" {
			group = e
		}
	}
	if group.Name != "Balaur" || len(group.Heads) != 2 || group.Heads[0].Key != "one" || group.Heads[1].Key != "two" || len(group.Strategy) != 1 || group.Strategy[0].Text != "Balance zuerst." || len(group.DropCategories) != 1 {
		t.Fatalf("group: %+v", group)
	}
	if !group.InCombat(map[string]bool{"two": true}) {
		t.Fatal("head does not activate encounter")
	}
	if group.Heads[0].Affinities[SchoolFire] != -10 || group.Heads[1].Affinities[SchoolFire] != -20 || len(group.Affinities) != 0 {
		t.Fatal("head stats merged")
	}
	group.Heads[0].Affinities[SchoolFire] = 99
	group.Heads[0].Schools[0] = SchoolDeath
	again, _ := r.CachedWorld("Darkmoor")
	for _, e := range again {
		if e.Key == "balaur" && (e.Heads[0].Affinities[SchoolFire] != -10 || e.Heads[0].Schools[0] != SchoolBalance) {
			t.Fatal("head cache mutated")
		}
	}
	info, found, err := r.Resolve(context.Background(), "Darkmoor/Room", "two")
	if err != nil || !found || info.Affinities[SchoolFire] != -20 || len(info.Strategy) != 1 || len(info.DropCategories) != 1 || len(info.GameIDs) != 1 || info.GameIDs[0] != "two" {
		t.Fatalf("individual combat enemy lost: %+v %v", info, err)
	}
}

func TestEncounterRejectsMissingOrRepeatedMembers(t *testing.T) {
	for _, def := range []string{"x: {enemies: [missing]}", "x: {enemies: [one, one]}", "x: {enemies: [one]}\ny: {enemies: [one]}"} {
		r := NewRepository("de", NewRemoteLoader(testFiles{"core/enemies/Darkmoor.yaml": "one: {}", "core/encounters/Darkmoor.yaml": def}))
		if _, err := r.ListWorld(context.Background(), "Darkmoor"); err == nil {
			t.Fatalf("invalid encounter accepted: %s", def)
		}
	}
}
