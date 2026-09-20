package discord

import (
	"context"

	"github.com/hugolgst/rich-go/client"

	"corvin101/internal/data/zones"
	"corvin101/internal/game/state"
)

func (p *Presence) buildActivity(
	ctx context.Context,
	snapshot state.Snapshot,
) (client.Activity, error) {
	switch snapshot.Game.Mode {
	case state.ModeBattle:
		return p.buildBattleActivity(
			ctx,
			snapshot,
		)

	case state.ModeRoaming:
		return p.buildRoamingActivity(
			ctx,
			snapshot,
		)

	case state.ModeCharacterSelect:
		return p.buildCharacterSelectActivity(
			snapshot,
		), nil

	default:
		return p.baseActivity(
			snapshot,
		), nil
	}
}

func (p *Presence) baseActivity(
	snapshot state.Snapshot,
) client.Activity {
	activity := client.Activity{}

	if !snapshot.Game.SessionStartedAt.IsZero() {
		startedAt := snapshot.Game.SessionStartedAt

		activity.Timestamps = &client.Timestamps{
			Start: &startedAt,
		}
	}

	return activity
}

func (p *Presence) buildRoamingActivity(
	ctx context.Context,
	snapshot state.Snapshot,
) (client.Activity, error) {
	activity := p.baseActivity(snapshot)

	zone := p.resolveZone(
		ctx,
		snapshot.Game.ZoneKey,
	)

	applyZone(
		&activity,
		snapshot.Game.ZoneKey,
		zone,
	)

	return activity, nil
}

func (p *Presence) buildBattleActivity(
	ctx context.Context,
	snapshot state.Snapshot,
) (client.Activity, error) {
	activity := p.baseActivity(snapshot)

	zone := p.resolveZone(
		ctx,
		snapshot.Game.ZoneKey,
	)

	applyZone(
		&activity,
		snapshot.Game.ZoneKey,
		zone,
	)

	activity.SmallImage = "battle"
	activity.SmallText = "Im Kampf"

	if !snapshot.Combat.StartedAt.IsZero() {
		startedAt := snapshot.Combat.StartedAt

		activity.Timestamps = &client.Timestamps{
			Start: &startedAt,
		}
	}

	return activity, nil
}

func (p *Presence) buildCharacterSelectActivity(
	snapshot state.Snapshot,
) client.Activity {
	activity := p.baseActivity(snapshot)

	activity.Details = "Charakterauswahl"
	activity.LargeImage = "corvin101"

	return activity
}

func (p *Presence) resolveZone(
	ctx context.Context,
	zoneKey string,
) zones.ZoneInfo {
	if p.zones == nil {
		return zones.ZoneInfo{
			Name:  zoneKey,
			World: zones.WorldKey(zoneKey),
			Image: "dungeons",
		}
	}

	return p.zones.ResolveOrFallback(
		ctx,
		zoneKey,
	)
}

func applyZone(
	activity *client.Activity,
	zoneKey string,
	zone zones.ZoneInfo,
) {
	if zone.Name != "" {
		activity.Details = zone.Name
	} else {
		activity.Details = zoneKey
	}

	if zone.Sub != "" {
		activity.State = zone.Sub
	}

	if zone.Image != "" {
		activity.LargeImage = zone.Image
	}

	if zone.World != "" {
		activity.LargeText = zone.World
	}
}
