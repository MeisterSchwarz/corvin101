package state

import (
	"time"

	"corvin101/internal/game/events"
)

type Snapshot struct {
	Game   GameSnapshot
	Combat CombatSnapshot
}

type GameSnapshot struct {
	Mode             Mode
	ZoneKey          string
	SessionStartedAt time.Time
}

type CombatSnapshot struct {
	Active bool

	ClientDuelID   string
	ResolverDuelID string

	Round int
	Phase events.CombatPhase

	StartedAt time.Time

	Participants  []ParticipantSnapshot
	LastSpell     *SpellSnapshot
	RecentEffects []EffectSnapshot
}

type ParticipantSnapshot struct {
	ID        string
	SubCircle int
	Kind      events.ParticipantKind
	MobID     string

	Team   int
	Health int
	Pips   int
	PPips  int

	Effects ParticipantEffectsSnapshot
}

type ParticipantEffectsSnapshot struct {
	Hanging []ActiveEffectSnapshot
	Auras   []ActiveEffectSnapshot
}

type ActiveEffectSnapshot struct {
	RawEffect string
	Param     int
	Charm     *bool
}

type SpellSnapshot struct {
	Round  int
	Caster int
	Name   string
}

type EffectSnapshot struct {
	Type events.Type

	Caster int
	Target int

	RawEffect string
	Param     int
	Charm     *bool
}

func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := Snapshot{
		Game: GameSnapshot{
			Mode:             s.game.Mode,
			ZoneKey:          s.game.ZoneKey,
			SessionStartedAt: s.game.SessionStartedAt,
		},

		Combat: CombatSnapshot{
			Active:         s.combat.Active,
			ClientDuelID:   s.combat.ClientDuelID,
			ResolverDuelID: s.combat.ResolverDuelID,
			Round:          s.combat.Round,
			Phase:          s.combat.Phase,
			StartedAt:      s.combat.StartedAt,

			Participants: make(
				[]ParticipantSnapshot,
				0,
				len(s.combat.Participants),
			),

			RecentEffects: make(
				[]EffectSnapshot,
				0,
				len(s.combat.RecentEffects),
			),
		},
	}

	for _, participant := range s.combat.Participants {
		if participant == nil {
			continue
		}

		participantSnapshot := ParticipantSnapshot{
			ID:        participant.ID,
			SubCircle: participant.SubCircle,
			Kind:      participant.Kind,
			MobID:     participant.MobID,

			Team:   participant.Team,
			Health: participant.Health,
			Pips:   participant.Pips,
			PPips:  participant.PPips,

			Effects: ParticipantEffectsSnapshot{
				Hanging: make(
					[]ActiveEffectSnapshot,
					0,
					len(participant.Effects.Hanging),
				),

				Auras: make(
					[]ActiveEffectSnapshot,
					0,
					len(participant.Effects.Auras),
				),
			},
		}

		for _, effect := range participant.Effects.Hanging {
			participantSnapshot.Effects.Hanging = append(
				participantSnapshot.Effects.Hanging,
				ActiveEffectSnapshot{
					RawEffect: effect.RawEffect,
					Param:     effect.Param,
					Charm:     cloneBool(effect.Charm),
				},
			)
		}

		for _, effect := range participant.Effects.Auras {
			participantSnapshot.Effects.Auras = append(
				participantSnapshot.Effects.Auras,
				ActiveEffectSnapshot{
					RawEffect: effect.RawEffect,
					Param:     effect.Param,
					Charm:     cloneBool(effect.Charm),
				},
			)
		}

		result.Combat.Participants = append(
			result.Combat.Participants,
			participantSnapshot,
		)
	}

	if s.combat.LastSpell != nil {
		result.Combat.LastSpell = &SpellSnapshot{
			Round:  s.combat.LastSpell.Round,
			Caster: s.combat.LastSpell.Caster,
			Name:   s.combat.LastSpell.Name,
		}
	}

	for _, effect := range s.combat.RecentEffects {
		result.Combat.RecentEffects = append(
			result.Combat.RecentEffects,
			EffectSnapshot{
				Type:      effect.Type,
				Caster:    effect.Caster,
				Target:    effect.Target,
				RawEffect: effect.RawEffect,
				Param:     effect.Param,
				Charm:     cloneBool(effect.Charm),
			},
		)
	}

	return result
}
