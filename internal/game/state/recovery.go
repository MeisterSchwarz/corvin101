package state

import "corvin101/internal/game/events"

// Only resolve a complete roster against a complete numeric snapshot. Names
// are never assigned by list order, health, appearance or timing proximity.
func (s *Store) recoverRosterIdentities() {
	if len(s.combat.rosterNames) == 0 || len(s.combat.snapshotIDs) == 0 || s.combat.snapshotAt.Before(s.combat.rosterAt) {
		return
	}
	names := make(map[string]int)
	for _, name := range s.combat.rosterNames {
		names[name]++
	}
	var local *ParticipantState
	for _, id := range s.combat.snapshotIDs {
		p := s.combat.Participants[id]
		if p != nil && p.Kind == events.ParticipantLocalPlayer {
			if local != nil {
				return
			}
			local = p
		}
	}
	if local == nil {
		return
	}
	// Some clients include the local player's stats, others omit them.
	if names["Local Player"] > 1 {
		return
	}
	delete(names, "Local Player")
	var opponents, allies []*ParticipantState
	for _, id := range s.combat.snapshotIDs {
		p := s.combat.Participants[id]
		if p == nil {
			return
		}
		if p.Kind == events.ParticipantLocalPlayer {
			continue
		}
		name := ""
		if p.Kind == events.ParticipantMob {
			name = p.MobID
		}
		if p.Kind == events.ParticipantOtherPlayer {
			name = "Non-local Player"
		}
		if name != "" {
			if names[name] <= 0 {
				return
			}
			names[name]--
			continue
		}
		if p.Team == local.Team {
			allies = append(allies, p)
		} else {
			opponents = append(opponents, p)
		}
	}
	if names["Non-local Player"] != len(allies) {
		return
	}
	delete(names, "Non-local Player")
	total, unique := 0, ""
	distinct := 0
	for name, count := range names {
		if count > 0 {
			total += count
			unique = name
			distinct++
		}
	}
	if total != len(opponents) {
		return
	}
	// Mixed unknown enemies remain unresolved until direct mappings eliminate
	// ambiguity; identical templates may safely share the same MobID.
	for _, p := range allies {
		p.Kind = events.ParticipantOtherPlayer
	}
	if distinct == 1 {
		for _, p := range opponents {
			p.Kind = events.ParticipantMob
			p.MobID = unique
		}
	}
}
