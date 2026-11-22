package parser_test

import (
	"corvin101/internal/game/events"
	"corvin101/internal/game/parser"
	"corvin101/internal/game/state"
	"fmt"
	"testing"
)

func TestMixedLateJoinRecoversExplicitActorIdentities(t *testing.T) {
	p, s := parser.New(), state.New()
	feed := func(line string) {
		for _, ev := range p.Feed("11/14/26 13:16:12 " + line) {
			s.HandleEvent(ev)
		}
	}
	feed("Duel 1615861: MSG_CombatAdd(Local Player)")
	feed("My SubCircle: 5, OtherPart's SubCircle: 0")
	feed("Added participant 12593827 to duel 1615861")
	// These explicit names arrive before the complete stats/snapshot pair.
	for _, actor := range []struct{ global, name string }{
		{"2889340635936499830", "DM_Vampire_BaseMasked_A_02"},
		{"2889340635936499833", "DM_Vampire_BaseMasked_A_01"},
		{"2889340635936499827", "DM_Vampire_BaseMasked_A_03"},
	} {
		feed(fmt.Sprintf("AnimationBehavior OwnerID:%s [templateID:2199024954439  Debug Name:%s]", actor.global, actor.name))
		feed("Duel 1615861: MSG_CombatStats(" + actor.name + ")")
	}
	feed("Duel 1615861: MSG_CombatStats(Local Player)")
	feed("Duel 1615861: MSG_CombatPips()")
	feed("---BEGIN DEBUGDUMPDUEL---")
	feed("Participant: ID 12593827, Team 0, Health 10017 Pips 1 PPips 1")
	for _, id := range []string{"1615990", "1615993", "1615987"} {
		feed("Participant: ID " + id + ", Team 1, Health 6050 Pips 2 PPips 0")
	}
	feed("---END DEBUGDUMPDUEL---")
	feed("CombatResolver::ResolveCombatRound. Duel ID:2889340635936499701 Round:2 Caster:5 Spell:Fire Dragon")
	want := map[string]string{"1615990": "DM_Vampire_BaseMasked_A_02", "1615993": "DM_Vampire_BaseMasked_A_01", "1615987": "DM_Vampire_BaseMasked_A_03"}
	for _, participant := range s.Snapshot().Combat.Participants {
		if name, ok := want[participant.ID]; ok && (participant.Kind != events.ParticipantMob || participant.MobID != name) {
			t.Fatalf("incorrect explicit identity: %+v", participant)
		}
	}
}

func TestLateJoinStatsAndCastRecoverEnemyWithoutOriginalAdd(t *testing.T) {
	p, s := parser.New(), state.New()
	for _, line := range []string{
		"11/11/26 16:42:19 Duel 15824845: MSG_CombatStats(DM_Card_SpellCardMystical_E_01)",
		"11/11/26 16:42:19 Duel 15824845: MSG_CombatStats(Non-local Player)",
		"11/11/26 16:42:19 Duel 15824845: MSG_CombatPips()",
		"11/11/26 16:42:21 Duel 15824845: MSG_CombatAdd(Local Player)",
		"11/11/26 16:42:21 My SubCircle: 5, OtherPart's SubCircle: 0",
		"11/11/26 16:42:21 Added participant 12635547 to duel 15824845",
		"11/11/26 16:42:29 ---BEGIN DEBUGDUMPDUEL---",
		"11/11/26 16:42:29 Participant: ID 15824874, Team 1, Health 16115 Pips 2 PPips 4",
		"11/11/26 16:42:29 Participant: ID 12593827, Team 0, Health 10017 Pips 1 PPips 1",
		"11/11/26 16:42:29 Participant: ID 12635547, Team 0, Health 10632 Pips 1 PPips 1",
		"11/11/26 16:42:29 ---END DEBUGDUMPDUEL---",
		"11/11/26 16:42:29 {IsSim:0} Participant (sc 0) cast spell 80% HP - AoE - Mob",
		"11/11/26 16:42:29 {IsSim:0} Processing spell pips - ID(15824874) rank(0)",
	} {
		for _, ev := range p.Feed(line) {
			s.HandleEvent(ev)
		}
	}
	boss, ok := s.ParticipantBySubCircle(0)
	if !ok || boss.ID != "15824874" || boss.Kind != events.ParticipantMob || boss.MobID != "DM_Card_SpellCardMystical_E_01" {
		t.Fatalf("late join: %+v", boss)
	}
	for _, participant := range s.Snapshot().Combat.Participants {
		if participant.Kind == events.ParticipantUnknown {
			t.Fatalf("unknown identity: %+v", participant)
		}
	}
}

func TestSimulatedAndUnpairedPipsDoNotMapParticipants(t *testing.T) {
	for _, lines := range [][]string{
		{"{IsSim:1} Participant (sc 0) cast spell X", "{IsSim:1} Processing spell pips - ID(999)"},
		{"{IsSim:0} Participant (sc 0) cast spell X", "{IsSim:1} simulation", "{IsSim:0} Processing spell pips - ID(999)"},
		{"{IsSim:0} Participant (sc 0) cast spell X", "CombatResolver::ResolveCombatRound.", "{IsSim:0} Processing spell pips - ID(999)"},
		{"{IsSim:0} Processing spell pips - ID(999)"},
	} {
		p := parser.New()
		p.Feed("11/11/26 16:42:21 Duel 1: MSG_CombatPhase(kPhase_Execution) PlayerID:0")
		for _, line := range lines {
			for _, ev := range p.Feed("11/11/26 16:42:29 " + line) {
				if ev.Type == events.ParticipantMapped {
					t.Fatal("unsafe position mapping")
				}
			}
		}
	}
}
