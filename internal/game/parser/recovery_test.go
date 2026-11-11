package parser_test

import (
	"corvin101/internal/game/events"
	"corvin101/internal/game/parser"
	"corvin101/internal/game/state"
	"testing"
)

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
