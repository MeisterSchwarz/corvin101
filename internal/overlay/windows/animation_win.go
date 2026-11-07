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
	var lastPanelFrame time.Time

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			w.mu.RLock()
			manager := w.manager
			combatRows := 0
			if w.panelOpen && !w.worldMenuOpen {
				for _, enemy := range w.panelEnemies {
					if w.panelCombatIDs[enemy.Key] {
						combatRows++
					}
				}
			}
			panelAnimating := combatRows > w.panelPage*panelRows(w.height)
			w.mu.RUnlock()

			w.updateMu.Lock()
			animating := manager != nil && manager.Animating()
			w.updateMu.Unlock()
			panelFrame := panelAnimating && time.Since(lastPanelFrame) >= time.Second/30
			if !animating && !panelFrame {
				continue
			}
			if panelFrame {
				lastPanelFrame = time.Now()
			}

			w.render()
		}
	}
}
