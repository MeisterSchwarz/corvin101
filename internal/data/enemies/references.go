package enemies

import (
	"corvin101/internal/data/drops"
	"corvin101/internal/data/items"
	"fmt"
	"reflect"
	"slices"
	"sort"
)

func validateEnemyReferences(core coreEnemyFile) error {
	for id, enemy := range core {
		if enemy.Mode != "" && enemy.Mode != "standard" && enemy.Mode != "challenge" {
			return fmt.Errorf("%s: invalid mode %q", id, enemy.Mode)
		}
		if enemy.VariantOf != "" {
			target, ok := core[enemy.VariantOf]
			if !ok || enemy.VariantOf == id || target.VariantOf != "" || target.Ref != "" || enemyMode(target.Mode) == enemyMode(enemy.Mode) {
				return fmt.Errorf("%s: invalid variant_of %s", id, enemy.VariantOf)
			}
		}
		if enemy.Ref == "" {
			continue
		}
		target, exists := core[enemy.Ref]
		if !exists {
			return fmt.Errorf("%s: missing ref target %s", id, enemy.Ref)
		}
		if enemy.Ref == id || target.Ref != "" {
			return fmt.Errorf("%s: ref must point directly to a full enemy, not %s", id, enemy.Ref)
		}
		if len(enemy.Schools) > 0 || len(enemy.Affinities) > 0 {
			return fmt.Errorf("%s: ref inherits schools and affinities; only locations may be added", id)
		}
	}
	return nil
}

func buildEnemyCatalog(core coreEnemyFile, variants map[string]EnemyInfo) map[string]EnemyInfo {
	result := make(map[string]EnemyInfo)
	keys := make([]string, 0, len(core))
	for key := range core {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, id := range keys {
		root := id
		if core[id].Ref != "" {
			root = core[id].Ref
		}
		info, exists := result[root]
		if !exists {
			info = variants[root]
			info.GameIDs = nil
			info.Locations = nil
			info.LocationNames = nil
			info.DropCategories = nil
		}
		variant := variants[id]
		for _, paragraph := range variant.Strategy {
			if !slices.ContainsFunc(info.Strategy, func(existing CheatNote) bool { return reflect.DeepEqual(existing, paragraph) }) {
				info.Strategy = append(cloneCheatNotes(info.Strategy), cloneCheatNotes([]CheatNote{paragraph})[0])
			}
		}
		info.GameIDs = append(info.GameIDs, id)
		info.Locations = append(info.Locations, variant.Locations...)
		info.DropCategories = mergeDropCategories(info.DropCategories, variant.DropCategories)
		for _, note := range variant.Cheats {
			if !slices.ContainsFunc(info.Cheats, func(existing CheatMechanic) bool {
				return reflect.DeepEqual(existing, note)
			}) {
				info.Cheats = append(cloneCheats(info.Cheats), cloneCheats([]CheatMechanic{note})[0])
			}
		}
		info.DropsUnavailable = info.DropsUnavailable || variant.DropsUnavailable
		result[root] = info
	}
	for key, info := range result {
		slices.Sort(info.Locations)
		info.Locations = slices.Compact(info.Locations)
		result[key] = info
	}
	return result
}

func mergeDropCategories(target, source []drops.Category) []drops.Category {
	result := drops.Clone(target)
	for _, category := range source {
		index := slices.IndexFunc(result, func(v drops.Category) bool { return v.Key == category.Key })
		if index < 0 {
			result = append(result, drops.Clone([]drops.Category{category})[0])
			continue
		}
		if category.Gold != nil {
			if result[index].Gold == nil {
				gold := *category.Gold
				result[index].Gold = &gold
			} else {
				result[index].Gold.Min = min(result[index].Gold.Min, category.Gold.Min)
				result[index].Gold.Max = max(result[index].Gold.Max, category.Gold.Max)
			}
		}
		for _, item := range category.Items {
			if !slices.ContainsFunc(result[index].Items, func(v items.Info) bool { return v.ID == item.ID }) {
				result[index].Items = append(result[index].Items, item)
			}
		}
	}
	for i := range result {
		sort.Slice(result[i].Items, func(a, b int) bool { return result[i].Items[a].Name < result[i].Items[b].Name })
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
}

// InCombat accepts every game ID grouped under this one bestiary entry.
func (e WorldEnemy) InCombat(ids map[string]bool) bool {
	if ids[e.Key] {
		return true
	}
	for _, id := range e.GameIDs {
		if ids[id] {
			return true
		}
	}
	return false
}
