//go:build windows

package windows

import (
	"corvin101/internal/data/enemies"
	"corvin101/internal/overlay/widgets"
	"testing"
)

func TestAffinityPlaqueBoundsContainWrappedGroups(t *testing.T) {
	for _, scale := range []float64{0.65, 1, 1.5} {
		groups := []widgets.EnemyAffinityGroup{
			{Value: 20, Schools: []enemies.School{enemies.SchoolMyth, enemies.SchoolLife, enemies.SchoolDeath}},
			{Value: -40, Schools: []enemies.School{enemies.SchoolBalance}},
			{Value: 0, Schools: []enemies.School{enemies.SchoolFire, enemies.SchoolIce, enemies.SchoolStorm, enemies.SchoolShadow}},
		}
		layout := layoutAffinityPlaque(groups, scale)
		if len(layout.groups) != 3 || layout.groups[0].columns != 3 || layout.groups[2].columns != 3 {
			t.Fatal("incorrect wrapping")
		}
		for _, g := range layout.groups {
			if g.bounds.Left < 0 || g.bounds.Top < 0 || g.bounds.Right > layout.width || g.bounds.Bottom > layout.height {
				t.Fatalf("group outside hover region: %+v", g)
			}
		}
		for slot := 0; slot < 4; slot++ {
			bounds := enemyAffinitySlotBounds(1280, 720, slot, groups, scale)
			if bounds.Left < 0 || bounds.Right > 1280 || bounds.Width() != layout.width || bounds.Height() != layout.height {
				t.Fatalf("invalid slot %d: %+v", slot, bounds)
			}
		}
	}
}
