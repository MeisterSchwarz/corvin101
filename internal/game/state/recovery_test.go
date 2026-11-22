package state

import (
	"corvin101/internal/game/events"
	"testing"
)

func TestFleeRejoinRetainsOnlyIdentityInSameDuel(t *testing.T) {
	for _, duel := range []string{"1497718", "another-duel"} {
		t.Run(duel, func(t *testing.T) {
			s := New()
			s.HandleEvent(events.Event{Type: events.ParticipantMapped, ParticipantMapping: &events.ParticipantMappingEvent{ClientDuelID: "1497718", ParticipantID: "1497734", SubCircle: 0, Kind: events.ParticipantMob, MobID: "DM_Player_BranchBoss_A_05"}})
			p := s.combat.Participants["1497734"]
			p.Health = 16080
			p.Effects.Hanging = append(p.Effects.Hanging, ActiveEffect{RawEffect: "old-effect"})
			s.HandleEvent(events.Event{Type: events.CombatFled})
			if len(s.combat.Participants) != 0 || s.combat.Active {
				t.Fatal("flee left combat active")
			}
			s.HandleEvent(events.Event{Type: events.DuelSnapshotReceived, Snapshot: &events.DuelSnapshot{ClientDuelID: duel, Participants: []events.Participant{{ID: "1497734", SubCircle: -1, Team: 1, Health: 2944}, {ID: "new-enemy", SubCircle: -1, Team: 1}}}})
			p = s.combat.Participants["1497734"]
			if duel == "1497718" {
				if p.Kind != events.ParticipantMob || p.MobID != "DM_Player_BranchBoss_A_05" {
					t.Fatalf("identity lost: %+v", p)
				}
			} else if p.Kind != events.ParticipantUnknown || p.MobID != "" {
				t.Fatal("identity crossed duels")
			}
			if p.Health != 2944 || len(p.Effects.Hanging) != 0 {
				t.Fatal("stale combat values restored")
			}
			if duel == "1497718" && p.SubCircle != 0 {
				t.Fatal("known enemy slot lost")
			}
			if duel != "1497718" && p.SubCircle != -1 {
				t.Fatal("slot crossed duels")
			}
			if s.combat.Participants["new-enemy"].Kind != events.ParticipantUnknown {
				t.Fatal("identity guessed for different ID")
			}
		})
	}
}

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
