//go:build windows

package windows

import (
	"corvin101/internal/data/enemies"
	"corvin101/internal/overlay/widget"
	"testing"
)

func TestPanelMousePaging(t *testing.T) {
	w := &Window{panelOpen: true, width: 1280, height: 720, selectedEnemy: "keep", panelEnemies: make([]enemies.WorldEnemy, panelRows(720)*3)}
	list, details := enemyPanelColumns(w.width, w.height)
	p := widget.Point{X: list.Left + 10, Y: list.Top + 10}
	if !w.wheelPanel(p, -60) || w.panelPage != 0 {
		t.Fatal("partial wheel changed page")
	}
	w.wheelPanel(p, -60)
	if w.panelPage != 1 {
		t.Fatal("wheel did not accumulate")
	}
	w.wheelPanel(p, -1200)
	if w.panelPage != 2 {
		t.Fatal("last page not clamped")
	}
	w.wheelPanel(p, 1200)
	if w.panelPage != 0 {
		t.Fatal("first page not clamped")
	}
	if w.wheelPanel(widget.Point{X: details.Left + 10, Y: details.Top + 10}, -120) || w.panelPage != 0 {
		t.Fatal("wheel outside list consumed")
	}
	previous, next, _ := pageControls(w.width, w.height)
	click := func(r widget.Rect) { w.clickPanel(widget.Point{X: r.Left + 5, Y: r.Top + 5}) }
	click(previous)
	if w.panelPage != 0 {
		t.Fatal("disabled previous changed page")
	}
	click(next)
	if w.panelPage != 1 || w.selectedEnemy != "keep" {
		t.Fatal("next button failed or selection lost")
	}
	click(previous)
	if w.panelPage != 0 {
		t.Fatal("previous button failed")
	}
	w.worldMenuOpen = true
	if w.wheelPanel(p, -120) {
		t.Fatal("wheel consumed while world chooser open")
	}
	w.worldMenuOpen = false
	w.panelOpen = false
	if w.wheelPanel(p, -120) {
		t.Fatal("closed panel consumed wheel")
	}
}
