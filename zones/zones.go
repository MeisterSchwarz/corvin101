package zones

import (
	"embed"
	"encoding/json"
	"log"
	"strings"
	"sync"
)

//go:embed *.json
var fs embed.FS

// Cached zones per world key
var (
	cache = make(map[string]map[string]ZoneInfo)
	mu    sync.RWMutex
)

// World file schema
type worldFile struct {
	Meta struct {
		World string `json:"world"`
		Image string `json:"image"`
	} `json:"meta"`
	Zones map[string]struct {
		Name string `json:"name"`
		Sub  string `json:"sub"`
	} `json:"zones"`
}

// Resolved zone information
type ZoneInfo struct {
	Name  string `json:"name"`
	Sub   string `json:"sub"`
	World string `json:"world"`
	Image string `json:"image"`
}

// loads and caches zones for a world
func loadWorld(worldKey string) {
	mu.RLock()
	_, ok := cache[worldKey]
	mu.RUnlock()
	if ok {
		return
	}

	filename := worldKey + ".json"
	raw, err := fs.ReadFile(filename)
	if err != nil {
		log.Printf("[ZONES] missing world file: %s", worldKey)
		cacheEmpty(worldKey)
		return
	}

	var wf worldFile
	if err := json.Unmarshal(raw, &wf); err != nil {
		log.Printf("[ZONES] invalid world file %s: %v", filename, err)
		cacheEmpty(worldKey)
		return
	}

	zones := make(map[string]ZoneInfo, len(wf.Zones))
	for key, z := range wf.Zones {
		zones[key] = ZoneInfo{
			Name:  z.Name,
			Sub:   z.Sub,
			World: wf.Meta.World,
			Image: wf.Meta.Image,
		}
	}

	mu.Lock()
	cache[worldKey] = zones
	mu.Unlock()

	log.Printf("[ZONES] loaded %s (%d zones)", worldKey, len(zones))
}

// stores an empty world entry
func cacheEmpty(worldKey string) {
	mu.Lock()
	cache[worldKey] = map[string]ZoneInfo{}
	mu.Unlock()
}

// extractWorldKey derives the world key from a zone key
func extractWorldKey(zoneKey string) string {
	base := strings.SplitN(zoneKey, "/", 2)[0]

	switch {
	case strings.HasPrefix(base, "G14"):
		return "Dungeons"
	case strings.HasPrefix(base, "DD"):
		return "Dungeons"

	case strings.HasPrefix(base, "Housing"):
		return "Housing"

	case strings.HasPrefix(base, "ThePhantomZoneWorld"):
		return "Minigames"

	case strings.HasPrefix(base, "WL_"):
		return "Wallaru"
	}

	// Fallback: prefix before first underscore
	if i := strings.Index(base, "_"); i != -1 {
		return base[:i]
	}

	return base
}

// returns display data for a zone key
func Resolve(zoneKey string) (name, sub, world, image string) {
	worldKey := extractWorldKey(zoneKey)

	loadWorld(worldKey)

	mu.RLock()
	z, ok := cache[worldKey][zoneKey]
	mu.RUnlock()

	if ok {
		return z.Name, z.Sub, z.World, z.Image
	}

	reportMissingZone(zoneKey)
	return "???", "", "???", "dungeons"
}
