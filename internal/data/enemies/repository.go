package enemies

import (
	"context"
	"log"
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

	worldLoads map[string]*worldLoad
}

func NewRepository(
	language string,
	loader *RemoteLoader,
) *Repository {
	return &Repository{
		language: language,
		loader:   loader,

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

		name := "???"

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
