package parser

import (
	"corvin101/internal/game/events"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var statsNameRe = regexp.MustCompile(`Duel (\d+): MSG_CombatStats\(([^)]+)\)`)
var statsEndRe = regexp.MustCompile(`Duel (\d+): MSG_CombatPips\(\)`)
var castPositionRe = regexp.MustCompile(`\{IsSim:0\} Participant \(sc (\d+)\) cast spell `)
var castPipsRe = regexp.MustCompile(`\{IsSim:0\} Processing spell pips - ID\((\d+)\)`)
var actorIdentityRe = regexp.MustCompile(`OwnerID:(\d+)\s+\[templateID:\d+\s+Debug Name:([^\]\s]+)\]`)

func (p *Parser) resetRecovery() {
	p.statsDuel = ""
	p.statsNames = nil
	p.statsAt = time.Time{}
	p.castPosition = nil
	p.castPositionDuel = ""
	p.castPositionAt = time.Time{}
}

func (p *Parser) parseRecovery(line string, at time.Time) ([]events.Event, bool) {
	if match := actorIdentityRe.FindStringSubmatch(line); match != nil && p.currentClientDuelID != "" {
		id, err := strconv.ParseUint(match[1], 10, 64)
		if err == nil && id>>32 != 0 && uint32(id) != 0 {
			return []events.Event{{Type: events.ParticipantMapped, Timestamp: at, Raw: line, ParticipantMapping: &events.ParticipantMappingEvent{ClientDuelID: p.currentClientDuelID, GlobalID: match[1], ParticipantID: strconv.FormatUint(uint64(uint32(id)), 10), SubCircle: -1, Kind: events.ParticipantMob, MobID: match[2]}}}, true
		}
	}
	if match := statsNameRe.FindStringSubmatch(line); match != nil {
		if p.statsDuel != match[1] || at.Sub(p.statsAt) > 5*time.Second || at.Before(p.statsAt) {
			p.statsNames = nil
		}
		p.statsDuel = match[1]
		p.statsAt = at
		p.statsNames = append(p.statsNames, strings.TrimSpace(match[2]))
		return nil, true
	}
	if match := statsEndRe.FindStringSubmatch(line); match != nil {
		names := p.statsNames
		duel := p.statsDuel
		p.statsNames = nil
		p.statsDuel = ""
		if duel == match[1] && len(names) > 0 && !at.Before(p.statsAt) && at.Sub(p.statsAt) <= 5*time.Second {
			return []events.Event{{Type: events.DuelRosterReceived, Timestamp: at, Raw: line, Roster: &events.DuelRoster{ClientDuelID: duel, Names: names}}}, true
		}
		return nil, true
	}
	if match := castPositionRe.FindStringSubmatch(line); match != nil {
		position, _ := strconv.Atoi(match[1])
		p.castPosition = &position
		p.castPositionDuel = p.currentClientDuelID
		p.castPositionAt = at
		return nil, true
	}
	if match := castPipsRe.FindStringSubmatch(line); match != nil {
		position := p.castPosition
		p.castPosition = nil
		if position != nil && p.castPositionDuel != "" && p.castPositionDuel == p.currentClientDuelID && !at.Before(p.castPositionAt) && at.Sub(p.castPositionAt) <= time.Second {
			return []events.Event{{Type: events.ParticipantMapped, Timestamp: at, Raw: line, ParticipantMapping: &events.ParticipantMappingEvent{ClientDuelID: p.currentClientDuelID, ParticipantID: match[1], SubCircle: *position}}}, true
		}
		return nil, true
	}
	// Do not let a caster observation span another spell, phase or simulation.
	if strings.Contains(line, "ResolveCombatRound.") || strings.Contains(line, "MSG_CombatPhase(") || strings.Contains(line, "{IsSim:1}") || strings.Contains(line, "DEBUGDUMPDUEL") {
		p.castPosition = nil
	}
	return nil, false
}
