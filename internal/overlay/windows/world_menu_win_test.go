//go:build windows

package windows

import (
	"corvin101/internal/overlay/widget"
	"testing"
)

func TestWorldGridHitAreas(t *testing.T) {
	for _, viewport := range []widget.Viewport{{Width: 1920, Height: 1080}, {Width: 1280, Height: 720}, {Width: 800, Height: 600}} {
		bounds := panelBounds(viewport.Width, viewport.Height)
		top := bounds.Top + viewport.Height/12
		for i := 0; i < 27; i++ {
			cell := worldCell(bounds, top, 27, i)
			center := widget.Point{X: (cell.Left + cell.Right) / 2, Y: (cell.Top + cell.Bottom) / 2}
			if !bounds.Contains(center) {
				t.Fatalf("cell %d outside panel: %+v", i, cell)
			}
			hits := 0
			for j := 0; j < 27; j++ {
				if worldCell(bounds, top, 27, j).Contains(center) {
					hits++
				}
			}
			if hits != 1 {
				t.Fatalf("cell %d has %d hit targets", i, hits)
			}
		}
	}
}

func TestManualWorldSurvivesZoneUpdates(t *testing.T) {
	w := &Window{}
	w.updatePanelWorld("WizardCity/Hub")
	w.selectPanelWorld("Avalon")
	w.updatePanelWorld("WizardCity/Hub")
	w.updatePanelWorld("WizardCity/OtherZone")
	if w.panelWorld != "Avalon" {
		t.Fatalf("manual selection overwritten: %s", w.panelWorld)
	}
	// Reopening uses the real zone, not the browsed world.
	w.selectPanelWorld("WizardCity")
	if w.panelWorld != "WizardCity" || w.panelPage != 0 {
		t.Fatalf("reset failed: %s / %d", w.panelWorld, w.panelPage)
	}
}
