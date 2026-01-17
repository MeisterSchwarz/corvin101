package state

import (
	"sync"
	"time"
)

// Game state
type Mode int

const (
	None Mode = iota
	Roaming
	Battle
	CharacterSelect
)

var (
	mu sync.RWMutex

	GameRunning     bool
	LastLogActivity time.Time

	current      Mode
	battleStart  *time.Time
	lastZone     string
	sessionStart *time.Time
)

// returns the active game mode
func Current() Mode {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// returns the battle start time
func BattleStart() *time.Time {
	mu.RLock()
	defer mu.RUnlock()
	return battleStart
}

// returns the last known zone
func LastZone() string {
	mu.RLock()
	defer mu.RUnlock()
	return lastZone
}

// returns the session start time
func SessionStart() *time.Time {
	mu.RLock()
	defer mu.RUnlock()
	return sessionStart
}

// initializes a new session
func StartSession() {
	now := time.Now()
	mu.Lock()
	sessionStart = &now
	mu.Unlock()
}

/*
	State transitions
*/

// switches state to Battle
func EnterBattle() bool {
	mu.Lock()
	defer mu.Unlock()

	if current == Battle {
		return false
	}

	now := time.Now()
	current = Battle
	battleStart = &now
	return true
}

// switches state back to Roaming
func LeaveBattle() bool {
	mu.Lock()
	defer mu.Unlock()

	if current != Battle {
		return false
	}

	current = Roaming
	battleStart = nil
	return true
}

// updates zone and roaming state
func EnterZone(zone string) bool {
	if zone == "" {
		return false
	}

	mu.Lock()
	defer mu.Unlock()

	changed := false

	if zone != lastZone {
		lastZone = zone
		changed = true
	}

	if current != Roaming {
		current = Roaming
		changed = true
	}

	return changed
}

// switches to character select
func EnterCharacterSelect() bool {
	mu.Lock()
	defer mu.Unlock()

	if current == CharacterSelect {
		return false
	}

	current = CharacterSelect
	return true
}

// clears all runtime state
func Reset() {
	mu.Lock()
	defer mu.Unlock()

	current = None
	lastZone = ""
	battleStart = nil
	sessionStart = nil
	GameRunning = false
}
