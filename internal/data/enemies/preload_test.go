package enemies

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type preloadTestFiles struct {
	mu    sync.Mutex
	data  testFiles
	calls map[string]int
}

func (f *preloadTestFiles) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.calls[path]++
	f.mu.Unlock()
	return f.data.ReadFile(ctx, path)
}
func (f *preloadTestFiles) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, count := range f.calls {
		n += count
	}
	return n
}

func TestPreloadCachesWorldsBeforeSelection(t *testing.T) {
	files := &preloadTestFiles{calls: make(map[string]int), data: testFiles{
		"core/worlds.json":         `{"Avalon":{},"Novus":{},"Empty":{},"Housing":{},"Minigames":{}}`,
		"i18n/de/worlds.json":      "{}",
		"core/enemies/Avalon.yaml": "shared:\n  schools: [fire]\n",
		"core/enemies/Novus.yaml":  "shared:\n  schools: [ice]\n",
	}}
	repo := NewRepository("de", NewRemoteLoader(files))
	if err := repo.Preload(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, world := range []string{"Avalon", "Novus", "Empty"} {
		if _, cached := repo.CachedWorld(world); !cached {
			t.Fatalf("%s not preloaded", world)
		}
	}
	if _, cached := repo.CachedWorld("Housing"); cached {
		t.Fatal("housing preloaded")
	}
	if _, cached := repo.CachedWorld("Minigames"); cached {
		t.Fatal("minigames preloaded")
	}
	before := files.count()
	for world, school := range map[string]School{"Avalon": SchoolFire, "Novus": SchoolIce} {
		if _, err := repo.ListWorld(context.Background(), world); err != nil {
			t.Fatal(err)
		}
		info, found, err := repo.Resolve(context.Background(), world+"/Hub", "shared")
		if err != nil || !found || info.Schools[0] != school {
			t.Fatalf("world IDs collided in %s: %+v %v", world, info, err)
		}
	}
	if _, err := repo.ListWorld(context.Background(), "Empty"); err != nil {
		t.Fatal(err)
	}
	if files.count() != before {
		t.Fatal("cached selection performed network reads")
	}
}

func TestPreloadContinuesAfterInvalidWorld(t *testing.T) {
	files := testFiles{
		"core/worlds.json": `{"Avalon":{},"Novus":{}}`, "i18n/de/worlds.json": "{}",
		"core/enemies/Avalon.yaml": "enemy: {}",
		"core/enemies/Novus.yaml":  "enemy:\n  affinities:\n    death_ 50\n",
	}
	repo := NewRepository("de", NewRemoteLoader(files))
	if err := repo.Preload(context.Background()); err == nil {
		t.Fatal("invalid world not reported")
	}
	if _, cached := repo.CachedWorld("Avalon"); !cached {
		t.Fatal("valid world lost")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repo.Preload(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}
