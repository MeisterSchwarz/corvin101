package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"corvin101/internal/config"
	"corvin101/internal/data/zones"
	"corvin101/internal/game/events"
	"corvin101/internal/game/parser"
	"corvin101/internal/game/state"
	"corvin101/internal/logreader"
	"corvin101/internal/presence/discord"
	"corvin101/internal/rest"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(
			"[config] load failed: ",
			err,
		)
	}

	logPath, ok := logreader.ResolveLogPath(cfg)
	if !ok {
		log.Fatal(
			"[main] could not resolve Wizard101 log path",
		)
	}

	fmt.Printf(
		"Using log: %s\n",
		logPath,
	)

	// --------------------------------------------------------
	// Core game components
	// --------------------------------------------------------

	gameParser := parser.New()
	gameState := state.New()

	// --------------------------------------------------------
	// Zone data
	// --------------------------------------------------------

	restClient := rest.New()

	zoneLoader := zones.NewRemoteLoader(
		restClient,
	)

	zoneRepository := zones.NewRepository(
		"de",
		zoneLoader,
	)

	// --------------------------------------------------------
	// Discord Rich Presence
	// --------------------------------------------------------

	presence := discord.New(
		config.DiscordClientID,
		zoneRepository,
	)

	defer presence.Close()

	// --------------------------------------------------------
	// Replay
	// --------------------------------------------------------

	handleReplayLine := func(line string) {
		parsedEvents := gameParser.Feed(line)

		for _, event := range parsedEvents {
			printEvent(event)

			gameState.HandleEvent(
				event,
			)
		}
	}

	fmt.Println()
	fmt.Println(
		"Replaying existing log...",
	)

	if err := logreader.Replay(
		logPath,
		handleReplayLine,
	); err != nil {
		log.Fatal(
			"[main] replay failed: ",
			err,
		)
	}

	fmt.Println()
	fmt.Println(
		"========================================",
	)
	fmt.Println(
		" STATE AFTER REPLAY",
	)
	fmt.Println(
		"========================================",
	)

	snapshot := gameState.Snapshot()

	printState(
		snapshot,
	)

	// --------------------------------------------------------
	// Discord erst NACH dem Replay verbinden.
	//
	// Dadurch erzeugen historische Events keine unnötigen
	// Presence-Updates.
	// --------------------------------------------------------

	if err := presence.Connect(); err != nil {
		log.Printf(
			"[discord] connect failed: %v",
			err,
		)
	} else {
		presence.Update(
			context.Background(),
			snapshot,
		)
	}

	// --------------------------------------------------------
	// Live Watch
	// --------------------------------------------------------

	fmt.Println()
	fmt.Println(
		"========================================",
	)
	fmt.Println(
		" LIVE WATCH",
	)
	fmt.Println(
		"========================================",
	)
	fmt.Println()
	fmt.Println(
		"Watching for new log events...",
	)
	fmt.Println(
		"Press Ctrl+C to stop.",
	)
	fmt.Println()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer cancel()

	reader := logreader.New(
		logPath,
	)

	err = reader.Watch(
		ctx,
		func(line string) {
			parsedEvents := gameParser.Feed(line)

			if len(parsedEvents) == 0 {
				return
			}

			for _, event := range parsedEvents {
				printEvent(event)

				gameState.HandleEvent(
					event,
				)
			}

			// Einen Snapshot NACH Verarbeitung aller Events
			// dieser Logzeile erstellen.
			snapshot := gameState.Snapshot()

			// Discord entscheidet selbst, ob sich überhaupt
			// etwas Presence-Relevantes geändert hat.
			presence.Update(
				ctx,
				snapshot,
			)

			fmt.Println()
			fmt.Println(
				"----------------------------------------",
			)
			fmt.Println(
				" CURRENT STATE",
			)
			fmt.Println(
				"----------------------------------------",
			)

			printState(
				snapshot,
			)

			fmt.Println()
		},
	)

	if err != nil {
		log.Fatal(
			"[main] watch failed: ",
			err,
		)
	}

	// --------------------------------------------------------
	// Final State
	// --------------------------------------------------------

	fmt.Println()
	fmt.Println(
		"========================================",
	)
	fmt.Println(
		" FINAL STATE",
	)
	fmt.Println(
		"========================================",
	)

	printState(
		gameState.Snapshot(),
	)
}

func printEvent(
	event events.Event,
) {
	fmt.Printf(
		"[EVENT] %-22s",
		event.Type.String(),
	)

	switch {
	case event.Zone != nil:
		fmt.Printf(
			" zone=%q",
			event.Zone.ZoneKey,
		)

	case event.Spell != nil:
		fmt.Printf(
			" resolver=%s round=%d caster=%d spell=%q",
			event.Spell.ResolverDuelID,
			event.Spell.Round,
			event.Spell.Caster,
			event.Spell.SpellName,
		)

	case event.Effect != nil:
		fmt.Printf(
			" caster=%d target=%d effect=%s param=%d",
			event.Effect.Caster,
			event.Effect.Target,
			event.Effect.RawEffect,
			event.Effect.Param,
		)

		if event.Effect.Charm != nil {
			fmt.Printf(
				" charm=%t",
				*event.Effect.Charm,
			)
		}

	case event.ParticipantMapping != nil:
		fmt.Printf(
			" duel=%s participant=%s subcircle=%d kind=%s mob=%q",
			event.ParticipantMapping.ClientDuelID,
			event.ParticipantMapping.ParticipantID,
			event.ParticipantMapping.SubCircle,
			event.ParticipantMapping.Kind.String(),
			event.ParticipantMapping.MobID,
		)

	case event.Participant != nil:
		fmt.Printf(
			" duel=%s participant=%s",
			event.Participant.ClientDuelID,
			event.Participant.Participant.ID,
		)

		if event.Participant.RawName != "" {
			fmt.Printf(
				" name=%q",
				event.Participant.RawName,
			)
		}

	case event.Snapshot != nil:
		fmt.Printf(
			" duel=%s participants=%d",
			event.Snapshot.ClientDuelID,
			len(event.Snapshot.Participants),
		)

	case event.Combat != nil:
		fmt.Printf(
			" duel=%s phase=%s",
			event.Combat.ClientDuelID,
			event.Combat.Phase.String(),
		)
	}

	fmt.Println()
}

func printState(
	snapshot state.Snapshot,
) {
	fmt.Printf(
		"Mode:          %s\n",
		snapshot.Game.Mode.String(),
	)

	fmt.Printf(
		"Zone:          %s\n",
		emptyFallback(
			snapshot.Game.ZoneKey,
		),
	)

	fmt.Println()

	fmt.Printf(
		"Combat:        %t\n",
		snapshot.Combat.Active,
	)

	fmt.Printf(
		"Client Duel:   %s\n",
		emptyFallback(
			snapshot.Combat.ClientDuelID,
		),
	)

	fmt.Printf(
		"Resolver Duel: %s\n",
		emptyFallback(
			snapshot.Combat.ResolverDuelID,
		),
	)

	fmt.Printf(
		"Round:         %d\n",
		snapshot.Combat.Round,
	)

	fmt.Printf(
		"Phase:         %s\n",
		snapshot.Combat.Phase.String(),
	)

	if snapshot.Combat.LastSpell != nil {
		fmt.Printf(
			"Last Spell:    %s (caster SC%d, round %d)\n",
			snapshot.Combat.LastSpell.Name,
			snapshot.Combat.LastSpell.Caster,
			snapshot.Combat.LastSpell.Round,
		)
	} else {
		fmt.Println(
			"Last Spell:    -",
		)
	}

	fmt.Println()
	fmt.Println(
		"Participants:",
	)

	participants := append(
		[]state.ParticipantSnapshot(nil),
		snapshot.Combat.Participants...,
	)

	sort.Slice(
		participants,
		func(i int, j int) bool {
			if participants[i].SubCircle !=
				participants[j].SubCircle {

				return participants[i].SubCircle <
					participants[j].SubCircle
			}

			return participants[i].ID <
				participants[j].ID
		},
	)

	if len(participants) == 0 {
		fmt.Println(
			"  -",
		)
	}

	for _, participant := range participants {
		fmt.Printf(
			"  SC=%2d ID=%-10s kind=%-11s team=%d hp=%d pips=%d ppips=%d",
			participant.SubCircle,
			participant.ID,
			participant.Kind.String(),
			participant.Team,
			participant.Health,
			participant.Pips,
			participant.PPips,
		)

		if participant.MobID != "" {
			fmt.Printf(
				" mob=%q",
				participant.MobID,
			)
		}

		fmt.Println()

		printParticipantEffects(
			participant,
		)
	}

	fmt.Println()
	fmt.Println(
		"Recent Effects:",
	)

	effects := snapshot.Combat.RecentEffects

	if len(effects) == 0 {
		fmt.Println(
			"  -",
		)

		return
	}

	start := 0

	const maxPrintedEffects = 10

	if len(effects) > maxPrintedEffects {
		start =
			len(effects) -
				maxPrintedEffects
	}

	for _, effect := range effects[start:] {
		fmt.Printf(
			"  %-22s caster=SC%-2d target=SC%-2d %-30s param=%4d",
			effect.Type.String(),
			effect.Caster,
			effect.Target,
			effect.RawEffect,
			effect.Param,
		)

		if effect.Charm != nil {
			fmt.Printf(
				" charm=%t",
				*effect.Charm,
			)
		}

		fmt.Println()
	}
}

func printParticipantEffects(
	participant state.ParticipantSnapshot,
) {
	if len(participant.Effects.Hanging) == 0 &&
		len(participant.Effects.Auras) == 0 {

		fmt.Println(
			"      Effects: -",
		)

		return
	}

	if len(participant.Effects.Hanging) > 0 {
		fmt.Println(
			"      Hanging:",
		)

		for _, effect := range participant.Effects.Hanging {
			fmt.Printf(
				"        %-30s param=%4d",
				effect.RawEffect,
				effect.Param,
			)

			if effect.Charm != nil {
				fmt.Printf(
					" charm=%t",
					*effect.Charm,
				)
			}

			fmt.Println()
		}
	}

	if len(participant.Effects.Auras) > 0 {
		fmt.Println(
			"      Auras:",
		)

		for _, effect := range participant.Effects.Auras {
			fmt.Printf(
				"        %-30s param=%4d",
				effect.RawEffect,
				effect.Param,
			)

			if effect.Charm != nil {
				fmt.Printf(
					" charm=%t",
					*effect.Charm,
				)
			}

			fmt.Println()
		}
	}
}

func emptyFallback(
	value string,
) string {
	if value == "" {
		return "-"
	}

	return value
}
