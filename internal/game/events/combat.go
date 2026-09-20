package events

type CombatPhase int

const (
	PhaseUnknown CombatPhase = iota
	PhasePrePlanning
	PhasePlanning
	PhaseExecution
	PhaseResolution
	PhaseVictory
	PhaseEnded
)

func (p CombatPhase) String() string {
	switch p {
	case PhasePrePlanning:
		return "PrePlanning"
	case PhasePlanning:
		return "Planning"
	case PhaseExecution:
		return "Execution"
	case PhaseResolution:
		return "Resolution"
	case PhaseVictory:
		return "Victory"
	case PhaseEnded:
		return "Ended"
	default:
		return "Unknown"
	}
}

type CombatEvent struct {
	// ClientDuelID ist die ID aus Meldungen wie:
	//
	// Duel 9807540: MSG_CombatPhase(...)
	//
	// Sie wird absichtlich getrennt von ResolverDuelID gehalten.
	ClientDuelID string

	// ResolverDuelID ist die ID aus:
	//
	// CombatResolver::ResolveCombatRound.
	// Duel ID:2889340635944691380
	//
	// Aktuell nehmen wir NICHT an, dass diese ID semantisch
	// identisch mit ClientDuelID ist.
	ResolverDuelID string

	Phase CombatPhase
}

type SpellEvent struct {
	ResolverDuelID string

	Round  int
	Caster int

	SpellName string
}

type EffectEvent struct {
	Caster int
	Target int

	// RawEffect behält absichtlich den Wizard101-Namen.
	//
	// Beispiel:
	// kModifyOutgoingDamage
	// kModifyIncomingDamage
	// kDamage
	//
	// Die semantische Interpretation erfolgt später.
	RawEffect string

	Param int

	// Optional metadata available on some effect log lines.
	Charm *bool
}
