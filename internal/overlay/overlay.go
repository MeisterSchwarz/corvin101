package overlay

import (
	"context"
	"sync"

	"corvin101/internal/game/state"
)

type Window interface {
	Run(ctx context.Context) error
	Update(snapshot state.Snapshot)
}

type Overlay struct {
	mu sync.Mutex

	window Window
}

func New(window Window) *Overlay {
	return &Overlay{
		window: window,
	}
}

func (o *Overlay) Run(ctx context.Context) error {
	if o.window == nil {
		return nil
	}

	return o.window.Run(ctx)
}

func (o *Overlay) Update(snapshot state.Snapshot) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.window == nil {
		return
	}

	o.window.Update(snapshot)
}
