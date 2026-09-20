package parser

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"corvin101/internal/game/events"
)

type duelDumpBuilder struct {
	clientDuelID string

	participants []events.Participant
	rawLines     []string
}

var duelDumpParticipantRe = regexp.MustCompile(
	`Participant: ID (\d+), Team (\d+), Health (-?\d+) Pips (\d+) PPips (\d+)`,
)

func isDuelDumpStart(line string) bool {
	return strings.Contains(
		line,
		"---BEGIN DEBUGDUMPDUEL---",
	)
}

func isDuelDumpEnd(line string) bool {
	return strings.Contains(
		line,
		"---END DEBUGDUMPDUEL---",
	)
}

func (p *Parser) feedDuelDump(
	line string,
	timestamp time.Time,
) ([]events.Event, bool) {
	if p.duelDump == nil {
		return nil, false
	}

	p.duelDump.rawLines = append(
		p.duelDump.rawLines,
		line,
	)

	if isDuelDumpEnd(line) {
		builder := p.duelDump
		p.duelDump = nil

		raw := strings.Join(builder.rawLines, "\n")

		return []events.Event{
			{
				Type:      events.DuelSnapshotReceived,
				Timestamp: timestamp,
				Raw:       raw,

				Snapshot: &events.DuelSnapshot{
					ClientDuelID: builder.clientDuelID,
					Participants: builder.participants,
				},
			},
		}, true
	}

	match := duelDumpParticipantRe.FindStringSubmatch(line)
	if match == nil {
		// Die Zeile gehört weiterhin zum Dump,
		// auch wenn wir sie noch nicht verstehen.
		return nil, true
	}

	team, err := strconv.Atoi(match[2])
	if err != nil {
		return nil, true
	}

	health, err := strconv.Atoi(match[3])
	if err != nil {
		return nil, true
	}

	pips, err := strconv.Atoi(match[4])
	if err != nil {
		return nil, true
	}

	powerPips, err := strconv.Atoi(match[5])
	if err != nil {
		return nil, true
	}

	p.duelDump.participants = append(
		p.duelDump.participants,
		events.Participant{
			ID:        match[1],
			SubCircle: -1,
			Team:      team,
			Health:    health,
			Pips:      pips,
			PPips:     powerPips,
		},
	)

	return nil, true
}
