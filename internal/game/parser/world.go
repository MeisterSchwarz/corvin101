package parser

import (
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

	return events.Event{
		Type:      events.CombatFled,
		Timestamp: timestamp,
		Raw:       line,

		Combat: &events.CombatEvent{
			ClientDuelID: p.currentClientDuelID,
		},
	}, true
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
	if !strings.Contains(
		line,
		"Finished loading zone",
	) {
		return events.Event{}, false
	}

	zoneKey := extractZone(line)
	if zoneKey == "" {
		return events.Event{}, false
	}

	return events.Event{
		Type:      events.ZoneLoaded,
		Timestamp: timestamp,
		Raw:       line,

		Zone: &events.ZoneEvent{
			ZoneKey: zoneKey,
		},
	}, true
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
