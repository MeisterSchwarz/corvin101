package enemies

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/fs"
	"slices"
)

type CheatNote struct {
	ID, Text string
	Enemies  []CheatEnemyReference
}
type CheatEnemyReference struct {
	Key, EnemyID, FamilyID, Label, Mode string
	HeadID                              string
	Schools                             []School
}
type strategyEntry struct {
	ID      string                `yaml:"id"`
	Enemies enemyReferenceSources `yaml:"enemies"`
}
type enemyReferenceSources map[string]struct {
	Enemy string `yaml:"enemy"`
	Label string `yaml:"label"`
}

func (s enemyReferenceSources) resolve(textFor func(string) string) ([]CheatEnemyReference, error) {
	keys := make([]string, 0, len(s))
	for key := range s {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var result []CheatEnemyReference
	for _, key := range keys {
		ref := s[key]
		if ref.Enemy == "" {
			return nil, fmt.Errorf("empty enemy reference %s", key)
		}
		label := ""
		if ref.Label != "" {
			label = textFor(ref.Label)
		}
		result = append(result, CheatEnemyReference{Key: key, EnemyID: ref.Enemy, Label: label})
	}
	return result, nil
}

func (e *strategyEntry) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode && n.Tag == "!!str" {
		e.ID = n.Value
		return nil
	}
	type plain strategyEntry
	return n.Decode((*plain)(e))
}
func cloneCheatNotes(source []CheatNote) []CheatNote {
	result := slices.Clone(source)
	for i := range result {
		result[i].Enemies = slices.Clone(result[i].Enemies)
		for j := range result[i].Enemies {
			result[i].Enemies[j].Schools = slices.Clone(result[i].Enemies[j].Schools)
		}
	}
	return result
}

type enemyCheatData struct {
	Mechanics []CheatMechanic
	Strategy  []CheatNote
}

func (e EnemyInfo) HasCheatSheet() bool { return len(e.Cheats) > 0 || len(e.Strategy) > 0 }

func enemyMode(mode string) string {
	if mode == "" {
		return "standard"
	}
	return mode
}

type CheatMechanic struct {
	ID, Text string
	Enemies  []CheatEnemyReference
	Notes    []CheatNote
	Schedule *CheatSchedule
}

type CheatSchedule struct {
	Repeat int                  `yaml:"repeat"`
	Timing string               `yaml:"timing,omitempty"`
	Events []CheatScheduleEvent `yaml:"events"`
}
type CheatScheduleEvent struct {
	Round  int    `yaml:"round"`
	Timing string `yaml:"timing,omitempty"`
	Text   string `yaml:"text"`
	ID     string `yaml:"-"`
}

func cloneCheats(source []CheatMechanic) []CheatMechanic {
	result := slices.Clone(source)
	for i := range result {
		result[i].Enemies = cloneCheatNotes([]CheatNote{{Enemies: result[i].Enemies}})[0].Enemies
		result[i].Notes = cloneCheatNotes(result[i].Notes)
		if source[i].Schedule != nil {
			schedule := *source[i].Schedule
			schedule.Events = slices.Clone(schedule.Events)
			result[i].Schedule = &schedule
		}
	}
	return result
}

func (r *Repository) loadCheats(ctx context.Context, world string) (map[string]enemyCheatData, error) {
	raw, err := r.loader.files.ReadFile(ctx, "core/cheats/"+world+".yaml")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var core map[string]struct {
		Ref       string          `yaml:"ref"`
		Strategy  []strategyEntry `yaml:"strategy"`
		Mechanics []struct {
			Enemies  enemyReferenceSources `yaml:"enemies"`
			ID       string                `yaml:"id"`
			Notes    []string              `yaml:"notes"`
			Schedule *CheatSchedule        `yaml:"schedule"`
		} `yaml:"mechanics"`
		// Read old published data during the migration as plain mechanics.
		Notes []string `yaml:"notes"`
	}
	if err = yaml.Unmarshal(raw, &core); err != nil {
		return nil, fmt.Errorf("parse cheats %s: %w", world, err)
	}
	names := map[string]map[string]string{}
	raw, err = r.loader.files.ReadFile(ctx, "i18n/"+r.language+"/cheats/"+world+".json")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		if err = json.Unmarshal(raw, &names); err != nil {
			return nil, err
		}
	}
	result := map[string]enemyCheatData{}
	for enemy, entry := range core {
		textOwner := enemy
		if entry.Ref != "" {
			target, ok := core[entry.Ref]
			if !ok || entry.Ref == enemy || target.Ref != "" {
				return nil, fmt.Errorf("cheats %s/%s: ref must point directly to an existing definition", world, enemy)
			}
			if len(entry.Mechanics) > 0 || len(entry.Strategy) > 0 || len(entry.Notes) > 0 {
				return nil, fmt.Errorf("cheats %s/%s: ref cannot be combined with own mechanics or strategy", world, enemy)
			}
			textOwner = entry.Ref
			entry = target
		}
		textFor := func(id string) string {
			text := names[textOwner][id]
			if text == "" {
				text = id
			}
			return text
		}
		data := enemyCheatData{}
		for _, strategy := range entry.Strategy {
			if strategy.ID == "" {
				return nil, fmt.Errorf("cheats %s/%s: empty strategy key", world, enemy)
			}
			note := CheatNote{ID: strategy.ID, Text: textFor(strategy.ID)}
			keys := make([]string, 0, len(strategy.Enemies))
			for key := range strategy.Enemies {
				keys = append(keys, key)
			}
			slices.Sort(keys)
			for _, key := range keys {
				ref := strategy.Enemies[key]
				if ref.Enemy == "" {
					return nil, fmt.Errorf("strategy %s: empty enemy reference", strategy.ID)
				}
				label := ""
				if ref.Label != "" {
					label = textFor(ref.Label)
				}
				note.Enemies = append(note.Enemies, CheatEnemyReference{Key: key, EnemyID: ref.Enemy, Label: label})
			}
			data.Strategy = append(data.Strategy, note)
		}
		for _, mechanic := range entry.Mechanics {
			if mechanic.ID == "" {
				return nil, fmt.Errorf("cheats %s/%s: mechanic id is required", world, enemy)
			}
			info := CheatMechanic{ID: mechanic.ID, Text: textFor(mechanic.ID)}
			info.Enemies, err = mechanic.Enemies.resolve(textFor)
			if err != nil {
				return nil, fmt.Errorf("mechanic %s: %w", mechanic.ID, err)
			}
			if mechanic.Schedule != nil {
				schedule := *mechanic.Schedule
				if schedule.Repeat < 0 || len(schedule.Events) == 0 {
					return nil, fmt.Errorf("cheats %s/%s: invalid schedule", world, enemy)
				}
				if schedule.Timing == "" {
					schedule.Timing = "start"
				}
				if schedule.Timing != "start" && schedule.Timing != "end" {
					return nil, fmt.Errorf("invalid schedule timing %q", schedule.Timing)
				}
				for i := range schedule.Events {
					event := &schedule.Events[i]
					if event.Round < 1 || (schedule.Repeat > 0 && event.Round > schedule.Repeat) || event.Text == "" {
						return nil, fmt.Errorf("invalid schedule event in %s/%s", world, enemy)
					}
					if event.Timing == "" {
						event.Timing = schedule.Timing
					}
					if event.Timing != "start" && event.Timing != "end" {
						return nil, fmt.Errorf("invalid event timing %q", event.Timing)
					}
					event.ID, event.Text = event.Text, textFor(event.Text)
				}
				info.Schedule = &schedule
			}
			for _, id := range mechanic.Notes {
				info.Notes = append(info.Notes, CheatNote{ID: id, Text: textFor(id)})
			}
			data.Mechanics = append(data.Mechanics, info)
		}
		if len(entry.Mechanics) == 0 {
			for _, id := range entry.Notes {
				data.Mechanics = append(data.Mechanics, CheatMechanic{ID: id, Text: textFor(id)})
			}
		}
		result[enemy] = data
	}
	return result, nil
}
