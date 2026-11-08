package parser_test

import (
	"corvin101/internal/game/events"
	"corvin101/internal/game/parser"
	"corvin101/internal/game/state"
	"testing"
)

func TestOtherPlayerFleeKeepsKnownOpponent(t *testing.T) {
	p, s := parser.New(), state.New()
	feed := func(line string) {
		for _, event := range p.Feed(line) {
			s.HandleEvent(event)
		}
	}
	for _, line := range []string{
		"11/08/26 15:52:40 Duel 12721508: MSG_CombatAdd(G14-DM-Malistaire_Boss)",
		"11/08/26 15:52:40 Added participant 12721527 to duel 12721508",
		"11/08/26 15:52:40 Duel 12721508: MSG_CombatAdd(Local Player)",
		"11/08/26 15:52:40 My SubCircle: 4, OtherPart's SubCircle: 0",
		"11/08/26 15:52:40 Added participant 12635547 to duel 12721508",
		"11/08/26 15:52:40 My player was added to a combat.",
		"11/08/26 15:52:42 Duel 12721508: MSG_CombatPhase(kPhase_Planning) PlayerID:0",
		"11/08/26 15:57:56 MSG_CombatFlee()",
		"11/08/26 15:57:57 Duel 12721508: MSG_CombatRemove(Non-local Player)",
		"11/08/26 16:01:46 ---BEGIN DEBUGDUMPDUEL---",
		"11/08/26 16:01:46 Participant: ID 12721527, Team 1, Health 8091 Pips 0 PPips 3",
		"11/08/26 16:01:46 ---END DEBUGDUMPDUEL---",
	} {
		feed(line)
	}
	snapshot := s.Snapshot()
	if !snapshot.Combat.Active || snapshot.Game.Mode != state.ModeBattle {
		t.Fatalf("other player's flight reset combat: %+v", snapshot)
	}
	boss, found := s.ParticipantBySubCircle(0)
	if !found || boss.Kind != events.ParticipantMob || boss.MobID != "G14-DM-Malistaire_Boss" {
		t.Fatalf("boss identity lost: %+v", boss)
	}
	feed("11/08/26 16:02:00 MSG_CombatFlee()")
	feed("11/08/26 16:02:01 Duel 12721508: MSG_CombatRemove(Local Player)")
	if s.Snapshot().Combat.Active {
		t.Fatal("confirmed local flight did not end combat")
	}
}
