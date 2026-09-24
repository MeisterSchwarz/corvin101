//go:build windows

package windows

import (
	"context"
	"time"
)

const animationFrameInterval = time.Second / 60

func (w *Window) runAnimationLoop(ctx context.Context) {
	ticker := time.NewTicker(animationFrameInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			w.mu.RLock()
			manager := w.manager
			w.mu.RUnlock()

			if manager == nil {
				continue
			}

			if !manager.Animating() {
				continue
			}

			w.render()
		}
	}
}
