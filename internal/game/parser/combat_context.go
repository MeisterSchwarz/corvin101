package parser

import (
	"regexp"
	"strings"
)

var clientDuelContextRe = regexp.MustCompile(`(?:Duel (\d+):|\bduel (\d+)\b)`)
var localResolverContextRe = regexp.MustCompile(`HandleEnterCombat\(\) combatGid:(\d+)`)

// Nearby duels also log adds, numeric dumps, spells and effects. Bind to the
// explicit local-player add and preserve that ownership after its duel ends.
func (p *Parser) acceptCombatContext(line string) bool {
	if match := combatAddRe.FindStringSubmatch(line); match != nil && strings.TrimSpace(match[2]) == "Local Player" {
		if p.currentClientDuelID != match[1] || p.localDuelEnded {
			p.pendingParticipant = nil
			p.unmappedParticipants = nil
			p.duelDump = nil
			p.resetRecovery()
		}
		p.localDuelID, p.currentClientDuelID = match[1], match[1]
		p.localResolverID = ""
		p.localDuelEnded, p.foreignCombat = false, false
		return true
	}
	// Keep pre-join recovery available: some clients log the roster first.
	if p.localDuelID == "" {
		return true
	}
	if match := clientDuelContextRe.FindStringSubmatch(line); match != nil {
		id := match[1]
		if id == "" {
			id = match[2]
		}
		p.foreignCombat = id != p.localDuelID
	}
	if match := resolveCombatRoundRe.FindStringSubmatch(line); match != nil {
		if p.localResolverID != "" {
			p.foreignCombat = match[1] != p.localResolverID
		} else if !p.foreignCombat && !p.localDuelEnded {
			p.localResolverID = match[1]
		}
	}
	if p.foreignCombat || p.localDuelEnded {
		return false
	}
	if match := localResolverContextRe.FindStringSubmatch(line); match != nil {
		p.localResolverID = match[1]
	}
	return true
}
