package enemies

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"slices"
	"sort"
	"strings"
	"sync"

	"corvin101/internal/data/drops"
	"corvin101/internal/data/zones"
)

type worldLoad struct {
	once sync.Once
	err  error
}

type Repository struct {
	mu sync.RWMutex

	language string
	loader   *RemoteLoader
	zones    *zones.Repository
	drops    *drops.Repository

	variants map[string]map[string]EnemyInfo
	worlds   map[string]map[string]EnemyInfo

	worldLoads map[string]*worldLoad
}

func NewRepository(
	language string,
	loader *RemoteLoader,
	zoneRepositories ...*zones.Repository,
) *Repository {
	zoneRepository := zones.NewRepository(language, zones.NewRemoteLoader(loader.files))
	if len(zoneRepositories) > 0 && zoneRepositories[0] != nil {
		zoneRepository = zoneRepositories[0]
	}
	return &Repository{
		language: language,
		loader:   loader,
		zones:    zoneRepository,
		drops:    drops.NewRepository(loader.files, language),
		worlds:   make(map[string]map[string]EnemyInfo),

		variants: make(map[string]map[string]EnemyInfo),

		worldLoads: make(
			map[string]*worldLoad,
		),
	}
}

func (r *Repository) Resolve(
	ctx context.Context,
	zoneKey string,
	mobID string,
) (EnemyInfo, bool, error) {
	if zoneKey == "" ||
		mobID == "" {

		return EnemyInfo{}, false, nil
	}

	worldKey := zones.WorldKey(
		zoneKey,
	)

	if worldKey == "" {
		return EnemyInfo{}, false, nil
	}

	if err := r.ensureWorld(
		ctx,
		worldKey,
	); err != nil {
		return EnemyInfo{}, false, err
	}

	r.mu.RLock()

	info, exists :=
		r.variants[worldKey][mobID]
	info.Schools = slices.Clone(info.Schools)
	info.Heads = cloneHeads(info.Heads)
	info.Cheats = cloneCheats(info.Cheats)
	info.Strategy = cloneCheatNotes(info.Strategy)
	info.GameIDs = slices.Clone(info.GameIDs)
	info.Locations = slices.Clone(info.Locations)
	info.LocationNames = slices.Clone(info.LocationNames)
	info.DropCategories = drops.Clone(info.DropCategories)

	r.mu.RUnlock()

	return info, exists, nil
}

func (r *Repository) ensureWorld(
	ctx context.Context,
	worldKey string,
) error {
	r.mu.Lock()

	load, exists :=
		r.worldLoads[worldKey]

	if !exists {
		load = &worldLoad{}

		r.worldLoads[worldKey] = load
	}

	r.mu.Unlock()

	load.once.Do(
		func() {
			load.err = r.loadWorld(
				ctx,
				worldKey,
			)
		},
	)
	if load.err != nil {
		r.mu.Lock()
		if r.worldLoads[worldKey] == load {
			delete(r.worldLoads, worldKey)
		}
		r.mu.Unlock()
	}

	return load.err
}

func (r *Repository) loadWorld(
	ctx context.Context,
	worldKey string,
) error {
	coreFile, translations, err :=
		r.loader.LoadEnemies(
			ctx,
			r.language,
			worldKey,
		)

	// Not every world has enemy data yet. Cache this as an empty catalog.
	if errors.Is(err, fs.ErrNotExist) {
		r.mu.Lock()
		r.worlds[worldKey] = make(map[string]EnemyInfo)
		r.variants[worldKey] = make(map[string]EnemyInfo)
		r.mu.Unlock()
		return nil
	}
	if err != nil {
		return err
	}
	if err := validateEnemyReferences(coreFile); err != nil {
		return fmt.Errorf("enemies %s: %w", worldKey, err)
	}

	enemyInfos := make(
		map[string]EnemyInfo,
		len(coreFile),
	)
	worldDrops, dropErr := r.drops.LoadWorld(ctx, worldKey)
	worldCheats, cheatErr := r.loadCheats(ctx, worldKey)
	if cheatErr != nil {
		log.Printf("[cheats] load %s: %v", worldKey, cheatErr)
	}
	if dropErr != nil {
		log.Printf("[drops] load %s: %v", worldKey, dropErr)
	}

	encounters, err := r.loadEncounters(ctx, worldKey)
	if err != nil {
		return fmt.Errorf("encounters %s: %w", worldKey, err)
	}
	for mobID, coreData := range coreFile {
		root := mobID
		if coreData.Ref != "" {
			root = coreData.Ref
		}
		base := coreFile[root]
		cheats, ownCheats := worldCheats[mobID]
		if !ownCheats {
			cheats = worldCheats[root]
		}
		groupKey := encounterForEnemy(encounters, mobID)
		if groupKey != "" {
			if shared, ok := worldCheats[groupKey]; ok {
				cheats = shared
			}
		}
		enemyDrops := worldDrops[mobID]
		if groupKey != "" {
			enemyDrops = mergeDropCategories(enemyDrops, worldDrops[groupKey])
		}

		translation, translated :=
			translations[root]

		name := root

		if translated &&
			translation.Name != "" {

			name = translation.Name
		} else {
			log.Printf(
				"[enemies] missing %s translation: %s",
				r.language,
				mobID,
			)
		}

		enemyInfos[mobID] =
			EnemyInfo{
				Mode:             enemyMode(base.Mode),
				VariantOf:        base.VariantOf,
				Cheats:           cloneCheats(cheats.Mechanics),
				Strategy:         cloneCheatNotes(cheats.Strategy),
				GameIDs:          []string{mobID},
				Name:             name,
				Schools:          slices.Clone(base.Schools),
				Locations:        slices.Clone(coreData.Locations),
				LocationNames:    r.locationNames(ctx, coreData.Locations),
				DropCategories:   drops.Clone(enemyDrops),
				DropsUnavailable: dropErr != nil,

				Affinities: cloneEnemyAffinities(
					base.Affinities,
				),
			}
	}
	for id, info := range enemyInfos {
		referenceTexts := append([]CheatNote(nil), info.Strategy...)
		for _, mechanic := range info.Cheats {
			referenceTexts = append(referenceTexts, CheatNote{Enemies: mechanic.Enemies})
		}
		for i := range referenceTexts {
			for j := range referenceTexts[i].Enemies {
				ref := &referenceTexts[i].Enemies[j]
				target, ok := enemyInfos[ref.EnemyID]
				if !ok {
					return fmt.Errorf("strategy %s/%s: unknown enemy %s", worldKey, id, ref.EnemyID)
				}
				family := ref.EnemyID
				if target.VariantOf != "" {
					family = target.VariantOf
				}
				if target.Mode != info.Mode {
					for candidate, variant := range enemyInfos {
						if (candidate == family || variant.VariantOf == family) && variant.Mode == info.Mode {
							ref.EnemyID = candidate
							target = variant
							break
						}
					}
				}
				ref.FamilyID = family
				if encounter := encounterForEnemy(encounters, ref.EnemyID); encounter != "" {
					ref.HeadID = family
					ref.FamilyID = encounter
					if encounters[encounter].VariantOf != "" {
						ref.FamilyID = encounters[encounter].VariantOf
					}
				}
				ref.Mode = target.Mode
				ref.Schools = slices.Clone(target.Schools)
				if ref.Label == "" {
					ref.Label = target.Name
				}
			}
		}
		enemyInfos[id] = info
	}
	catalog := buildEnemyCatalog(coreFile, enemyInfos)
	if err := groupEncounters(encounters, catalog); err != nil {
		return err
	}
	for key, info := range catalog {
		info.LocationNames = r.locationNames(ctx, info.Locations)
		catalog[key] = info
	}

	for mobID := range translations {

		if _, exists :=
			coreFile[mobID]; !exists {

			log.Printf(
				"[enemies] translation without core enemy: %s",
				mobID,
			)
		}
	}

	r.mu.Lock()
	r.worlds[worldKey] = catalog

	r.variants[worldKey] = enemyInfos

	r.mu.Unlock()

	log.Printf(
		"[enemies] loaded %d enemies from %s",
		len(catalog),
		worldKey,
	)

	return nil
}

type WorldEnemy struct {
	Key        string
	VariantKey string
	EnemyInfo
}

// ListWorld returns only enemies registered in this world's core file.
func (r *Repository) ListWorld(ctx context.Context, worldKey string) ([]WorldEnemy, error) {
	if worldKey == "" {
		return nil, nil
	}
	if err := r.ensureWorld(ctx, worldKey); err != nil {
		return nil, err
	}
	result, _ := r.CachedWorld(worldKey)
	return result, nil
}

// CachedWorld never performs network I/O. A cached empty world is also a hit.
func (r *Repository) CachedWorld(worldKey string) ([]WorldEnemy, bool) {
	r.mu.RLock()
	if _, ok := r.worlds[worldKey]; !ok {
		r.mu.RUnlock()
		return nil, false
	}
	result := make([]WorldEnemy, 0, len(r.worlds[worldKey]))
	for key, info := range r.worlds[worldKey] {
		info.Heads = cloneHeads(info.Heads)
		info.Cheats = cloneCheats(info.Cheats)
		info.Strategy = cloneCheatNotes(info.Strategy)
		info.GameIDs = slices.Clone(info.GameIDs)
		info.Schools = slices.Clone(info.Schools)
		info.Locations = slices.Clone(info.Locations)
		info.LocationNames = slices.Clone(info.LocationNames)
		info.DropCategories = drops.Clone(info.DropCategories)
		info.Affinities = cloneEnemyAffinities(info.Affinities)
		result = append(result, WorldEnemy{Key: key, EnemyInfo: info})
	}
	r.mu.RUnlock()
	sort.Slice(result, func(i, j int) bool {
		a, b := strings.ToLower(result[i].Name), strings.ToLower(result[j].Name)
		if a == b {
			return result[i].Key < result[j].Key
		}
		return a < b
	})
	return result, true
}

func cloneEnemyAffinities(
	source map[School]int,
) map[School]int {
	if len(source) == 0 {
		return nil
	}

	result := make(
		map[School]int,
		len(source),
	)

	for school, value := range source {
		result[school] = value
	}

	return result
}

func (r *Repository) locationNames(ctx context.Context, keys []string) []string {
	names := make([]string, 0, len(keys))
	for _, key := range keys {
		info, found, err := r.zones.Resolve(ctx, key)
		name := key
		if err != nil {
			log.Printf("[enemies] resolve location %s: %v", key, err)
		}
		if found && err == nil {
			name = info.Name
			if info.Sub != "" && info.Sub != info.Name {
				name += " – " + info.Sub
			}
		}
		names = append(names, name)
	}
	return names
}
