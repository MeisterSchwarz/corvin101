package zones

import (
	"context"
	"io/fs"
	"testing"
)

type zoneTestFiles map[string]string

func (f zoneTestFiles) ReadFile(_ context.Context, path string) ([]byte, error) {
	if data, ok := f[path]; ok {
		return []byte(data), nil
	}
	return nil, fs.ErrNotExist
}

func TestResolveFlatYAMLZones(t *testing.T) {
	files := zoneTestFiles{
		"core/worlds.json":              `{"WizardCity":{"image":"wizardcity"}}`,
		"i18n/de/worlds.json":           `{"WizardCity":{"name":"Zauberstadt"}}`,
		"core/zones/WizardCity.yaml":    "WizardCity/Test:\n  type: dungeon\n  parent: WizardCity/Hub\n  instance: true\n  repeatable: true\nWizardCity/Hub: {}\n",
		"i18n/de/zones/WizardCity.json": `{"WizardCity/Test":["Testgebiet","Untertitel"],"WizardCity/Hub":"Zentrum"}`,
	}
	repo := NewRepository("de", NewRemoteLoader(files))
	info, found, err := repo.Resolve(context.Background(), "WizardCity/Test")
	if err != nil || !found {
		t.Fatalf("resolve: found=%v err=%v", found, err)
	}
	if info.Name != "Testgebiet" || info.Sub != "Untertitel" || info.World != "Zauberstadt" || info.Image != "wizardcity" || info.Type != "dungeon" || info.Parent != "" || !info.Instance || !info.Repeatable {
		t.Fatalf("lost data: %+v", info)
	}
	hub, found, err := repo.Resolve(context.Background(), "WizardCity/Hub")
	if err != nil || !found || hub.Name != "Zentrum" || hub.Instance || hub.Repeatable {
		t.Fatalf("empty properties: %+v, %v, %v", hub, found, err)
	}
}

func TestListWorldsOrderAndFallback(t *testing.T) {
	repo := NewRepository("de", NewRemoteLoader(zoneTestFiles{
		"core/worlds.json":    `{"Avalon":{"order":8},"WizardCity":{"order":1},"Unknown":{"order":9}}`,
		"i18n/de/worlds.json": `{"WizardCity":{"name":"Wizard City"},"Avalon":{"name":"Avalon"}}`,
	}))
	worlds, err := repo.ListWorlds(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(worlds) != 3 || worlds[0].Key != "WizardCity" || worlds[0].Name != "Wizard City" || worlds[1].Key != "Avalon" || worlds[2].Name != "Unknown" {
		t.Fatalf("unexpected catalog: %+v", worlds)
	}
}

func TestZoneIDsAreWorldScopedAndParentIgnored(t *testing.T) {
	repo := NewRepository("de", NewRemoteLoader(zoneTestFiles{
		"core/worlds.json":          `{"Avalon":{},"Novus":{}}`,
		"i18n/de/worlds.json":       "{}",
		"core/zones/Avalon.yaml":    "Avalon/A:\n  id: 500\n  parent: 400\nAvalon/NoID: {}\n",
		"i18n/de/zones/Avalon.json": `{"Avalon/A":"Gebiet A"}`,
		"core/zones/Novus.yaml":     "Novus/B:\n  id: 500\n",
		"i18n/de/zones/Novus.json":  `{"Novus/B":"Gebiet B"}`,
	}))
	for world, want := range map[string]string{"Avalon": "Gebiet A", "Novus": "Gebiet B"} {
		info, found, err := repo.ResolveID(context.Background(), world, 500)
		if err != nil || !found || info.Name != want || info.Parent != "" {
			t.Fatalf("%s: %+v %v %v", world, info, found, err)
		}
	}
	if _, found, err := repo.ResolveID(context.Background(), "Avalon", 0); err != nil || found {
		t.Fatal("zone without ID matched zero")
	}
}
