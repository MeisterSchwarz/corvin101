package events

import "time"

type Type int

const (
	Unknown Type = iota

	// World / game
	ZoneLoaded
	CharacterSelected

	// Combat lifecycle
	CombatJoined
	CombatLeft
	CombatFled
	CombatVictory

	// Combat state
	CombatPhaseChanged
	SpellCast

	// Participants
	ParticipantAdded
	ParticipantMapped
	ParticipantRemoved
	DuelSnapshotReceived

	// Effects
	EffectProcessed
	HangingEffectAdded
	HangingEffectRemoved
	AuraEffectAdded
)

func (t Type) String() string {
	switch t {
	case ZoneLoaded:
		return "ZoneLoaded"

	case CharacterSelected:
		return "CharacterSelected"

	case CombatJoined:
		return "CombatJoined"

	case CombatLeft:
		return "CombatLeft"

	case CombatFled:
		return "CombatFled"

	case CombatVictory:
		return "CombatVictory"

	case CombatPhaseChanged:
		return "CombatPhaseChanged"

	case SpellCast:
		return "SpellCast"

	case ParticipantAdded:
		return "ParticipantAdded"

	case ParticipantMapped:
		return "ParticipantMapped"

	case ParticipantRemoved:
		return "ParticipantRemoved"

	case DuelSnapshotReceived:
		return "DuelSnapshotReceived"

	case EffectProcessed:
		return "EffectProcessed"

	case HangingEffectAdded:
		return "HangingEffectAdded"

	case HangingEffectRemoved:
		return "HangingEffectRemoved"

	case AuraEffectAdded:
		return "AuraEffectAdded"

	default:
		return "Unknown"
	}
}

type Event struct {
	Type      Type
	Timestamp time.Time
	Raw       string

	Zone *ZoneEvent

	Combat *CombatEvent

	Spell *SpellEvent

	Effect *EffectEvent

	Participant *ParticipantEvent

	ParticipantMapping *ParticipantMappingEvent

	Snapshot *DuelSnapshot
}
