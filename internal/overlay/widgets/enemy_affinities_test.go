package widgets

import (
	"corvin101/internal/data/enemies"
	"testing"
)

func TestExplicitZeroAffinityIsKnown(t *testing.T) {
	groups := groupAffinities(map[enemies.School]int{enemies.SchoolBalance: 0})
	if len(groups) != 1 || groups[0].Value != 0 || len(groups[0].Schools) != 1 || groups[0].Schools[0] != enemies.SchoolBalance {
		t.Fatalf("explicit zero lost: %+v", groups)
	}
	if groups := groupAffinities(nil); len(groups) != 0 {
		t.Fatalf("unknown affinities became known: %+v", groups)
	}
}
