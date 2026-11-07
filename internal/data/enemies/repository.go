package enemies

import (
	"context"
	"log"
	"sort"
	"strings"
	"sync"

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

	enemies map[string]EnemyInfo
	worlds  map[string]map[string]EnemyInfo

	worldLoads map[string]*worldLoad
}

func NewRepository(
	language string,
	loader *RemoteLoader,
) *Repository {
	return &Repository{
		language: language,
		loader:   loader,
		worlds:   make(map[string]map[string]EnemyInfo),

		enemies: make(
			map[string]EnemyInfo,
		),

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
		r.enemies[mobID]

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

	if err != nil {
		return err
	}

	enemyInfos := make(
		map[string]EnemyInfo,
		len(coreFile),
	)

	for mobID, coreData := range coreFile {

		translation, translated :=
			translations[mobID]

		name := mobID

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
				Name: name,

				Affinities: cloneEnemyAffinities(
					coreData.Affinities,
				),
			}
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
	r.worlds[worldKey] = enemyInfos

	for mobID, info := range enemyInfos {
		r.enemies[mobID] = info
	}

	r.mu.Unlock()

	log.Printf(
		"[enemies] loaded %d enemies from %s",
		len(enemyInfos),
		worldKey,
	)

	return nil
}

type WorldEnemy struct {
	Key string
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
	r.mu.RLock()
	result := make([]WorldEnemy, 0, len(r.worlds[worldKey]))
	for key, info := range r.worlds[worldKey] {
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
	return result, nil
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
