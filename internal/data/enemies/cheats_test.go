package enemies

import (
	"context"
	"testing"
)

func TestStrategyEnemyReferenceUsesMatchingModeAndKeepsSchoolOrder(t *testing.T) {
	r := NewRepository("de", NewRemoteLoader(testFiles{
		"core/enemies/Ahmarra.yaml":   "boss: {}\nhard-boss: {variant_of: boss, mode: challenge}\nlife: {schools: [storm, life]}\nhard-life: {variant_of: life, mode: challenge, schools: [life, storm]}\n",
		"core/cheats/Ahmarra.yaml":    "hard-boss: {ref: boss}\nboss:\n  mechanics:\n    - id: approach\n      enemies:\n        life_judge: {enemy: life}\n  strategy:\n    - id: approach\n      enemies:\n        life_judge: {enemy: life, label: life_label}\n",
		"i18n/de/cheats/Ahmarra.json": `{"boss":{"approach":"Falle auf {life_judge}.","life_label":"Lebensrichter"}}`,
	}))
	for _, test := range []struct {
		id, target string
		first      School
	}{{"boss", "life", SchoolStorm}, {"hard-boss", "hard-life", SchoolLife}} {
		info, _, err := r.Resolve(context.Background(), "Ahmarra/Room", test.id)
		if err != nil {
			t.Fatal(err)
		}
		ref := info.Strategy[0].Enemies[0]
		mechanicRef := info.Cheats[0].Enemies[0]
		if mechanicRef.EnemyID != test.target || len(mechanicRef.Schools) != 2 || mechanicRef.Schools[0] != test.first {
			t.Fatalf("mechanic reference: %+v", mechanicRef)
		}
		info.Cheats[0].Enemies[0].Schools[0] = SchoolFire
		if ref.EnemyID != test.target || ref.FamilyID != "life" || ref.Label != "Lebensrichter" || len(ref.Schools) != 2 || ref.Schools[0] != test.first {
			t.Fatalf("reference %s: %+v", test.id, ref)
		}
		ref.Schools[0] = SchoolFire
		info, _, _ = r.Resolve(context.Background(), "Ahmarra/Room", test.id)
		if info.Strategy[0].Enemies[0].Schools[0] != test.first {
			t.Fatal("reference schools mutated cache")
		}
		if info.Cheats[0].Enemies[0].Schools[0] != test.first {
			t.Fatal("mechanic reference mutated cache")
		}
	}
}

func TestCheatReferenceSharesDefinitionAndTranslation(t *testing.T) {
	r := NewRepository("de", NewRemoteLoader(testFiles{
		"core/enemies/Ahmarra.yaml":   "normal: {}\nhard:\n  variant_of: normal\n  mode: challenge\n",
		"core/cheats/Ahmarra.yaml":    "hard:\n  ref: normal\nnormal:\n  mechanics:\n    - id: objection\n      notes: [permanent]\n  strategy: [approach]\n",
		"i18n/de/cheats/Ahmarra.json": `{"normal":{"objection":"Einwand","permanent":"Bleibt überstimmt.","approach":"Mythosfalle wirken."}}`,
	}))
	for _, id := range []string{"normal", "hard"} {
		info, found, err := r.Resolve(context.Background(), "Ahmarra/Room", id)
		if err != nil || !found || len(info.Cheats) != 1 || info.Cheats[0].Text != "Einwand" || info.Cheats[0].Notes[0].Text != "Bleibt überstimmt." || len(info.Strategy) != 1 || info.Strategy[0].Text != "Mythosfalle wirken." {
			t.Fatalf("%s reference: %+v %v", id, info, err)
		}
		info.Cheats[0].Notes[0].Text = "changed"
		info.Strategy[0].Text = "changed"
	}
}

func TestInvalidCheatReferencesRejected(t *testing.T) {
	for _, raw := range []string{
		"a: {ref: missing}", "a: {ref: a}", "a: {ref: b}\nb: {ref: c}\nc: {mechanics: []}", "a: {ref: b, strategy: [x]}\nb: {mechanics: []}",
	} {
		r := NewRepository("de", NewRemoteLoader(testFiles{"core/cheats/Ahmarra.yaml": raw}))
		if _, err := r.loadCheats(context.Background(), "Ahmarra"); err == nil {
			t.Fatalf("invalid reference accepted: %s", raw)
		}
	}
}

func TestStrategyOnlyTranslationsInheritanceAndIsolation(t *testing.T) {
	r := NewRepository("de", NewRemoteLoader(testFiles{
		"core/enemies/Ahmarra.yaml":   "main: {}\nalias:\n  ref: main\nplain: {}\n",
		"core/cheats/Ahmarra.yaml":    "main:\n  strategy: [preparation, approach]\n",
		"i18n/de/cheats/Ahmarra.json": `{"main":{"preparation":"Nutze Auren."}}`,
	}))
	for _, id := range []string{"main", "alias"} {
		info, found, err := r.Resolve(context.Background(), "Ahmarra/Room", id)
		if err != nil || !found || !info.HasCheatSheet() || len(info.Cheats) != 0 || len(info.Strategy) != 2 || info.Strategy[0].Text != "Nutze Auren." || info.Strategy[1].Text != "approach" {
			t.Fatalf("strategy %s: %+v %v", id, info, err)
		}
		info.Strategy[0].Text = "changed"
	}
	entries, _ := r.CachedWorld("Ahmarra")
	for _, e := range entries {
		if e.Key == "main" {
			if len(e.Strategy) != 2 {
				t.Fatal("inherited strategy duplicated")
			}
			e.Strategy[0].Text = "changed"
		}
	}
	info, _, _ := r.Resolve(context.Background(), "Ahmarra/Room", "main")
	if info.Strategy[0].Text != "Nutze Auren." {
		t.Fatal("strategy cache mutated")
	}
	info, _, _ = r.Resolve(context.Background(), "Ahmarra/Room", "plain")
	if info.HasCheatSheet() {
		t.Fatal("strategy assigned to unrelated enemy")
	}
}

func TestCheatScheduleTranslationsDefaultsAndIsolation(t *testing.T) {
	r := NewRepository("de", NewRemoteLoader(testFiles{
		"core/enemies/Darkmoor.yaml":   "boss: {}\n",
		"core/cheats/Darkmoor.yaml":    "boss:\n  mechanics:\n    - id: cycle\n      schedule:\n        repeat: 3\n        events:\n          - round: 1\n            text: forced\n          - round: 3\n            timing: end\n            text: cleanup\n",
		"i18n/de/cheats/Darkmoor.json": `{"boss":{"cycle":"Zyklus","forced":"Zwangszauber","cleanup":"Entfernung"}}`,
	}))
	info, _, err := r.Resolve(context.Background(), "Darkmoor/Room", "boss")
	if err != nil {
		t.Fatal(err)
	}
	s := info.Cheats[0].Schedule
	if s == nil || s.Repeat != 3 || len(s.Events) != 2 || s.Events[0].Timing != "start" || s.Events[1].Timing != "end" || s.Events[1].ID != "cleanup" || s.Events[1].Text != "Entfernung" {
		t.Fatalf("schedule: %+v", s)
	}
	s.Events[0].Text = "changed"
	s.Repeat = 9
	entries, _ := r.CachedWorld("Darkmoor")
	entries[0].Cheats[0].Schedule.Events[0].Text = "changed again"
	info, _, _ = r.Resolve(context.Background(), "Darkmoor/Room", "boss")
	if info.Cheats[0].Schedule.Repeat != 3 || info.Cheats[0].Schedule.Events[0].Text != "Zwangszauber" {
		t.Fatal("schedule cache mutated")
	}
}

func TestInvalidCheatSchedulesAreRejected(t *testing.T) {
	for _, schedule := range []string{
		"repeat: -1\n        events: [{round: 1, text: x}]",
		"repeat: 3\n        events: [{round: 4, text: x}]",
		"events: [{round: 0, text: x}]",
		"events: [{round: 1, text: x, timing: invalid}]",
		"events: []",
	} {
		r := NewRepository("de", NewRemoteLoader(testFiles{"core/cheats/Darkmoor.yaml": "boss:\n  mechanics:\n    - id: cycle\n      schedule:\n        " + schedule + "\n"}))
		if _, err := r.loadCheats(context.Background(), "Darkmoor"); err == nil {
			t.Fatalf("accepted invalid schedule: %s", schedule)
		}
	}
}

func TestCheatsTranslationsInheritanceAndCacheIsolation(t *testing.T) {
	r := NewRepository("de", NewRemoteLoader(testFiles{
		"core/enemies/Ahmarra.yaml":   "main: {}\nalias:\n  ref: main\nplain: {}\n",
		"core/cheats/Ahmarra.yaml":    "main:\n  mechanics:\n    - id: blade\n      notes: [exception, untranslated]\n    - id: missing\n",
		"i18n/de/cheats/Ahmarra.json": `{"main":{"blade":"Keine Klingen.","exception":"Auren sind erlaubt."}}`,
	}))
	for _, id := range []string{"main", "alias"} {
		info, found, err := r.Resolve(context.Background(), "Ahmarra/Zone", id)
		if err != nil || !found || len(info.Cheats) != 2 || info.Cheats[0].Text != "Keine Klingen." || info.Cheats[1].Text != "missing" {
			t.Fatalf("%s: %+v %v", id, info, err)
		}
		info.Cheats[0].Text = "changed"
		if len(info.Cheats[0].Notes) != 2 || info.Cheats[0].Notes[0].Text != "Auren sind erlaubt." || info.Cheats[0].Notes[1].Text != "untranslated" {
			t.Fatalf("notes not resolved in order: %+v", info.Cheats)
		}
		info.Cheats[0].Notes[0].Text = "changed"
	}
	entries, _ := r.ListWorld(context.Background(), "Ahmarra")
	for _, e := range entries {
		if e.Key == "main" && len(e.Cheats) != 2 {
			t.Fatal("inherited notes duplicated")
		}
		if e.Key == "plain" && len(e.Cheats) != 0 {
			t.Fatal("cheats assigned to unrelated enemy")
		}
		if len(e.Cheats) > 0 {
			e.Cheats[0].Text = "changed"
			e.Cheats[0].Notes[0].Text = "changed"
		}
	}
	info, _, _ := r.Resolve(context.Background(), "Ahmarra/Zone", "main")
	if info.Cheats[0].Text != "Keine Klingen." || info.Cheats[0].Notes[0].Text != "Auren sind erlaubt." {
		t.Fatal("cache mutated")
	}
}
