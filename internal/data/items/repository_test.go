package items

import (
	"context"
	"io/fs"
	"testing"
)

type testItemFiles map[string]string

func (f testItemFiles) ReadFile(_ context.Context, path string) ([]byte, error) {
	if raw, ok := f[path]; ok {
		return []byte(raw), nil
	}
	return nil, fs.ErrNotExist
}

func TestCategoryLoadsYAMLAndKeepsStableKeys(t *testing.T) {
	files := testItemFiles{
		"core/items/treasurecards.yaml":    "tc_storm_trap:\n  school: storm\ntc_ice_trap: {}\n",
		"i18n/de/items/treasurecards.json": `{"tc_storm_trap":"Sturmfalle"}`,
		"core/items/pets.yaml":             "{}\n",
	}
	repo := NewRepository(files, "de")
	catalog, err := repo.Category(context.Background(), "treasurecards")
	if err != nil || len(catalog) != 2 || catalog["tc_storm_trap"].ID != "tc_storm_trap" || catalog["tc_storm_trap"].Name != "Sturmfalle" || catalog["tc_ice_trap"].Name != "tc_ice_trap" {
		t.Fatalf("YAML catalog: %+v, %v", catalog, err)
	}
	delete(files, "core/items/treasurecards.yaml")
	delete(catalog, "tc_storm_trap")
	cached, err := repo.Category(context.Background(), "treasurecards")
	if err != nil || len(cached) != 2 {
		t.Fatal("cache changed or reloaded")
	}
	pets, err := repo.Category(context.Background(), "pets")
	if err != nil || len(pets) != 0 {
		t.Fatalf("empty category: %+v, %v", pets, err)
	}
}
