package enemies

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
)

// Preload warms every bestiary world, independently of game sessions and UI.
// Per-world synchronization also shares requests with interactive lookups.
func (r *Repository) Preload(ctx context.Context) error {
	worlds, err := r.zones.ListWorlds(ctx)
	if err != nil {
		return fmt.Errorf("load world list: %w", err)
	}
	jobs := make(chan string)
	var workers sync.WaitGroup
	var resultMu sync.Mutex
	var failures []error
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for world := range jobs {
				if ctx.Err() != nil {
					return
				}
				if err := r.ensureWorld(ctx, world); err != nil {
					resultMu.Lock()
					failures = append(failures, fmt.Errorf("%s: %w", world, err))
					resultMu.Unlock()
				}
			}
		}()
	}
	total := 0
enqueue:
	for _, world := range worlds {
		if world.Key == "Housing" || world.Key == "Minigames" {
			continue
		}
		select {
		case <-ctx.Done():
			break enqueue
		case jobs <- world.Key:
			total++
		}
	}
	close(jobs)
	workers.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if len(failures) > 0 {
		return errors.Join(failures...)
	}
	log.Printf("[bestiary] preloaded %d worlds", total)
	return nil
}
