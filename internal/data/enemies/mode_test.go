package enemies

import (
	"context"
	"testing"
)

func TestChallengeVariantsKeepIndependentStatsAndIDs(t *testing.T) {
	r := NewRepository("de", NewRemoteLoader(testFiles{
		"core/enemies/Ahmarra.yaml": "normal:\n  schools: [storm]\n  affinities: {fire: -20}\nhard:\n  variant_of: normal\n  mode: challenge\n  locations: [Ahmarra/Room_Hard]\n",
	}))
	entries, err := r.ListWorld(context.Background(), "Ahmarra")
	if err != nil || len(entries) != 2 {
		t.Fatalf("separate variants: %+v %v", entries, err)
	}
	normal, _, _ := r.Resolve(context.Background(), "Ahmarra/Room", "normal")
	hard, _, _ := r.Resolve(context.Background(), "Ahmarra/Room_Hard", "hard")
	if normal.Mode != "standard" || hard.Mode != "challenge" || hard.VariantOf != "normal" {
		t.Fatal("variant metadata missing")
	}
	if len(hard.Schools) != 0 || len(hard.Affinities) != 0 {
		t.Fatal("challenge inherited unverified standard stats")
	}
	for _, e := range entries {
		if e.Key == "normal" && e.InCombat(map[string]bool{"hard": true}) {
			t.Fatal("challenge ID counted as standard combat enemy")
		}
	}
}

func TestInvalidVariantReferencesRejected(t *testing.T) {
	for _, raw := range []string{"hard: {variant_of: missing, mode: challenge}", "hard: {variant_of: hard, mode: challenge}", "normal: {}\nhard: {variant_of: normal}", "hard: {mode: typo}"} {
		r := NewRepository("de", NewRemoteLoader(testFiles{"core/enemies/Ahmarra.yaml": raw}))
		if _, err := r.ListWorld(context.Background(), "Ahmarra"); err == nil {
			t.Fatalf("accepted invalid variant: %s", raw)
		}
	}
}
