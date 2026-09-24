package state

import (
	"time"

	"corvin101/internal/game/events"
)

type CombatState struct {
	Active bool

	ClientDuelID   string
	ResolverDuelID string

	Round int
	Phase events.CombatPhase

	StartedAt time.Time

	// Participant ID -> Participant
	Participants map[string]*ParticipantState

	// SubCircle -> Participant ID
	ParticipantBySubCircle map[int]string

	LastSpell *SpellState

	// Nur Event-Historie / Debugging.
	RecentEffects []EffectState
}

type ParticipantState struct {
	ID string

	// -1 = unbekannt.
	SubCircle int

	Kind  events.ParticipantKind
	MobID string

	Team   int
	Health int
	Pips   int
	PPips  int

	Effects ParticipantEffects
}

type ParticipantEffects struct {
	// Persistente Effekte aus AddHangingEffect.
	Hanging []ActiveEffect

	// Aura-Effekte aus AddAuraEffect.
	Auras []ActiveEffect
}

type ActiveEffect struct {
	RawEffect string
	Param     int
	Charm     *bool
	AddedAt   time.Time
}

type SpellState struct {
	Round     int
	Caster    int
	Name      string
	Timestamp time.Time
}

type EffectState struct {
	Type events.Type

	Caster int
	Target int

	RawEffect string
	Param     int
	Charm     *bool

	Timestamp time.Time
}

func newCombatState() CombatState {
	return CombatState{
		Participants:           make(map[string]*ParticipantState),
		ParticipantBySubCircle: make(map[int]string),
		RecentEffects:          make([]EffectState, 0, 32),
	}
}

func newParticipantState(id string) *ParticipantState {
	return &ParticipantState{
		ID:        id,
		SubCircle: -1,
		Effects: ParticipantEffects{
			Hanging: make([]ActiveEffect, 0, 8),
			Auras:   make([]ActiveEffect, 0, 4),
		},
	}
}

func (s *Store) handleCombatEvent(event events.Event) {
	switch event.Type {
	case events.CombatJoined:
		s.handleCombatJoined(event)

	case events.CombatPhaseChanged:
		s.handleCombatPhaseChanged(event)

	case events.CombatVictory:
		s.handleCombatVictory(event)

	case events.SpellCast:
		s.handleSpellCast(event)

	case events.EffectProcessed:
		s.handleEffectProcessed(event)

	case events.HangingEffectAdded:
		s.handleHangingEffectAdded(event)

	case events.HangingEffectRemoved:
		s.handleHangingEffectRemoved(event)

	case events.AuraEffectAdded:
		s.handleAuraEffectAdded(event)

	case events.ParticipantAdded:
		s.handleParticipantAdded(event)

	case events.ParticipantMapped:
		s.handleParticipantMapped(event)

	case events.DuelSnapshotReceived:
		if event.Snapshot != nil {
			s.applyDuelSnapshot(*event.Snapshot)
		}

	case events.CombatFled, events.CombatLeft:
		s.resetCombat()
	}
}

func (s *Store) handleCombatJoined(event events.Event) {
	s.combat.Active = true

	if s.combat.StartedAt.IsZero() {
		s.combat.StartedAt = time.Now()
	}

	if event.Combat != nil && event.Combat.ClientDuelID != "" {
		s.combat.ClientDuelID = event.Combat.ClientDuelID
	}
}

func (s *Store) handleCombatPhaseChanged(event events.Event) {
	if event.Combat == nil {
		return
	}

	previousPhase := s.combat.Phase
	nextPhase := event.Combat.Phase

	s.combat.Active = true
	s.combat.Phase = nextPhase

	if nextPhase == events.PhasePlanning &&
		previousPhase != events.PhasePlanning {

		s.combat.Round++
	}

	if event.Combat.ClientDuelID != "" {
		s.combat.ClientDuelID = event.Combat.ClientDuelID
	}
}

func (s *Store) handleCombatVictory(event events.Event) {
	if event.Combat == nil {
		return
	}

	s.combat.Active = true
	s.combat.Phase = events.PhaseVictory

	if event.Combat.ClientDuelID != "" {
		s.combat.ClientDuelID = event.Combat.ClientDuelID
	}
}

func (s *Store) handleSpellCast(event events.Event) {
	if event.Spell == nil {
		return
	}

	s.combat.Active = true
	s.combat.Round = event.Spell.Round
	s.combat.ResolverDuelID = event.Spell.ResolverDuelID

	s.combat.LastSpell = &SpellState{
		Round:     event.Spell.Round,
		Caster:    event.Spell.Caster,
		Name:      event.Spell.SpellName,
		Timestamp: event.Timestamp,
	}
}

func (s *Store) handleEffectProcessed(event events.Event) {
	if event.Effect == nil {
		return
	}

	// ProcessTargets ist ein ausgeführter Effekt.
	// Er bedeutet nicht automatisch, dass danach ein
	// persistenter Buff/Debuff aktiv ist.
	s.appendRecentEffect(event)
}

func (s *Store) handleHangingEffectAdded(event events.Event) {
	if event.Effect == nil {
		return
	}

	s.appendRecentEffect(event)

	participant := s.participantBySubCircleUnsafe(event.Effect.Target)
	if participant == nil {
		return
	}

	participant.Effects.Hanging = append(
		participant.Effects.Hanging,
		ActiveEffect{
			RawEffect: event.Effect.RawEffect,
			Param:     event.Effect.Param,
			Charm:     cloneBool(event.Effect.Charm),
			AddedAt:   event.Timestamp,
		},
	)
}

func (s *Store) handleHangingEffectRemoved(event events.Event) {
	if event.Effect == nil {
		return
	}

	s.appendRecentEffect(event)

	participant := s.participantBySubCircleUnsafe(event.Effect.Target)
	if participant == nil {
		return
	}

	participant.Effects.Hanging = removeMatchingHangingEffect(
		participant.Effects.Hanging,
		event.Effect,
	)
}

func (s *Store) handleAuraEffectAdded(event events.Event) {
	if event.Effect == nil {
		return
	}

	s.appendRecentEffect(event)

	participant := s.participantBySubCircleUnsafe(event.Effect.Target)
	if participant == nil {
		return
	}

	participant.Effects.Auras = append(
		participant.Effects.Auras,
		ActiveEffect{
			RawEffect: event.Effect.RawEffect,
			Param:     event.Effect.Param,
			Charm:     cloneBool(event.Effect.Charm),
			AddedAt:   event.Timestamp,
		},
	)
}

func (s *Store) handleParticipantAdded(event events.Event) {
	if event.Participant == nil {
		return
	}

	incoming := event.Participant.Participant

	if incoming.ID == "" {
		return
	}

	participant, ok := s.combat.Participants[incoming.ID]
	if !ok {
		participant = newParticipantState(incoming.ID)
		s.combat.Participants[incoming.ID] = participant
	}

	if incoming.SubCircle >= 0 {
		s.assignSubCircle(participant, incoming.SubCircle)
	}

	if incoming.Kind != events.ParticipantUnknown {
		participant.Kind = incoming.Kind
	}

	if incoming.MobID != "" {
		participant.MobID = incoming.MobID
	}

	if event.Participant.ClientDuelID != "" {
		s.combat.ClientDuelID = event.Participant.ClientDuelID
	}
}

func (s *Store) handleParticipantMapped(event events.Event) {
	if event.ParticipantMapping == nil {
		return
	}

	mapping := event.ParticipantMapping

	if mapping.ParticipantID == "" || mapping.SubCircle < 0 {
		return
	}

	participant, ok := s.combat.Participants[mapping.ParticipantID]
	if !ok {
		participant = newParticipantState(mapping.ParticipantID)
		s.combat.Participants[mapping.ParticipantID] = participant
	}

	s.assignSubCircle(participant, mapping.SubCircle)

	if mapping.Kind != events.ParticipantUnknown {
		participant.Kind = mapping.Kind
	}

	if mapping.MobID != "" {
		participant.MobID = mapping.MobID
	}

	if mapping.ClientDuelID != "" {
		s.combat.ClientDuelID = mapping.ClientDuelID
	}
}

func (s *Store) assignSubCircle(
	participant *ParticipantState,
	subCircle int,
) {
	if participant == nil || subCircle < 0 {
		return
	}

	// Falls der Participant vorher einer anderen SC
	// zugeordnet war, alten Index entfernen.
	if participant.SubCircle >= 0 && participant.SubCircle != subCircle {
		delete(
			s.combat.ParticipantBySubCircle,
			participant.SubCircle,
		)
	}

	participant.SubCircle = subCircle

	s.combat.ParticipantBySubCircle[subCircle] = participant.ID
}

func (s *Store) appendRecentEffect(event events.Event) {
	if event.Effect == nil {
		return
	}

	s.combat.RecentEffects = append(
		s.combat.RecentEffects,
		EffectState{
			Type:      event.Type,
			Caster:    event.Effect.Caster,
			Target:    event.Effect.Target,
			RawEffect: event.Effect.RawEffect,
			Param:     event.Effect.Param,
			Charm:     cloneBool(event.Effect.Charm),
			Timestamp: event.Timestamp,
		},
	)

	const maxEffects = 100

	if len(s.combat.RecentEffects) > maxEffects {
		start := len(s.combat.RecentEffects) - maxEffects

		trimmed := make([]EffectState, maxEffects)
		copy(trimmed, s.combat.RecentEffects[start:])

		s.combat.RecentEffects = trimmed
	}
}

// Diese Funktion wird innerhalb von HandleEvent aufgerufen.
// Dort hält der Store bereits seinen Write-Lock.
// Deshalb hier NICHT erneut locken.
func (s *Store) participantBySubCircleUnsafe(
	subCircle int,
) *ParticipantState {
	participantID, ok :=
		s.combat.ParticipantBySubCircle[subCircle]

	if !ok {
		return nil
	}

	participant, ok :=
		s.combat.Participants[participantID]

	if !ok {
		return nil
	}

	return participant
}

func removeMatchingHangingEffect(
	current []ActiveEffect,
	effect *events.EffectEvent,
) []ActiveEffect {
	if effect == nil {
		return current
	}

	for index, active := range current {
		if !activeEffectMatches(active, effect) {
			continue
		}

		result := make(
			[]ActiveEffect,
			0,
			len(current)-1,
		)

		result = append(
			result,
			current[:index]...,
		)

		result = append(
			result,
			current[index+1:]...,
		)

		return result
	}

	return current
}

func activeEffectMatches(
	active ActiveEffect,
	effect *events.EffectEvent,
) bool {
	if active.RawEffect != effect.RawEffect {
		return false
	}

	if active.Param != effect.Param {
		return false
	}

	return boolPointersEqual(
		active.Charm,
		effect.Charm,
	)
}

func boolPointersEqual(
	left *bool,
	right *bool,
) bool {
	if left == nil && right == nil {
		return true
	}

	if left == nil || right == nil {
		return false
	}

	return *left == *right
}

func (s *Store) applyDuelSnapshot(
	snapshot events.DuelSnapshot,
) {
	if snapshot.ClientDuelID != "" {
		s.combat.ClientDuelID =
			snapshot.ClientDuelID
	}

	// Der DEBUGDUMPDUEL beschreibt die aktuell vorhandenen
	// Teilnehmer. Deshalb merken wir uns alle IDs, die im
	// Snapshot tatsächlich noch vorkommen.
	present :=
		make(
			map[string]struct{},
			len(snapshot.Participants),
		)

	for _, incoming := range snapshot.Participants {
		if incoming.ID == "" {
			continue
		}

		present[incoming.ID] =
			struct{}{}

		participant, ok :=
			s.combat.Participants[incoming.ID]

		if !ok {
			participant =
				newParticipantState(incoming.ID)

			s.combat.Participants[incoming.ID] =
				participant
		}

		participant.Team = incoming.Team
		participant.Health = incoming.Health
		participant.Pips = incoming.Pips
		participant.PPips = incoming.PPips

		if incoming.Kind != events.ParticipantUnknown {
			participant.Kind = incoming.Kind
		}

		if incoming.MobID != "" {
			participant.MobID = incoming.MobID
		}

		// DEBUGDUMPDUEL liefert normalerweise keine SC.
		// Eine bekannte Zuordnung wird deshalb nicht mit -1
		// überschrieben.
		if incoming.SubCircle >= 0 {
			s.assignSubCircle(
				participant,
				incoming.SubCircle,
			)
		}
	}

	// Alles, was vorher im Combat-State existierte,
	// aber im vollständigen Duel-Snapshot nicht mehr
	// vorkommt, ist nicht mehr Teil des Duels.
	for participantID, participant := range s.combat.Participants {

		if _, ok := present[participantID]; ok {
			continue
		}

		// Auch den SC-Index sauber entfernen.
		if participant != nil &&
			participant.SubCircle >= 0 {

			if mappedID, ok :=
				s.combat.ParticipantBySubCircle[participant.SubCircle]; ok &&
				mappedID == participantID {

				delete(
					s.combat.ParticipantBySubCircle,
					participant.SubCircle,
				)
			}
		}

		delete(
			s.combat.Participants,
			participantID,
		)
	}
}

func (s *Store) resetCombat() {
	s.combat = newCombatState()
}

func (s *Store) ParticipantBySubCircle(
	subCircle int,
) (ParticipantState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	participantID, ok :=
		s.combat.ParticipantBySubCircle[subCircle]

	if !ok {
		return ParticipantState{}, false
	}

	participant, ok :=
		s.combat.Participants[participantID]

	if !ok || participant == nil {
		return ParticipantState{}, false
	}

	return cloneParticipantState(*participant), true
}

func cloneParticipantState(
	participant ParticipantState,
) ParticipantState {
	result := participant

	result.Effects = cloneParticipantEffects(
		participant.Effects,
	)

	return result
}

func cloneParticipantEffects(
	effects ParticipantEffects,
) ParticipantEffects {
	result := ParticipantEffects{
		Hanging: make(
			[]ActiveEffect,
			len(effects.Hanging),
		),
		Auras: make(
			[]ActiveEffect,
			len(effects.Auras),
		),
	}

	for i, effect := range effects.Hanging {
		result.Hanging[i] =
			cloneActiveEffect(effect)
	}

	for i, effect := range effects.Auras {
		result.Auras[i] =
			cloneActiveEffect(effect)
	}

	return result
}

func cloneActiveEffect(
	effect ActiveEffect,
) ActiveEffect {
	result := effect
	result.Charm = cloneBool(effect.Charm)

	return result
}

func cloneBool(
	value *bool,
) *bool {
	if value == nil {
		return nil
	}

	result := *value

	return &result
}
