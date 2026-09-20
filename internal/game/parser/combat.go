package parser

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"corvin101/internal/game/events"
)

var (
	combatPhaseRe = regexp.MustCompile(
		`Duel (\d+): MSG_CombatPhase\((kPhase_[A-Za-z]+)\)`,
	)

	combatActionsRe = regexp.MustCompile(
		`Duel (\d+): MSG_CombatActions\(\)`,
	)

	resolveCombatRoundRe = regexp.MustCompile(
		`CombatResolver::ResolveCombatRound\. Duel ID:(\d+) Round:(\d+) Caster:(\d+) Spell:(.+)$`,
	)

	processTargetsRe = regexp.MustCompile(
		`EffectTargetPairList::ProcessTargets\. Caster:(\d+) Target:(\d+) Effect:(\S+) Param:(-?\d+)`,
	)

	addHangingEffectRe = regexp.MustCompile(
		`CombatParticipant::AddHangingEffect\. Target:(\d+) Effect:(\S+)(?: Charm:(Yes|No))? Param:(-?\d+)`,
	)

	removeHangingEffectRe = regexp.MustCompile(
		`CombatParticipant::RemoveHangingEffect\. Target:(\d+) Effect:(\S+)(?: Charm:(Yes|No))? Param:(-?\d+)`,
	)

	addAuraEffectRe = regexp.MustCompile(
		`CombatParticipant::AddAuraEffect\. Target:(\d+) Effect:(\S+) Param:(-?\d+)`,
	)

	combatAddRe = regexp.MustCompile(
		`Duel (\d+): MSG_CombatAdd\(([^)]+)\)`,
	)

	subCircleRe = regexp.MustCompile(
		`My SubCircle: (\d+), OtherPart's SubCircle: (\d+)`,
	)

	participantAddedRe = regexp.MustCompile(
		`Added participant (\d+) to duel (\d+)`,
	)

	combatRemoveRe = regexp.MustCompile(
		`Duel (\d+): MSG_CombatRemove\(([^)]+)\)`,
	)
)

// ------------------------------------------------------------
// Combat phase
// ------------------------------------------------------------

func (p *Parser) parseCombatPhase(
	line string,
	timestamp time.Time,
) (events.Event, bool) {
	match :=
		combatPhaseRe.FindStringSubmatch(
			line,
		)

	if match == nil {
		return events.Event{}, false
	}

	phase :=
		parseCombatPhaseName(
			match[2],
		)

	eventType :=
		events.CombatPhaseChanged

	if phase == events.PhaseVictory {
		eventType =
			events.CombatVictory
	}

	return events.Event{
		Type:      eventType,
		Timestamp: timestamp,
		Raw:       line,

		Combat: &events.CombatEvent{
			ClientDuelID: match[1],
			Phase:        phase,
		},
	}, true
}

// ------------------------------------------------------------
// Combat actions
// ------------------------------------------------------------

func (p *Parser) parseCombatActions(
	line string,
	timestamp time.Time,
) (events.Event, bool) {
	match :=
		combatActionsRe.FindStringSubmatch(
			line,
		)

	if match == nil {
		return events.Event{}, false
	}

	return events.Event{
		Type:      events.Unknown,
		Timestamp: timestamp,
		Raw:       line,

		Combat: &events.CombatEvent{
			ClientDuelID: match[1],
		},
	}, true
}

// ------------------------------------------------------------
// Resolve combat round
// ------------------------------------------------------------

func (p *Parser) parseSpellCast(
	line string,
	timestamp time.Time,
) (events.Event, bool) {
	match :=
		resolveCombatRoundRe.FindStringSubmatch(
			line,
		)

	if match == nil {
		return events.Event{}, false
	}

	round, err :=
		strconv.Atoi(
			match[2],
		)

	if err != nil {
		return events.Event{}, false
	}

	caster, err :=
		strconv.Atoi(
			match[3],
		)

	if err != nil {
		return events.Event{}, false
	}

	spellName :=
		strings.TrimSpace(
			match[4],
		)

	if spellName == "" {
		return events.Event{}, false
	}

	return events.Event{
		Type:      events.SpellCast,
		Timestamp: timestamp,
		Raw:       line,

		Spell: &events.SpellEvent{
			ResolverDuelID: match[1],
			Round:          round,
			Caster:         caster,
			SpellName:      spellName,
		},
	}, true
}

// ------------------------------------------------------------
// Effect processed
// ------------------------------------------------------------

func (p *Parser) parseEffectProcessed(
	line string,
	timestamp time.Time,
) (events.Event, bool) {
	match :=
		processTargetsRe.FindStringSubmatch(
			line,
		)

	if match == nil {
		return events.Event{}, false
	}

	caster, err :=
		strconv.Atoi(
			match[1],
		)

	if err != nil {
		return events.Event{}, false
	}

	target, err :=
		strconv.Atoi(
			match[2],
		)

	if err != nil {
		return events.Event{}, false
	}

	param, err :=
		strconv.Atoi(
			match[4],
		)

	if err != nil {
		return events.Event{}, false
	}

	return events.Event{
		Type:      events.EffectProcessed,
		Timestamp: timestamp,
		Raw:       line,

		Effect: &events.EffectEvent{
			Caster:    caster,
			Target:    target,
			RawEffect: match[3],
			Param:     param,
		},
	}, true
}

// ------------------------------------------------------------
// Hanging effect added
// ------------------------------------------------------------

func (p *Parser) parseHangingEffectAdded(
	line string,
	timestamp time.Time,
) (events.Event, bool) {
	match :=
		addHangingEffectRe.FindStringSubmatch(
			line,
		)

	if match == nil {
		return events.Event{}, false
	}

	target, err :=
		strconv.Atoi(
			match[1],
		)

	if err != nil {
		return events.Event{}, false
	}

	param, err :=
		strconv.Atoi(
			match[4],
		)

	if err != nil {
		return events.Event{}, false
	}

	var charm *bool

	if match[3] != "" {
		value :=
			match[3] == "Yes"

		charm = &value
	}

	return events.Event{
		Type:      events.HangingEffectAdded,
		Timestamp: timestamp,
		Raw:       line,

		Effect: &events.EffectEvent{
			Caster:    -1,
			Target:    target,
			RawEffect: match[2],
			Param:     param,
			Charm:     charm,
		},
	}, true
}

// ------------------------------------------------------------
// Hanging effect removed
// ------------------------------------------------------------

func (p *Parser) parseHangingEffectRemoved(
	line string,
	timestamp time.Time,
) (events.Event, bool) {
	match :=
		removeHangingEffectRe.FindStringSubmatch(
			line,
		)

	if match == nil {
		return events.Event{}, false
	}

	target, err :=
		strconv.Atoi(
			match[1],
		)

	if err != nil {
		return events.Event{}, false
	}

	param, err :=
		strconv.Atoi(
			match[4],
		)

	if err != nil {
		return events.Event{}, false
	}

	var charm *bool

	if match[3] != "" {
		value :=
			match[3] == "Yes"

		charm = &value
	}

	return events.Event{
		Type:      events.HangingEffectRemoved,
		Timestamp: timestamp,
		Raw:       line,

		Effect: &events.EffectEvent{
			Caster:    -1,
			Target:    target,
			RawEffect: match[2],
			Param:     param,
			Charm:     charm,
		},
	}, true
}

// ------------------------------------------------------------
// Aura added
// ------------------------------------------------------------

func (p *Parser) parseAuraEffectAdded(
	line string,
	timestamp time.Time,
) (events.Event, bool) {
	match :=
		addAuraEffectRe.FindStringSubmatch(
			line,
		)

	if match == nil {
		return events.Event{}, false
	}

	target, err :=
		strconv.Atoi(
			match[1],
		)

	if err != nil {
		return events.Event{}, false
	}

	param, err :=
		strconv.Atoi(
			match[3],
		)

	if err != nil {
		return events.Event{}, false
	}

	return events.Event{
		Type:      events.AuraEffectAdded,
		Timestamp: timestamp,
		Raw:       line,

		Effect: &events.EffectEvent{
			Caster:    -1,
			Target:    target,
			RawEffect: match[2],
			Param:     param,
		},
	}, true
}

// ------------------------------------------------------------
// MSG_CombatAdd
// ------------------------------------------------------------

func (p *Parser) parseCombatAdd(
	line string,
	timestamp time.Time,
) (events.Event, bool) {
	match :=
		combatAddRe.FindStringSubmatch(
			line,
		)

	if match == nil {
		return events.Event{}, false
	}

	clientDuelID :=
		match[1]

	rawName :=
		strings.TrimSpace(
			match[2],
		)

	p.currentClientDuelID =
		clientDuelID

	kind :=
		events.ParticipantUnknown

	mobID := ""

	switch rawName {
	case "Local Player":
		kind =
			events.ParticipantLocalPlayer

	default:
		kind =
			events.ParticipantMob

		mobID =
			rawName
	}

	p.pendingParticipant =
		&pendingParticipant{
			clientDuelID: clientDuelID,

			rawName: rawName,

			kind: kind,

			mobID: mobID,

			subCircle: -1,
		}

	// MSG_CombatAdd selbst ist noch kein vollständiger
	// ParticipantAdded-Event.
	//
	// Wir warten auf:
	//
	// Added participant X to duel Y
	//
	return events.Event{}, true
}

// ------------------------------------------------------------
// SubCircle mapping
// ------------------------------------------------------------

func (p *Parser) parseSubCircle(
	line string,
	timestamp time.Time,
) ([]events.Event, bool) {
	match :=
		subCircleRe.FindStringSubmatch(
			line,
		)

	if match == nil {
		return nil, false
	}

	mySubCircle, err :=
		strconv.Atoi(
			match[1],
		)

	if err != nil {
		return nil, true
	}

	otherSubCircle, err :=
		strconv.Atoi(
			match[2],
		)

	if err != nil {
		return nil, true
	}

	result :=
		make(
			[]events.Event,
			0,
			1,
		)

	// Die aktuell hinzukommende Entity bezeichnet sich
	// im Log als "My SubCircle".
	if p.pendingParticipant != nil {
		p.pendingParticipant.subCircle =
			mySubCircle
	}

	// Beispiel aus dem echten Log:
	//
	// erster Mob wurde bereits hinzugefügt,
	// seine SC war aber noch unbekannt.
	//
	// Danach:
	//
	// MSG_CombatAdd(Local Player)
	// My SubCircle: 4, OtherPart's SubCircle: 0
	//
	// Damit können wir den vorherigen noch nicht
	// zugeordneten Teilnehmer auf SC0 setzen.
	//
	// Wir tun das nur, wenn genau ein ungelöster Participant
	// existiert. Dadurch vermeiden wir Mehrdeutigkeiten.
	if len(p.unmappedParticipants) == 1 {
		unmapped :=
			p.unmappedParticipants[0]

		result =
			append(
				result,
				events.Event{
					Type: events.ParticipantMapped,

					Timestamp: timestamp,

					Raw: line,

					ParticipantMapping: &events.ParticipantMappingEvent{
						ClientDuelID: unmapped.clientDuelID,

						ParticipantID: unmapped.participantID,

						SubCircle: otherSubCircle,

						Kind: unmapped.kind,

						MobID: unmapped.mobID,
					},
				},
			)

		p.unmappedParticipants = nil
	}

	return result, true
}

// ------------------------------------------------------------
// Added participant
// ------------------------------------------------------------

func (p *Parser) parseParticipantAdded(
	line string,
	timestamp time.Time,
) ([]events.Event, bool) {
	match :=
		participantAddedRe.FindStringSubmatch(
			line,
		)

	if match == nil {
		return nil, false
	}

	participantID :=
		match[1]

	clientDuelID :=
		match[2]

	p.currentClientDuelID =
		clientDuelID

	participant :=
		events.Participant{
			ID: participantID,

			SubCircle: -1,

			Kind: events.ParticipantUnknown,
		}

	rawName := ""

	if p.pendingParticipant != nil {
		rawName =
			p.pendingParticipant.rawName

		participant.SubCircle =
			p.pendingParticipant.subCircle

		participant.Kind =
			p.pendingParticipant.kind

		participant.MobID =
			p.pendingParticipant.mobID
	}

	result :=
		[]events.Event{
			{
				Type: events.ParticipantAdded,

				Timestamp: timestamp,

				Raw: line,

				Participant: &events.ParticipantEvent{
					ClientDuelID: clientDuelID,

					Participant: participant,

					RawName: rawName,
				},
			},
		}

	// Falls wir die SubCircle bereits kennen,
	// erzeugen wir zusätzlich ein explizites Mapping-Event.
	if participant.SubCircle >= 0 {
		result =
			append(
				result,
				events.Event{
					Type: events.ParticipantMapped,

					Timestamp: timestamp,

					Raw: line,

					ParticipantMapping: &events.ParticipantMappingEvent{
						ClientDuelID: clientDuelID,

						ParticipantID: participantID,

						SubCircle: participant.SubCircle,

						Kind: participant.Kind,

						MobID: participant.MobID,
					},
				},
			)
	} else {
		// Noch keine SC bekannt.
		//
		// Wir merken uns den Participant für eine spätere
		// "OtherPart's SubCircle"-Zuordnung.
		p.unmappedParticipants =
			append(
				p.unmappedParticipants,
				unmappedParticipant{
					clientDuelID: clientDuelID,

					participantID: participantID,

					kind: participant.Kind,

					mobID: participant.MobID,
				},
			)
	}

	p.pendingParticipant = nil

	return result, true
}

// ------------------------------------------------------------
// Participant removed
// ------------------------------------------------------------

func (p *Parser) parseParticipantRemoved(
	line string,
	timestamp time.Time,
) (events.Event, bool) {
	match :=
		combatRemoveRe.FindStringSubmatch(
			line,
		)

	if match == nil {
		return events.Event{}, false
	}

	name :=
		strings.TrimSpace(
			match[2],
		)

	return events.Event{
		Type:      events.ParticipantRemoved,
		Timestamp: timestamp,
		Raw:       line,

		Participant: &events.ParticipantEvent{
			ClientDuelID: match[1],

			RawName: name,

			Participant: events.Participant{
				SubCircle: -1,
			},
		},
	}, true
}

// ------------------------------------------------------------
// Phase conversion
// ------------------------------------------------------------

func parseCombatPhaseName(
	value string,
) events.CombatPhase {
	switch value {
	case "kPhase_PrePlanning":
		return events.PhasePrePlanning

	case "kPhase_Planning":
		return events.PhasePlanning

	case "kPhase_Execution":
		return events.PhaseExecution

	case "kPhase_Resolution":
		return events.PhaseResolution

	case "kPhase_Victory":
		return events.PhaseVictory

	case "kPhase_Ended":
		return events.PhaseEnded

	default:
		return events.PhaseUnknown
	}
}
