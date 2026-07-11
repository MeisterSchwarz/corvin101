package enemy

import (
	"regexp"
	"sync"
)

type EnemyTracker struct {
	mu sync.RWMutex

	currentZoneKey string
	duelZoneKey    string

	duelID  string
	won     bool
	ended   bool
	enemies map[string]int
}

type Snapshot struct {
	DuelID string `json:"duelId"`

	CurrentZoneKey string `json:"currentZoneKey"`
	DuelZoneKey    string `json:"duelZoneKey"`

	Won     bool           `json:"won"`
	Ended   bool           `json:"ended"`
	Active  bool           `json:"active"`
	Enemies map[string]int `json:"enemies"`
}

var (
	removeRe = regexp.MustCompile(
		`Duel (\d+): MSG_CombatRemove\(([^)]+)\)`,
	)

	victoryRe = regexp.MustCompile(
		`Duel (\d+): MSG_CombatPhase\(kPhase_Victory\)`,
	)

	endRe = regexp.MustCompile(
		`Duel (\d+): MSG_EndDuel received`,
	)
)

func NewEnemyTracker() *EnemyTracker {
	return &EnemyTracker{
		enemies: make(map[string]int),
	}
}

// SetZoneKey sollte aufgerufen werden, sobald die aktuelle Zone bekannt ist.
// Bei einem bereits laufenden Kampf wird die Kampfzone nicht mehr überschrieben.
func (t *EnemyTracker) SetZoneKey(zoneKey string) {
	if zoneKey == "" {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.currentZoneKey = zoneKey
}

func (t *EnemyTracker) HandleLogLine(line string) {
	t.mu.Lock()
	defer t.mu.Unlock()

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

		t.ended = true
	}
}

func (t *EnemyTracker) Snapshot() Snapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	enemies := make(map[string]int, len(t.enemies))

	for enemyID, count := range t.enemies {
		enemies[enemyID] = count
	}

	return Snapshot{
		DuelID: t.duelID,

		CurrentZoneKey: t.currentZoneKey,
		DuelZoneKey:    t.duelZoneKey,

		Won:     t.won,
		Ended:   t.ended,
		Active:  t.duelID != "" && !t.ended,
		Enemies: enemies,
	}
}

func (t *EnemyTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.reset()
}

func (t *EnemyTracker) ensureDuel(duelID string) {
	if t.duelID == duelID {
		return
	}

	t.duelID = duelID
	t.duelZoneKey = t.currentZoneKey
	t.won = false
	t.ended = false
	t.enemies = make(map[string]int)
}

func (t *EnemyTracker) reset() {
	t.duelID = ""
	t.duelZoneKey = ""
	t.won = false
	t.ended = false
	t.enemies = make(map[string]int)
}
