package state

import (
	"corvin101/internal/game/events"
	"testing"
)

func TestActorIdentityRejectsUnrelatedEvidence(t *testing.T) {
	for _, test := range []struct {
		name, global, id, mob, duel string
		team                        int
	}{
		{"valid", "2889340635936499830", "1615990", "enemy", "duel", 1},
		{"namespace", "4296583286", "1615990", "enemy", "duel", 1},
		{"wrong-id", "2889340635936499833", "1615990", "enemy", "duel", 1},
		{"unlisted-template", "2889340635936499830", "1615990", "unlisted", "duel", 1},
		{"different-duel", "2889340635936499830", "1615990", "enemy", "other", 1},
		{"ally", "2889340635936499830", "1615990", "enemy", "duel", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := New()
			s.combat.ClientDuelID = "duel"
			s.combat.ResolverDuelID = "2889340635936499701"
			s.combat.Participants["local"] = &ParticipantState{ID: "local", Kind: events.ParticipantLocalPlayer, Team: 0}
			s.combat.Participants["1615990"] = &ParticipantState{ID: "1615990", Team: test.team, SubCircle: -1}
			s.combat.Participants["other"] = &ParticipantState{ID: "other", Team: 1, SubCircle: -1}
			s.combat.snapshotIDs = []string{"local", "1615990", "other"}
			s.combat.rosterNames = []string{"enemy", "different-enemy"}
			s.HandleEvent(events.Event{Type: events.ParticipantMapped, ParticipantMapping: &events.ParticipantMappingEvent{ClientDuelID: test.duel, GlobalID: test.global, ParticipantID: test.id, MobID: test.mob, SubCircle: -1, Kind: events.ParticipantMob}})
			p := s.combat.Participants["1615990"]
			if (p.Kind == events.ParticipantMob) != (test.name == "valid") {
				t.Fatalf("unexpected identity: %+v", p)
			}
		})
	}
}
