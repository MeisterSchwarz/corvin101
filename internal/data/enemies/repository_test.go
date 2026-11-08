package enemies

import (
	"context"
	"io/fs"
	"testing"
)

type testFiles map[string]string

func (f testFiles) ReadFile(_ context.Context, path string) ([]byte, error) {
	if data, ok := f[path]; ok {
		return []byte(data), nil
	}
	return nil, fs.ErrNotExist
}

func TestListWorld(t *testing.T) {
	files := testFiles{
		"core/enemies/Avalon.yaml":     "z: {}\na:\n  affinities:\n    storm: 35\n    myth: -40\n",
		"i18n/de/enemies/Avalon.json":  `{"a":"Ein Gegner","orphan":"Kein Core-Eintrag"}`,
		"core/enemies/Marleybone.json": `{"other":{}}`,
	}
	repo := NewRepository("de", NewRemoteLoader(files))
	ctx := context.Background()
	if _, err := repo.ListWorld(ctx, "Marleybone"); err != nil {
		t.Fatal(err)
	}
	entries, err := repo.ListWorld(ctx, "Avalon")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Key != "a" || entries[0].Name != "Ein Gegner" || entries[1].Name != "z" {
		t.Fatalf("unexpected list: %+v", entries)
	}
	if entries[0].Affinities[SchoolStorm] != 35 || entries[0].Affinities[SchoolMyth] != -40 {
		t.Fatalf("affinities lost: %+v", entries[0])
	}
	entries[0].Affinities[SchoolStorm] = 999
	again, err := repo.ListWorld(ctx, "Avalon")
	if err != nil || again[0].Affinities[SchoolStorm] != 35 {
		t.Fatalf("cache modified: %+v, %v", again, err)
	}
	other, err := repo.ListWorld(ctx, "Marleybone")
	if err != nil || len(other) != 1 || other[0].Name != "other" {
		t.Fatalf("JSON / missing translation fallback: %+v, %v", other, err)
	}
}

func TestFailedWorldCanBeRetried(t *testing.T) {
	files := testFiles{"core/enemies/Novus.yaml": "enemy:\n  affinities:\n    death_ 50\n"}
	repo := NewRepository("de", NewRemoteLoader(files))
	if _, err := repo.ListWorld(context.Background(), "Novus"); err == nil {
		t.Fatal("expected malformed YAML to fail")
	}
	files["core/enemies/Novus.yaml"] = "enemy:\n  affinities:\n    death: 50\n"
	entries, err := repo.ListWorld(context.Background(), "Novus")
	if err != nil || len(entries) != 1 || entries[0].Affinities[SchoolDeath] != 50 {
		t.Fatalf("retry failed: %+v, %v", entries, err)
	}
}

func TestEnemySchoolOrderAndCacheIsolation(t *testing.T) {
	repo := NewRepository("de", NewRemoteLoader(testFiles{
		"core/enemies/Selenopolis.yaml": "dual:\n  schools: [ice, balance]\n  affinities:\n    ice: -46\nsingle:\n  schools: [storm]\nunspecified: {}\n",
	}))
	ctx := context.Background()
	entries, err := repo.ListWorld(ctx, "Selenopolis")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 || len(entries[0].Schools) != 2 || entries[0].Schools[0] != SchoolIce || entries[0].Schools[1] != SchoolBalance {
		t.Fatalf("school order lost: %+v", entries)
	}
	if len(entries[1].Schools) != 1 || entries[1].Schools[0] != SchoolStorm || len(entries[2].Schools) != 0 {
		t.Fatalf("single/missing schools: %+v", entries)
	}
	entries[0].Schools[0] = SchoolDeath
	info, found, err := repo.Resolve(ctx, "Selenopolis/Hub", "dual")
	if err != nil || !found || len(info.Schools) != 2 || info.Schools[0] != SchoolIce || info.Schools[1] != SchoolBalance {
		t.Fatalf("resolve/cache: %+v, %v", info, err)
	}
	info.Schools[0] = SchoolFire
	again, err := repo.ListWorld(ctx, "Selenopolis")
	if err != nil || again[0].Schools[0] != SchoolIce {
		t.Fatalf("resolve mutated cached schools: %+v, %v", again, err)
	}
}

func TestCachedWorldNeverLoadsAndPreservesEmptyWorld(t *testing.T) {
	files := testFiles{"core/enemies/Empty.yaml": "{}"}
	repo := NewRepository("de", NewRemoteLoader(files))
	if _, ok := repo.CachedWorld("Empty"); ok {
		t.Fatal("unloaded world reported cached")
	}
	if _, err := repo.ListWorld(context.Background(), "Empty"); err != nil {
		t.Fatal(err)
	}
	delete(files, "core/enemies/Empty.yaml")
	if entries, ok := repo.CachedWorld("Empty"); !ok || len(entries) != 0 {
		t.Fatalf("empty cache hit lost: %v, %v", entries, ok)
	}
	if _, ok := repo.CachedWorld("Other"); ok {
		t.Fatal("unknown world reported cached")
	}
}

func TestTreasureCardDropsResolveToEnemyAndKeepIDs(t *testing.T) {
	files := testFiles{
		"core/enemies/Selenopolis.yaml":    "KT_SE_Elemental_MudBall_A_01:\n  schools: [ice]\nKT_SE_Elemental_MudBall_A_02:\n  schools: [balance]\n",
		"core/drops/Selenopolis.yaml":      "KT_SE_Elemental_MudBall_A_02:\n  treasurecards:\n    - item: tc_storm_trap\n    - item: tc_ice_trap\n",
		"core/items/treasurecards.json":    `{"tc_storm_trap":{"school":"storm"},"tc_ice_trap":{"school":"ice"}}`,
		"i18n/de/items/treasurecards.json": `{"tc_storm_trap":"Sturmfalle","tc_ice_trap":"Eisfalle"}`,
	}
	repo := NewRepository("de", NewRemoteLoader(files))
	entries, err := repo.ListWorld(context.Background(), "Selenopolis")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || len(entries[0].DropCategories) != 0 {
		t.Fatalf("drops assigned to wrong enemy: %+v", entries)
	}
	categories := entries[1].DropCategories
	if len(categories) != 1 || categories[0].Key != "treasurecards" || len(categories[0].Items) != 2 {
		t.Fatalf("categories: %+v", categories)
	}
	if categories[0].Items[0].ID != "tc_ice_trap" || categories[0].Items[0].Name != "Eisfalle" || categories[0].Items[1].ID != "tc_storm_trap" || categories[0].Items[1].Name != "Sturmfalle" {
		t.Fatalf("IDs or names lost: %+v", categories[0].Items)
	}
	categories[0].Items[0].Name = "modified"
	again, _, err := repo.Resolve(context.Background(), "Krokotopia/KT_Selenopolis/Hub", "KT_SE_Elemental_MudBall_A_02")
	if err != nil || again.DropCategories[0].Items[0].Name != "Eisfalle" {
		t.Fatal("caller mutated cached drops")
	}
}

func TestDropFailureDoesNotHideEnemies(t *testing.T) {
	files := testFiles{
		"core/enemies/Selenopolis.yaml": "enemy: {}",
		"core/drops/Selenopolis.yaml":   "enemy:\n  treasurecards:\n    - item: missing",
		"core/items/treasurecards.json": "{}",
	}
	repo := NewRepository("de", NewRemoteLoader(files))
	entries, err := repo.ListWorld(context.Background(), "Selenopolis")
	if err != nil || len(entries) != 1 || !entries[0].DropsUnavailable {
		t.Fatalf("expected visible enemy with unavailable drops: %+v, %v", entries, err)
	}
}

func TestLocationsResolveWithinWorldAndIgnoreParent(t *testing.T) {
	files := testFiles{
		"core/enemies/Selenopolis.yaml":  "enemy:\n  locations: [501, 999]\n",
		"core/worlds.json":               `{"Selenopolis":{},"Novus":{}}`,
		"i18n/de/worlds.json":            `{"Selenopolis":{"name":"Selenopolis"}}`,
		"core/zones/Selenopolis.yaml":    "Krokotopia/KT_Selenopolis/Interior:\n  id: 501\n  parent: 500\nKrokotopia/KT_Selenopolis/Hub: {}\n",
		"i18n/de/zones/Selenopolis.json": `{"Krokotopia/KT_Selenopolis/Interior":["Marktplatz der Ideen","Halle des Vorwortes"]}`,
	}
	repo := NewRepository("de", NewRemoteLoader(files))
	entries, err := repo.ListWorld(context.Background(), "Selenopolis")
	if err != nil || len(entries) != 1 {
		t.Fatalf("load: %+v %v", entries, err)
	}
	info := entries[0]
	if len(info.Locations) != 2 || info.Locations[0] != 501 || len(info.LocationNames) != 2 || info.LocationNames[0] != "Marktplatz der Ideen – Halle des Vorwortes" || info.LocationNames[1] != "Gebiet 999" {
		t.Fatalf("locations: %+v", info)
	}
	entries[0].Locations[0] = 0
	entries[0].LocationNames[0] = "changed"
	again, _ := repo.CachedWorld("Selenopolis")
	if again[0].Locations[0] != 501 || again[0].LocationNames[0] == "changed" {
		t.Fatal("location cache mutated")
	}
}
