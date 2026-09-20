package parser

import (
	"strings"
	"time"

	"corvin101/internal/game/events"
)

type pendingParticipant struct {
	clientDuelID string

	rawName string

	kind events.ParticipantKind

	mobID string

	subCircle int
}

type unmappedParticipant struct {
	clientDuelID string

	participantID string

	kind events.ParticipantKind

	mobID string
}

type Parser struct {
	duelDump *duelDumpBuilder

	currentClientDuelID string

	// Informationen zwischen
	//
	// MSG_CombatAdd(...)
	//
	// und
	//
	// Added participant ... to duel ...
	//
	// zwischenspeichern.
	pendingParticipant *pendingParticipant

	// Teilnehmer, deren Participant-ID wir bereits kennen,
	// deren SubCircle aber noch nicht bekannt ist.
	unmappedParticipants []unmappedParticipant
}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) Reset() {
	p.duelDump = nil

	p.currentClientDuelID = ""

	p.pendingParticipant = nil

	p.unmappedParticipants = nil
}

func (p *Parser) Feed(line string) []events.Event {
	// Nur Zeilenende entfernen.
	//
	// Den eigentlichen Inhalt der Logzeile behalten wir
	// unverändert, damit Event.Raw wirklich dem Log entspricht.
	line = strings.TrimRight(
		line,
		"\r\n",
	)

	if strings.TrimSpace(line) == "" {
		return nil
	}

	timestamp := parseTimestamp(line)

	// ------------------------------------------------------------
	// Multiline duel dump
	// ------------------------------------------------------------

	if p.duelDump != nil {
		if result, handled :=
			p.feedDuelDump(
				line,
				timestamp,
			); handled {

			return result
		}
	}

	if isDuelDumpStart(line) {
		p.duelDump = &duelDumpBuilder{
			clientDuelID: p.currentClientDuelID,

			participants: make(
				[]events.Participant,
				0,
				8,
			),

			rawLines: []string{
				line,
			},
		}

		return nil
	}

	// ------------------------------------------------------------
	// Participant context
	// ------------------------------------------------------------

	if event, ok :=
		p.parseCombatAdd(
			line,
			timestamp,
		); ok {

		return []events.Event{
			event,
		}
	}

	// SubCircle-Zeilen können bereits ein Mapping für einen
	// vorher hinzugefügten Participant ergeben.
	if parsedEvents, ok :=
		p.parseSubCircle(
			line,
			timestamp,
		); ok {

		return parsedEvents
	}

	// ------------------------------------------------------------
	// Combat
	// ------------------------------------------------------------

	if event, ok :=
		p.parseCombatPhase(
			line,
			timestamp,
		); ok {

		if event.Combat != nil &&
			event.Combat.ClientDuelID != "" {

			p.currentClientDuelID =
				event.Combat.ClientDuelID
		}

		return []events.Event{
			event,
		}
	}

	if event, ok :=
		p.parseCombatActions(
			line,
			timestamp,
		); ok {

		if event.Combat != nil &&
			event.Combat.ClientDuelID != "" {

			p.currentClientDuelID =
				event.Combat.ClientDuelID
		}

		return nil
	}

	if event, ok :=
		p.parseSpellCast(
			line,
			timestamp,
		); ok {

		return []events.Event{
			event,
		}
	}

	// ------------------------------------------------------------
	// Effects
	// ------------------------------------------------------------

	if event, ok :=
		p.parseEffectProcessed(
			line,
			timestamp,
		); ok {

		return []events.Event{
			event,
		}
	}

	if event, ok :=
		p.parseHangingEffectAdded(
			line,
			timestamp,
		); ok {

		return []events.Event{
			event,
		}
	}

	if event, ok :=
		p.parseHangingEffectRemoved(
			line,
			timestamp,
		); ok {

		return []events.Event{
			event,
		}
	}

	if event, ok :=
		p.parseAuraEffectAdded(
			line,
			timestamp,
		); ok {

		return []events.Event{
			event,
		}
	}

	// ------------------------------------------------------------
	// Participants
	// ------------------------------------------------------------

	if parsedEvents, ok :=
		p.parseParticipantAdded(
			line,
			timestamp,
		); ok {

		return parsedEvents
	}

	if event, ok :=
		p.parseParticipantRemoved(
			line,
			timestamp,
		); ok {

		return []events.Event{
			event,
		}
	}

	// ------------------------------------------------------------
	// World / lifecycle
	// ------------------------------------------------------------

	if event, ok :=
		p.parseCombatJoined(
			line,
			timestamp,
		); ok {

		return []events.Event{
			event,
		}
	}

	if event, ok :=
		p.parseCombatFled(
			line,
			timestamp,
		); ok {

		return []events.Event{
			event,
		}
	}

	if event, ok :=
		p.parseCombatLeft(
			line,
			timestamp,
		); ok {

		return []events.Event{
			event,
		}
	}

	if event, ok :=
		p.parseCharacterSelected(
			line,
			timestamp,
		); ok {

		return []events.Event{
			event,
		}
	}

	if event, ok :=
		p.parseZoneLoaded(
			line,
			timestamp,
		); ok {

		return []events.Event{
			event,
		}
	}

	return nil
}

func parseTimestamp(line string) time.Time {
	if len(line) < 17 {
		return time.Time{}
	}

	value := line[:17]

	timestamp, err :=
		time.Parse(
			"01/02/06 15:04:05",
			value,
		)

	if err != nil {
		return time.Time{}
	}

	return timestamp
}
