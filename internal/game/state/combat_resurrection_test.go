package state

import (
	"corvin101/internal/game/events"
	"testing"
)

func TestResurrectedParticipantRestoresIdentityWithinDuel(t *testing.T) {
	s := New()
	mapping := func(id, mob string) {
		s.HandleEvent(events.Event{Type: events.ParticipantMapped, ParticipantMapping: &events.ParticipantMappingEvent{ClientDuelID: "duel", ParticipantID: id, SubCircle: 0, Kind: events.ParticipantMob, MobID: mob}})
	}
	snapshot := func(duel, id string) {
		s.HandleEvent(events.Event{Type: events.DuelSnapshotReceived, Snapshot: &events.DuelSnapshot{ClientDuelID: duel, Participants: []events.Participant{{ID: id, SubCircle: -1, Health: 100, Team: 1}}}})
	}
	mapping("boss", "known-boss")
	s.HandleEvent(events.Event{Type: events.ParticipantRemoved, Participant: &events.ParticipantEvent{ClientDuelID: "duel", RawName: "known-boss"}})
	if _, ok := s.ParticipantBySubCircle(0); ok {
		t.Fatal("removed opponent remained active")
	}
	snapshot("duel", "boss")
	boss, ok := s.ParticipantBySubCircle(0)
	if !ok || boss.MobID != "known-boss" || boss.Kind != events.ParticipantMob || boss.Health != 100 {
		t.Fatalf("resurrection lost identity: %+v", boss)
	}
	s.HandleEvent(events.Event{Type: events.ParticipantRemoved, Participant: &events.ParticipantEvent{ClientDuelID: "duel", RawName: "known-boss"}})
	mapping("replacement", "new-boss")
	snapshot("duel", "boss")
	occupant, _ := s.ParticipantBySubCircle(0)
	if occupant.ID != "replacement" {
		t.Fatal("revived identity stole occupied slot")
	}
	snapshot("new-duel", "boss")
	for _, p := range s.Snapshot().Combat.Participants {
		if p.ID == "boss" && (p.MobID != "" || p.Kind != events.ParticipantUnknown || p.SubCircle != -1) {
			t.Fatal("identity leaked across duels")
		}
	}
}
