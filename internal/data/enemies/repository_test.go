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
