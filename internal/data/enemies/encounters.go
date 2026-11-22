package enemies

import (
	"context"
	"corvin101/internal/data/drops"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/fs"
	"slices"
)

type encounterDefinition struct {
	Mode      string   `yaml:"mode,omitempty"`
	VariantOf string   `yaml:"variant_of,omitempty"`
	Enemies   []string `yaml:"enemies"`
	Name      string   `yaml:"-"`
}

func (r *Repository) loadEncounters(ctx context.Context, world string) (map[string]encounterDefinition, error) {
	raw, err := r.loader.files.ReadFile(ctx, "core/encounters/"+world+".yaml")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var nodes map[string]yaml.Node
	if err = yaml.Unmarshal(raw, &nodes); err != nil {
		return nil, err
	}
	result := map[string]encounterDefinition{}
	for id, node := range nodes {
		if node.Kind != yaml.MappingNode {
			continue
		}
		var def encounterDefinition
		if err = node.Decode(&def); err != nil {
			return nil, err
		}
		if len(def.Enemies) > 0 {
			def.Name = id
			result[id] = def
		}
	}
	// Legacy encounter documents contain zone-keyed arrays of rules.
	if len(result) == 0 {
		return result, nil
	}
	raw, err = r.loader.files.ReadFile(ctx, "i18n/"+r.language+"/encounters/"+world+".json")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		var names map[string]translatedEnemy
		if err = json.Unmarshal(raw, &names); err != nil {
			return nil, err
		}
		for id, def := range result {
			if names[id].Name != "" {
				def.Name = names[id].Name
			}
			result[id] = def
		}
	}
	return result, nil
}
func encounterForEnemy(defs map[string]encounterDefinition, id string) string {
	for key, def := range defs {
		if slices.Contains(def.Enemies, id) {
			return key
		}
	}
	return ""
}

func groupEncounters(defs map[string]encounterDefinition, catalog map[string]EnemyInfo) error {
	for id, def := range defs {
		if enemyMode(def.Mode) != "standard" && enemyMode(def.Mode) != "challenge" {
			return fmt.Errorf("encounter %s: invalid mode", id)
		}
		if def.VariantOf != "" {
			base, ok := defs[def.VariantOf]
			if !ok || def.VariantOf == id || base.VariantOf != "" || enemyMode(base.Mode) == enemyMode(def.Mode) {
				return fmt.Errorf("encounter %s: invalid variant_of", id)
			}
		}
	}
	used := map[string]string{}
	for id, def := range defs {
		if _, exists := catalog[id]; exists {
			return fmt.Errorf("encounter key collides with enemy %s", id)
		}
		group := EnemyInfo{Name: def.Name, Mode: enemyMode(def.Mode), VariantOf: def.VariantOf}
		for _, key := range def.Enemies {
			info, ok := catalog[key]
			if !ok {
				return fmt.Errorf("encounter %s: unknown enemy %s", id, key)
			}
			if previous, ok := used[key]; ok {
				return fmt.Errorf("enemy %s occurs in encounters %s and %s", key, previous, id)
			}
			used[key] = id
			if enemyMode(info.Mode) != group.Mode {
				return fmt.Errorf("encounter %s: mixed modes", id)
			}
			group.Heads = append(group.Heads, EncounterHead{Key: key, EnemyInfo: info})
			group.GameIDs = append(group.GameIDs, key)
			group.GameIDs = append(group.GameIDs, info.GameIDs...)
			group.Locations = append(group.Locations, info.Locations...)
			group.DropCategories = mergeDropCategories(group.DropCategories, info.DropCategories)
			group.DropsUnavailable = group.DropsUnavailable || info.DropsUnavailable
			for _, m := range info.Cheats {
				if !slices.ContainsFunc(group.Cheats, func(v CheatMechanic) bool { return v.ID == m.ID }) {
					group.Cheats = append(group.Cheats, cloneCheats([]CheatMechanic{m})[0])
				}
			}
			for _, note := range info.Strategy {
				if !slices.ContainsFunc(group.Strategy, func(v CheatNote) bool { return v.ID == note.ID }) {
					group.Strategy = append(group.Strategy, cloneCheatNotes([]CheatNote{note})[0])
				}
			}
		}
		slices.Sort(group.Locations)
		group.Locations = slices.Compact(group.Locations)
		for _, key := range def.Enemies {
			delete(catalog, key)
		}
		catalog[id] = group
	}
	return nil
}

func cloneHeads(source []EncounterHead) []EncounterHead {
	result := slices.Clone(source)
	for i := range result {
		info := &result[i].EnemyInfo
		info.Schools = slices.Clone(info.Schools)
		info.GameIDs = slices.Clone(info.GameIDs)
		info.Locations = slices.Clone(info.Locations)
		info.LocationNames = slices.Clone(info.LocationNames)
		info.Affinities = cloneEnemyAffinities(info.Affinities)
		info.Cheats = cloneCheats(info.Cheats)
		info.Strategy = cloneCheatNotes(info.Strategy)
		info.DropCategories = drops.Clone(info.DropCategories)
	}
	return result
}
