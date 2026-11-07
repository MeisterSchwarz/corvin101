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
		// ZoneLoaded can occur while a duel is still active.
		if !s.combat.Active {
			s.game.Mode = ModeRoaming
		}

	case events.CharacterSelected:
		s.game.Mode = ModeCharacterSelect
		s.resetCombat()

	case events.CombatJoined:
		s.game.Mode = ModeBattle

	case events.CombatPhaseChanged:
		if event.Combat != nil && event.Combat.Phase == events.PhaseEnded {
			// Nur die Phase des aktuell verfolgten Kampfes auswerten.
			if event.Combat.ClientDuelID == "" ||
				s.combat.ClientDuelID == "" ||
				event.Combat.ClientDuelID == s.combat.ClientDuelID {
				s.game.Mode = ModeRoaming
			}
		}

	case events.ParticipantRemoved:
		if event.Participant != nil && event.Participant.RawName == "Local Player" {
			s.game.Mode = ModeRoaming
		}

	case events.CombatFled,
		events.CombatLeft:
		s.game.Mode = ModeRoaming
	}

	s.handleCombatEvent(event)
}
