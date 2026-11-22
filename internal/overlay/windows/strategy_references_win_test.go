//go:build windows

package windows

import (
	"corvin101/internal/data/enemies"
	"corvin101/internal/overlay/widget"
	"testing"
)

func TestStrategyLinkOpensCorrectVariant(t *testing.T) {
	w := &Window{panelOpen: true, detailCheats: true, width: 1280, height: 720, selectedEnemy: "boss", strategyLinksEnemy: "boss", panelEnemies: []enemies.WorldEnemy{{Key: "boss"}, {Key: "life"}, {Key: "hard-life", EnemyInfo: enemies.EnemyInfo{Mode: "challenge", VariantOf: "life"}}}}
	_, details := enemyPanelColumns(w.width, w.height)
	r := widget.Rect{Left: details.Left + 20, Right: details.Left + 200, Top: details.Top + 200, Bottom: details.Top + 230}
	w.strategyLinks = []strategyEnemyLink{{Bounds: r, Family: "life", Mode: "challenge"}}
	w.clickPanel(widget.Point{X: r.Left + 1, Y: r.Top + 1})
	if w.selectedEnemy != "life" || w.enemyModes["life"] != "challenge" || w.detailCheats {
		t.Fatal("reference did not navigate to challenge enemy")
	}
	if w.strategyLinkAtLocked(widget.Point{X: r.Left + 1, Y: r.Top + 1}) != nil {
		t.Fatal("stale link remained clickable")
	}
}
