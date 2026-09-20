package events

type ParticipantKind int

const (
	ParticipantUnknown ParticipantKind = iota

	ParticipantLocalPlayer
	ParticipantMob
	ParticipantOtherPlayer
)

func (k ParticipantKind) String() string {
	switch k {
	case ParticipantLocalPlayer:
		return "LocalPlayer"

	case ParticipantMob:
		return "Mob"

	case ParticipantOtherPlayer:
		return "OtherPlayer"

	default:
		return "Unknown"
	}
}

type Participant struct {
	// Interne Participant-ID des konkreten Kampfteilnehmers.
	ID string

	// CombatResolver-Slot.
	//
	// Beispielsweise:
	// 0, 1, 2, 3 = Gegner
	// 4, 5, 6, 7 = andere Team-Seite
	//
	// Wir verlassen uns aber nicht auf diese Semantik,
	// sondern speichern nur den beobachteten Wert.
	//
	// -1 = noch unbekannt.
	SubCircle int

	Kind ParticipantKind

	// Name/Identifier aus MSG_CombatAdd(...).
	//
	// Bei Mobs ist das beispielsweise:
	// AV-Froudling-Hobgoblin-Fiend-R11-01
	//
	// Beim LocalPlayer bleibt das leer.
	MobID string

	Team int

	Health int
	Pips   int
	PPips  int
}

type ParticipantEvent struct {
	ClientDuelID string

	Participant Participant

	// Originalwert aus MSG_CombatAdd(...).
	RawName string
}

type ParticipantMappingEvent struct {
	ClientDuelID string

	ParticipantID string

	SubCircle int

	Kind ParticipantKind

	MobID string
}

type DuelSnapshot struct {
	ClientDuelID string

	Participants []Participant
}

type ZoneEvent struct {
	ZoneKey string
}
