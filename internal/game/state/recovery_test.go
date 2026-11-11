package state

import (
	"corvin101/internal/game/events"
	"testing"
)

func TestRosterRecoveryRejectsMixedAndIncompleteAssignments(t *testing.T) {
	makeStore := func(names []string) *Store {
		s := New()
		s.combat.ClientDuelID = "duel"
		s.HandleEvent(events.Event{Type: events.ParticipantAdded, Participant: &events.ParticipantEvent{ClientDuelID: "duel", Participant: events.Participant{ID: "local", Kind: events.ParticipantLocalPlayer, SubCircle: 5}}})
		s.HandleEvent(events.Event{Type: events.DuelRosterReceived, Roster: &events.DuelRoster{ClientDuelID: "duel", Names: names}})
		s.HandleEvent(events.Event{Type: events.DuelSnapshotReceived, Snapshot: &events.DuelSnapshot{ClientDuelID: "duel", Participants: []events.Participant{{ID: "local", SubCircle: -1, Team: 0}, {ID: "a", SubCircle: -1, Team: 1}, {ID: "b", SubCircle: -1, Team: 1}}}})
		return s
	}
	s := makeStore([]string{"wolf", "troll"})
	if s.combat.Participants["a"].Kind != events.ParticipantUnknown || s.combat.Participants["b"].Kind != events.ParticipantUnknown {
		t.Fatal("mixed opponents guessed by order")
	}
	s.HandleEvent(events.Event{Type: events.ParticipantMapped, ParticipantMapping: &events.ParticipantMappingEvent{ClientDuelID: "duel", ParticipantID: "b", SubCircle: 1, Kind: events.ParticipantMob, MobID: "wolf"}})
	if s.combat.Participants["a"].MobID != "troll" {
		t.Fatal("unique remaining opponent not recovered")
	}
	s = makeStore([]string{"wolf", "wolf"})
	if s.combat.Participants["a"].MobID != "wolf" || s.combat.Participants["b"].MobID != "wolf" {
		t.Fatal("identical opponents not recovered")
	}
	s = makeStore([]string{"wolf"})
	if s.combat.Participants["a"].Kind != events.ParticipantUnknown || s.combat.Participants["b"].Kind != events.ParticipantUnknown {
		t.Fatal("incomplete roster used")
	}
	s.HandleEvent(events.Event{Type: events.DuelSnapshotReceived, Snapshot: &events.DuelSnapshot{ClientDuelID: "other", Participants: []events.Participant{{ID: "a", SubCircle: -1, Team: 1}}}})
	if s.combat.Participants["a"].MobID != "" {
		t.Fatal("roster leaked across duels")
	}
}
