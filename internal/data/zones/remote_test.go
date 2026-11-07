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
	if info.Name != "Testgebiet" || info.Sub != "Untertitel" || info.World != "Zauberstadt" || info.Image != "wizardcity" || info.Type != "dungeon" || info.Parent != "WizardCity/Hub" || !info.Instance || !info.Repeatable {
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
