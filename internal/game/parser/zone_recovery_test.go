package parser_test

import (
	"corvin101/internal/game/events"
	"corvin101/internal/game/parser"
	"corvin101/internal/game/state"
	"testing"
)

func TestZoneRecoveryFromScopedResourceRequest(t *testing.T) {
	p, s := parser.New(), state.New()
	feed := func(line string) []events.Event {
		ev := p.Feed("11/21/26 15:52:05 " + line)
		for _, event := range ev {
			s.HandleEvent(event)
		}
		return ev
	}
	feed("GameClient::LoadZone() - Finished loading zone 'Ahmarra/AH_Z06_ForbiddenCity'.")
	feed("Duel 7106443: MSG_CombatAdd(Local Player)")
	feed("My SubCircle: 4, OtherPart's SubCircle: 0")
	feed("Added participant 11858739 to duel 7106443")
	feed("My player was added to a combat.")
	resource := "SpecificAssetRequest: Failed to find resource Combat/Cine_Blizzard/AOE_BlizzardCam_Summon_1.nif, world: Ahmarra, zone: Interiors/AH_Z09I06_NightmareShrine, hint: 5"
	feed(resource)
	if got := s.Snapshot(); got.Game.ZoneKey != "Ahmarra/Interiors/AH_Z09I06_NightmareShrine" || !got.Combat.Active {
		t.Fatalf("zone recovery changed combat or failed: %+v", got)
	}
	if len(feed(resource)) != 0 {
		t.Fatal("duplicate location emitted")
	}
	feed("SpecificAssetRequest: Failed to find resource X, world: Equipment, zone: HighDetail, hint: 5")
	feed("MOB PRE_CACHE >>> checking goal zone Ahmarra/Interiors/Unrelated")
	if s.Snapshot().Game.ZoneKey != "Ahmarra/Interiors/AH_Z09I06_NightmareShrine" {
		t.Fatal("asset or quest target became current zone")
	}
	feed("Duel 999: MSG_CombatActions()")
	feed("SpecificAssetRequest: Failed to find resource X, world: Ahmarra, zone: Interiors/ForeignDuel, hint: 5")
	if s.Snapshot().Game.ZoneKey != "Ahmarra/Interiors/AH_Z09I06_NightmareShrine" {
		t.Fatal("foreign combat changed location")
	}
}

func TestAlternativeZoneLoadConfirmations(t *testing.T) {
	for _, line := range []string{"GameClient      Zone Ahmarra/Interiors/Test loaded in 0.391000 seconds.", "SpawnManager::OnPostLoad ZoneLevel:0 Name:Ahmarra/Interiors/Test Path:ZoneData/Ahmarra/Interiors/Test/"} {
		p := parser.New()
		ev := p.Feed(line)
		if len(ev) != 1 || ev[0].Zone == nil || ev[0].Zone.ZoneKey != "Ahmarra/Interiors/Test" {
			t.Fatalf("load confirmation ignored: %s", line)
		}
	}
}
