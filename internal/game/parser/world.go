package parser

import (
	"regexp"
	"strings"
	"time"

	"corvin101/internal/game/events"
)

func (p *Parser) parseCombatJoined(
	line string,
	timestamp time.Time,
) (events.Event, bool) {
	if !strings.Contains(
		line,
		"My player was added to a combat.",
	) {
		return events.Event{}, false
	}

	return events.Event{
		Type:      events.CombatJoined,
		Timestamp: timestamp,
		Raw:       line,

		Combat: &events.CombatEvent{
			ClientDuelID: p.currentClientDuelID,
		},
	}, true
}

func (p *Parser) parseCombatFled(
	line string,
	timestamp time.Time,
) (events.Event, bool) {
	if !strings.Contains(
		line,
		"MSG_CombatFlee()",
	) {
		return events.Event{}, false
	}

	// This message is also logged when another player flees. Wait for the
	// following CombatRemove to identify who actually left the duel.
	p.pendingFleeDuel = p.currentClientDuelID
	p.pendingFleeAt = timestamp
	return events.Event{}, true
}

func (p *Parser) parseCombatLeft(
	line string,
	timestamp time.Time,
) (events.Event, bool) {
	if !strings.Contains(
		line,
		"Ending Phase received, destroying combat",
	) {
		return events.Event{}, false
	}

	return events.Event{
		Type:      events.CombatLeft,
		Timestamp: timestamp,
		Raw:       line,

		Combat: &events.CombatEvent{
			ClientDuelID: p.currentClientDuelID,
		},
	}, true
}

func (p *Parser) parseCharacterSelected(
	line string,
	timestamp time.Time,
) (events.Event, bool) {
	if !strings.Contains(
		line,
		"Character selection is now enabled.",
	) {
		return events.Event{}, false
	}

	return events.Event{
		Type:      events.CharacterSelected,
		Timestamp: timestamp,
		Raw:       line,
	}, true
}

func (p *Parser) parseZoneLoaded(
	line string,
	timestamp time.Time,
) (events.Event, bool) {
	zoneKey := ""
	if strings.Contains(line, "Finished loading zone") {
		zoneKey = extractZone(line)
	}
	if zoneKey == "" {
		if match := completedZoneRe.FindStringSubmatch(line); match != nil {
			zoneKey = match[1]
			if zoneKey == "" {
				zoneKey = match[2]
			}
		}
	}
	if zoneKey == "" {
		return events.Event{}, false
	}
	p.lastZoneKey = zoneKey

	return events.Event{
		Type:      events.ZoneLoaded,
		Timestamp: timestamp,
		Raw:       line,

		Zone: &events.ZoneEvent{
			ZoneKey: zoneKey,
		},
	}, true
}

var completedZoneRe = regexp.MustCompile(`(?:GameClient\s+Zone ([A-Za-z0-9_/.-]+) loaded in|SpawnManager::OnPostLoad ZoneLevel:\d+ Name:([A-Za-z0-9_/.-]+) Path:)`)
var resourceZoneRe = regexp.MustCompile(`SpecificAssetRequest: Failed to find resource .*?, world: ([A-Za-z0-9_]+), zone: ([A-Za-z0-9_/.-]+), hint: \d+`)

// Scoped resource requests contain the zone used by the current client even
// when its normal loading-completed message is missing. Asset repositories
// such as WorldData/HighDetail and quest pre-cache targets are not locations.
func (p *Parser) parseResourceZone(line string, at time.Time) (events.Event, bool) {
	if p.localDuelID == "" || p.localDuelEnded || p.foreignCombat {
		return events.Event{}, false
	}
	match := resourceZoneRe.FindStringSubmatch(line)
	if match == nil {
		return events.Event{}, false
	}
	world, zone := match[1], match[2]
	if !strings.HasPrefix(zone, "Interiors/") && !strings.HasPrefix(zone, world+"_") {
		return events.Event{}, false
	}
	key := world + "/" + zone
	if key == p.lastZoneKey {
		return events.Event{}, false
	}
	p.lastZoneKey = key
	return events.Event{Type: events.ZoneLoaded, Timestamp: at, Raw: line, Zone: &events.ZoneEvent{ZoneKey: key}}, true
}

func extractZone(line string) string {
	start := strings.IndexByte(line, '\'')
	end := strings.LastIndexByte(line, '\'')

	if start == -1 || end <= start {
		return ""
	}

	return strings.TrimSpace(
		line[start+1 : end],
	)
}
