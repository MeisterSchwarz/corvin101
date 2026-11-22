package drops

import (
	"context"
	"io/fs"
	"testing"
)

type testDropFiles map[string]string

func (f testDropFiles) ReadFile(_ context.Context, path string) ([]byte, error) {
	if data, ok := f[path]; ok {
		return []byte(data), nil
	}
	return nil, fs.ErrNotExist
}

func TestSimpleItemListsAndGoldRange(t *testing.T) {
	repo := NewRepository(testDropFiles{
		"core/drops/Arcanum.yaml":          "enemy:\n  treasurecards:\n    - tc_storm_trap\n    - item: tc_ice_trap\n    - tc_storm_trap\n  gold: [10, 15]\n",
		"core/items/treasurecards.yaml":    "tc_storm_trap: {}\ntc_ice_trap: {}\n",
		"i18n/de/items/treasurecards.json": `{"tc_storm_trap":"Sturmfalle","tc_ice_trap":"Eisfalle"}`,
	}, "de")
	world, err := repo.LoadWorld(context.Background(), "Arcanum")
	if err != nil {
		t.Fatal(err)
	}
	categories := world["enemy"]
	if len(categories) != 2 || categories[0].Key != "gold" || categories[0].Gold == nil || categories[0].Gold.Min != 10 || categories[0].Gold.Max != 15 {
		t.Fatalf("gold range: %+v", categories)
	}
	if categories[1].Key != "treasurecards" || len(categories[1].Items) != 2 || categories[1].Items[0].Name != "Eisfalle" || categories[1].Items[1].ID != "tc_storm_trap" {
		t.Fatalf("item references: %+v", categories[1])
	}
	copy := Clone(categories)
	copy[0].Gold.Min = 999
	if categories[0].Gold.Min != 10 {
		t.Fatal("gold clone shares mutable range")
	}
}

func TestInvalidGoldRanges(t *testing.T) {
	for _, value := range []string{"[15, 10]", "[-1, 15]", "[10]", "[10, 15, 20]", "[10.5, 15]", "['10', 15]", "{}"} {
		t.Run(value, func(t *testing.T) {
			repo := NewRepository(testDropFiles{"core/drops/World.yaml": "enemy:\n  gold: " + value}, "de")
			if _, err := repo.LoadWorld(context.Background(), "World"); err == nil {
				t.Fatal("invalid range accepted")
			}
		})
	}
}
