package zones

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
)

type Repository struct {
	mu sync.RWMutex

	language string
	loader   *RemoteLoader

	// Welt-Key -> Zone-Key -> ZoneInfo
	worlds map[string]map[string]ZoneInfo

	coreWorlds       map[string]coreWorld
	translatedWorlds map[string]translatedWorld

	metadataOnce sync.Once
	metadataErr  error

	// Pro Welt ein eigener Ladezustand.
	worldLoads map[string]*worldLoad
}

type worldLoad struct {
	once sync.Once
	err  error
}

func NewRepository(
	language string,
	loader *RemoteLoader,
) *Repository {
	return &Repository{
		language: language,
		loader:   loader,

		worlds: make(
			map[string]map[string]ZoneInfo,
		),

		worldLoads: make(
			map[string]*worldLoad,
		),
	}
}

func (r *Repository) Resolve(
	ctx context.Context,
	zoneKey string,
) (ZoneInfo, bool, error) {
	if zoneKey == "" {
		return ZoneInfo{}, false, nil
	}

	worldKey :=
		WorldKey(zoneKey)

	if worldKey == "" {
		return ZoneInfo{}, false, nil
	}

	if err :=
		r.ensureWorld(
			ctx,
			worldKey,
		); err != nil {
		return ZoneInfo{}, false, err
	}

	r.mu.RLock()

	worldZones, exists :=
		r.worlds[worldKey]

	if !exists {
		r.mu.RUnlock()

		return ZoneInfo{}, false, nil
	}

	info, exists :=
		worldZones[zoneKey]

	r.mu.RUnlock()

	return info, exists, nil
}

func (r *Repository) ResolveOrFallback(
	ctx context.Context,
	zoneKey string,
) ZoneInfo {
	info, ok, err :=
		r.Resolve(
			ctx,
			zoneKey,
		)

	if err == nil && ok {
		return info
	}

	worldKey :=
		WorldKey(zoneKey)

	return ZoneInfo{
		Name:  zoneKey,
		World: worldKey,
		Image: "dungeons",
	}
}

func (r *Repository) ensureMetadata(
	ctx context.Context,
) error {
	r.metadataOnce.Do(
		func() {
			coreWorlds,
				translatedWorlds,
				err :=
				r.loader.LoadWorldMetadata(
					ctx,
					r.language,
				)

			if err != nil {
				r.metadataErr =
					err

				return
			}

			r.mu.Lock()

			r.coreWorlds =
				coreWorlds

			r.translatedWorlds =
				translatedWorlds

			r.mu.Unlock()

			log.Printf(
				"[zones] loaded world metadata (%d worlds)",
				len(coreWorlds),
			)
		},
	)

	return r.metadataErr
}

func (r *Repository) ensureWorld(
	ctx context.Context,
	worldKey string,
) error {
	if r.isWorldCached(worldKey) {
		return nil
	}

	load :=
		r.getWorldLoad(
			worldKey,
		)

	load.once.Do(
		func() {
			load.err =
				r.loadWorld(
					ctx,
					worldKey,
				)
		},
	)

	return load.err
}

func (r *Repository) getWorldLoad(
	worldKey string,
) *worldLoad {
	r.mu.Lock()
	defer r.mu.Unlock()

	load, exists :=
		r.worldLoads[worldKey]

	if exists {
		return load
	}

	load =
		&worldLoad{}

	r.worldLoads[worldKey] =
		load

	return load
}

func (r *Repository) isWorldCached(
	worldKey string,
) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists :=
		r.worlds[worldKey]

	return exists
}

func (r *Repository) loadWorld(
	ctx context.Context,
	worldKey string,
) error {
	if err :=
		r.ensureMetadata(ctx); err != nil {
		return fmt.Errorf(
			"load world metadata: %w",
			err,
		)
	}

	coreFile,
		translations,
		err :=
		r.loader.LoadWorldZones(
			ctx,
			r.language,
			worldKey,
		)

	if err != nil {
		return err
	}

	r.mu.RLock()

	worldCore,
		coreWorldExists :=
		r.coreWorlds[worldKey]

	worldTranslation,
		translatedWorldExists :=
		r.translatedWorlds[worldKey]

	r.mu.RUnlock()

	if !coreWorldExists {
		return fmt.Errorf(
			"missing world metadata in core/worlds.json: %s",
			worldKey,
		)
	}

	worldName :=
		worldKey

	if translatedWorldExists &&
		worldTranslation.Name != "" {

		worldName =
			worldTranslation.Name
	} else {
		log.Printf(
			"[zones] missing translated world name: %s",
			worldKey,
		)
	}

	worldZones :=
		make(
			map[string]ZoneInfo,
			len(coreFile.Zones),
		)

	for zoneKey, coreData := range coreFile.Zones {

		translation, translated :=
			translations[zoneKey]

		name := "???"
		sub := ""

		if translated {
			if translation.Name != "" {
				name =
					translation.Name
			}

			sub =
				translation.Sub
		} else {
			log.Printf(
				"[zones] missing %s translation: %s",
				r.language,
				zoneKey,
			)
		}

		worldZones[zoneKey] =
			ZoneInfo{
				Name: name,

				Sub: sub,

				World: worldName,

				Image: worldCore.Image,

				Type: coreData.Type,

				Parent: coreData.Parent,

				Instance: coreData.Instance,

				Repeatable: coreData.Repeatable,
			}
	}

	for zoneKey := range translations {

		if _, exists :=
			coreFile.Zones[zoneKey]; !exists {

			log.Printf(
				"[zones] translation without core zone: %s",
				zoneKey,
			)
		}
	}

	r.mu.Lock()

	r.worlds[worldKey] =
		worldZones

	r.mu.Unlock()

	log.Printf(
		"[zones] loaded %s (%d zones)",
		worldKey,
		len(worldZones),
	)

	return nil
}

func (r *Repository) CachedWorldCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.worlds)
}

func (r *Repository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.worlds =
		make(
			map[string]map[string]ZoneInfo,
		)

	r.worldLoads =
		make(
			map[string]*worldLoad,
		)
}

func WorldKey(
	zoneKey string,
) string {
	base, _, _ :=
		strings.Cut(
			zoneKey,
			"/",
		)

	switch {
	case strings.Contains(
		zoneKey,
		"WC_Catacombs",
	):
		return "Catacombs"

	case strings.Contains(
		zoneKey,
		"Selenopolis",
	):
		return "Selenopolis"

	case strings.HasPrefix(
		base,
		"G14",
	):
		return "Dungeons"

	case strings.HasPrefix(
		base,
		"Housing",
	):
		return "Housing"

	case strings.HasPrefix(
		base,
		"ThePhantomZoneWorld",
	):
		return "Minigames"

	default:
		return base
	}
}
