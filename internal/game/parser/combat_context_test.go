package parser_test

import (
	"corvin101/internal/game/parser"
	"corvin101/internal/game/state"
	"testing"
)

func TestNearbyDuelCannotReplaceOrReactivateLocalCombat(t *testing.T) {
	p, s := parser.New(), state.New()
	feed := func(lines ...string) {
		for _, line := range lines {
			for _, ev := range p.Feed("11/15/26 20:15:18 " + line) {
				s.HandleEvent(ev)
			}
		}
	}
	join := func(id string) {
		feed("Duel "+id+": MSG_CombatAdd(Local Player)", "My SubCircle: 4, OtherPart's SubCircle: 0", "Added participant 11858739 to duel "+id, "My player was added to a combat.")
	}
	join("2497627")
	feed("CombatPlanningPhaseWindow::HandleEnterCombat() combatGid:2889059160960670811", "Duel 2497627: MSG_CombatPhase(kPhase_Execution)")
	foreign := func() {
		feed(
			"Duel 2497624: MSG_CombatAdd(AH_Octopus_Alien_A_01)",
			"My SubCircle: 0, OtherPart's SubCircle: 4",
			"Added participant 3053548 to duel 2497624",
			"Duel 2497624: MSG_CombatActions()",
			"---BEGIN DEBUGDUMPDUEL---",
			"Participant: ID 3053548, Team 1, Health 3465 Pips 1 PPips 1",
			"---END DEBUGDUMPDUEL---",
			"CombatResolver::ResolveCombatRound. Duel ID:2889059160960670808 Round:2 Caster:0 Spell:Wild Bolt",
			"EffectTargetPairList::ProcessTargets. Caster:0 Target:4 Effect:kDamage Param:3497")
	}
	foreign()
	if got := s.Snapshot(); got.Combat.ClientDuelID != "2497627" || !got.Combat.Active || got.Combat.LastSpell != nil || len(got.Combat.RecentEffects) != 0 {
		t.Fatalf("nearby duel replaced local combat: %+v", got.Combat)
	}
	feed("Duel 2497627: MSG_CombatPhase(kPhase_Planning)", "CombatResolver::ResolveCombatRound. Duel ID:2889059160960670811 Round:2 Caster:4 Spell:Fireblade")
	if s.Snapshot().Combat.LastSpell == nil {
		t.Fatal("own combat failed to resume after foreign block")
	}
	feed("Duel 2497627: MSG_CombatPhase(kPhase_Ended)")
	foreign()
	if got := s.Snapshot(); got.Combat.Active || got.Game.Mode != state.ModeRoaming || len(got.Combat.Participants) != 0 {
		t.Fatalf("foreign fight reactivated tracker: %+v", got)
	}
	join("2497700")
	if got := s.Snapshot(); !got.Combat.Active || got.Game.Mode != state.ModeBattle || got.Combat.ClientDuelID != "2497700" {
		t.Fatal("next local duel was blocked")
	}
}
