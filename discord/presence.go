package discord

import (
	"log"
	"strings"
	"sync"

	"github.com/hugolgst/rich-go/client"

	"wizard101rpc/config"
	"wizard101rpc/state"
	"wizard101rpc/zones"
)

/*
	Connection handling
*/

var (
	mu        sync.Mutex
	connected bool
)

// logs into Discord if needed
func EnsureConnected() error {
	mu.Lock()
	defer mu.Unlock()

	if connected {
		return nil
	}

	if err := client.Login(config.DiscordClientID); err != nil {
		log.Println("[discord] login failed:", err)
		return err
	}

	connected = true
	return nil
}

// disconnects from Discord
func Logout() {
	mu.Lock()
	defer mu.Unlock()

	if !connected {
		return
	}

	client.Logout()
	connected = false
}

/*
	Log events
*/

type LogEventType int

const (
	EventNone LogEventType = iota
	EventCombatStart
	EventCombatEnd
	EventZoneLoaded
	EventCharacterSelect
)

type LogEvent struct {
	Type LogEventType
	Data string
}

// parses and applies a log line
func HandleLogLine(line string) {
	event := parseLogLine(line)
	if event.Type == EventNone {
		return
	}

	if applyEvent(event) {
		updateActivity()
	}
}

// extracts a log event from a line
func parseLogLine(line string) LogEvent {
	switch {
	case strings.Contains(line, "My player was added to a combat."):
		return LogEvent{Type: EventCombatStart}

	case strings.Contains(line, "Ending Phase received, destroying combat"),
		strings.Contains(line, "MSG_CombatFlee()"):
		return LogEvent{Type: EventCombatEnd}

	case strings.Contains(line, "Character selection is now enabled."):
		return LogEvent{Type: EventCharacterSelect}

	case strings.Contains(line, "Finished loading zone"):
		if zone := extractZone(line); zone != "" {
			return LogEvent{Type: EventZoneLoaded, Data: zone}
		}
	}

	return LogEvent{Type: EventNone}
}

// extracts zone key from log line
func extractZone(line string) string {
	start := strings.Index(line, "'")
	end := strings.LastIndex(line, "'")

	if start == -1 || end <= start {
		return ""
	}

	return line[start+1 : end]
}

// mutates state and returns if activity should update
func applyEvent(event LogEvent) bool {
	switch event.Type {
	case EventCombatStart:
		return state.EnterBattle()

	case EventCombatEnd:
		return state.LeaveBattle()

	case EventZoneLoaded:
		return state.EnterZone(event.Data)

	case EventCharacterSelect:
		return state.EnterCharacterSelect()
	}

	return false
}

/*
	Discord activity
*/

// updates Discord presence based on state
func updateActivity() {
	switch state.Current() {
	case state.Battle:
		updateBattle()
	case state.Roaming:
		updateRoaming()
	case state.CharacterSelect:
		updateCharacterSelect()
	}
}

// returns a shared activity template
func baseActivity() client.Activity {
	return client.Activity{
		Timestamps: &client.Timestamps{
			Start: state.SessionStart(),
		},
	}
}

// sets roaming presence
func updateRoaming() {
	name, sub, world, image := zones.Resolve(state.LastZone())

	a := baseActivity()
	a.Details = name
	a.State = sub

	if image != "" {
		a.LargeImage = image
		a.LargeText = world
	}

	client.SetActivity(a)
}

// sets battle presence
func updateBattle() {
	name, sub, world, image := zones.Resolve(state.LastZone())

	a := baseActivity()
	a.Timestamps.Start = state.BattleStart()

	if sub != "" {
		a.Details = "(⚔️) " + sub
		a.State = name
	} else {
		a.State = "(⚔️) " + name
	}

	if image != "" {
		a.LargeImage = image
		a.LargeText = world
	}

	client.SetActivity(a)
}

// sets character select presence
func updateCharacterSelect() {
	a := baseActivity()
	a.Details = "Charakterauswahl"
	a.LargeImage = "wizardcity"

	client.SetActivity(a)
}

/*
	Init from existing log
*/

// restores state from previous log lines
func InitFromLog(lines []string) {
	var (
		foundZone   bool
		foundBattle bool
		inBattle    bool
		lastEvent   LogEventType = EventNone
	)

	for i := len(lines) - 1; i >= 0; i-- {
		event := parseLogLine(lines[i])

		if lastEvent == EventNone && event.Type != EventNone {
			lastEvent = event.Type
		}

		switch event.Type {

		case EventZoneLoaded:
			if !foundZone {
				state.EnterZone(event.Data)
				foundZone = true
			}

		case EventCombatStart:
			if !foundBattle {
				inBattle = true
				foundBattle = true
			}

		case EventCombatEnd:
			if !foundBattle {
				inBattle = false
				foundBattle = true
			}

		case EventCharacterSelect:
			if lastEvent == EventCharacterSelect {
				state.EnterCharacterSelect()
			}
		}

		if foundZone && foundBattle {
			break
		}
	}

	if inBattle {
		state.EnterBattle()
	}

	updateActivity()
}
