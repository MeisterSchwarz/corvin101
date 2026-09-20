package state

import (
	"sync"
	"time"

	"corvin101/internal/game/events"
)

type Store struct {
	mu sync.RWMutex

	game   GameState
	combat CombatState
}

func New() *Store {
	store := &Store{}
	store.Reset()

	return store
}

func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.game = GameState{
		Mode:             ModeUnknown,
		SessionStartedAt: time.Now(),
	}

	s.combat = newCombatState()
}

func (s *Store) HandleEvent(event events.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch event.Type {
	case events.ZoneLoaded:
		if event.Zone == nil {
			return
		}

		s.game.ZoneKey = event.Zone.ZoneKey
		s.game.Mode = ModeRoaming

	case events.CharacterSelected:
		s.game.Mode = ModeCharacterSelect

	case events.CombatJoined:
		s.game.Mode = ModeBattle

	case events.CombatFled,
		events.CombatLeft:
		s.game.Mode = ModeRoaming
	}

	s.handleCombatEvent(event)
}
