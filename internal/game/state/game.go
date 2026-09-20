package state

import "time"

type Mode int

const (
	ModeUnknown Mode = iota
	ModeRoaming
	ModeBattle
	ModeCharacterSelect
)

func (m Mode) String() string {
	switch m {
	case ModeRoaming:
		return "Roaming"
	case ModeBattle:
		return "Battle"
	case ModeCharacterSelect:
		return "CharacterSelect"
	default:
		return "Unknown"
	}
}

type GameState struct {
	Mode Mode

	ZoneKey string

	SessionStartedAt time.Time
}
