package enemy

import (
	"log"
	"regexp"
)

type EnemyTracker struct {
	duelID  string
	won     bool
	enemies map[string]int
}

var (
	removeRe  = regexp.MustCompile(`Duel (\d+): MSG_CombatRemove\(([^)]+)\)`)
	victoryRe = regexp.MustCompile(`Duel (\d+): MSG_CombatPhase\(kPhase_Victory\)`)
	endRe     = regexp.MustCompile(`Duel (\d+): MSG_EndDuel received`)
)

func NewEnemyTracker() *EnemyTracker {
	return &EnemyTracker{
		enemies: map[string]int{},
	}
}

func (t *EnemyTracker) HandleLogLine(line string) {
	if m := removeRe.FindStringSubmatch(line); m != nil {
		t.ensureDuel(m[1])
		t.enemies[m[2]]++
		return
	}

	if m := victoryRe.FindStringSubmatch(line); m != nil {
		t.ensureDuel(m[1])
		t.won = true
		return
	}

	if m := endRe.FindStringSubmatch(line); m != nil {
		if t.duelID != m[1] {
			return
		}

		if t.won && len(t.enemies) > 0 {
			log.Printf("[combat] won duel %s", t.duelID)
			for enemy, count := range t.enemies {
				log.Printf("[combat] %dx %s", count, enemy)
			}
		}

		t.reset()
	}
}

func (t *EnemyTracker) ensureDuel(duelID string) {
	if t.duelID == duelID {
		return
	}

	t.duelID = duelID
	t.won = false
	t.enemies = map[string]int{}
}

func (t *EnemyTracker) reset() {
	t.duelID = ""
	t.won = false
	t.enemies = map[string]int{}
}
