package zones

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"wizlink/rest"
)

const language = "de"

// Cached zones per world key.
var (
	cache = make(map[string]map[string]ZoneInfo)

	coreWorlds       map[string]coreWorld
	translatedWorlds map[string]translatedWorld
	worldsLoaded     bool

	mu sync.RWMutex
)

// Sprachunabhängige Welt-Metadaten aus core/worlds.json.
type coreWorld struct {
	Image string `json:"image"`
	Order int    `json:"order"`
	Type  string `json:"type"`
}

// Übersetzte Welt-Metadaten aus i18n/<lang>/worlds.json.
type translatedWorld struct {
	Name string `json:"name"`
}

// Sprachunabhängige Zonendatei aus core/zones/<world>.json.
type coreZoneFile struct {
	Zones map[string]coreZone `json:"zones"`
}

// Sprachunabhängige Eigenschaften einer Zone.
type coreZone struct {
	Type       string `json:"type,omitempty"`
	Parent     string `json:"parent,omitempty"`
	Instance   bool   `json:"instance,omitempty"`
	Repeatable bool   `json:"repeatable,omitempty"`
}

// Übersetzte Zonendatei aus i18n/<lang>/zones/<world>.json.
type translatedZoneFile struct {
	Zones map[string]translatedZone `json:"zones"`
}

// Sprachabhängige Eigenschaften einer Zone.
type translatedZone struct {
	Name string `json:"name"`
	Sub  string `json:"sub,omitempty"`
}

// Vollständig aufgelöste Zoneninformationen.
type ZoneInfo struct {
	Name  string `json:"name"`
	Sub   string `json:"sub,omitempty"`
	World string `json:"world"`
	Image string `json:"image"`

	Type       string `json:"type,omitempty"`
	Parent     string `json:"parent,omitempty"`
	Instance   bool   `json:"instance,omitempty"`
	Repeatable bool   `json:"repeatable,omitempty"`
}

// Lädt die Welt-Metadaten einmalig und cached sie.
func loadWorldMetadata() error {
	mu.RLock()
	loaded := worldsLoaded
	mu.RUnlock()

	if loaded {
		return nil
	}

	corePath := "core/worlds.json"
	translationPath := fmt.Sprintf(
		"i18n/%s/worlds.json",
		language,
	)

	coreRaw, err := rest.ReadFile(corePath)
	if err != nil {
		return fmt.Errorf("load %s: %w", corePath, err)
	}

	translationRaw, err := rest.ReadFile(translationPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", translationPath, err)
	}

	var loadedCoreWorlds map[string]coreWorld
	if err := json.Unmarshal(coreRaw, &loadedCoreWorlds); err != nil {
		return fmt.Errorf("parse %s: %w", corePath, err)
	}

	var loadedTranslatedWorlds map[string]translatedWorld
	if err := json.Unmarshal(
		translationRaw,
		&loadedTranslatedWorlds,
	); err != nil {
		return fmt.Errorf("parse %s: %w", translationPath, err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Ein anderer Goroutine-Aufruf könnte die Daten inzwischen geladen haben.
	if worldsLoaded {
		return nil
	}

	coreWorlds = loadedCoreWorlds
	translatedWorlds = loadedTranslatedWorlds
	worldsLoaded = true

	log.Printf(
		"[ZONES] loaded world metadata (%d worlds)",
		len(coreWorlds),
	)

	return nil
}

// Lädt und cached alle Zonen einer Welt.
func loadWorld(worldKey string) {
	mu.RLock()
	_, alreadyLoaded := cache[worldKey]
	mu.RUnlock()

	if alreadyLoaded {
		return
	}

	if err := loadWorldMetadata(); err != nil {
		log.Printf(
			"[ZONES] failed loading world metadata: %v",
			err,
		)
		cacheEmpty(worldKey)
		return
	}

	corePath := fmt.Sprintf(
		"core/zones/%s.json",
		worldKey,
	)

	translationPath := fmt.Sprintf(
		"i18n/%s/zones/%s.json",
		language,
		worldKey,
	)

	coreRaw, err := rest.ReadFile(corePath)
	if err != nil {
		log.Printf(
			"[ZONES] missing core zone file %s: %v",
			corePath,
			err,
		)
		cacheEmpty(worldKey)
		return
	}

	translationRaw, err := rest.ReadFile(translationPath)
	if err != nil {
		log.Printf(
			"[ZONES] missing translation file %s: %v",
			translationPath,
			err,
		)
		cacheEmpty(worldKey)
		return
	}

	var coreFile coreZoneFile
	if err := json.Unmarshal(coreRaw, &coreFile); err != nil {
		log.Printf(
			"[ZONES] invalid core zone file %s: %v",
			corePath,
			err,
		)
		cacheEmpty(worldKey)
		return
	}

	var translations translatedZoneFile
	if err := json.Unmarshal(
		translationRaw,
		&translations,
	); err != nil {
		log.Printf(
			"[ZONES] invalid translation file %s: %v",
			translationPath,
			err,
		)
		cacheEmpty(worldKey)
		return
	}

	mu.RLock()
	worldCore, coreWorldExists := coreWorlds[worldKey]
	worldTranslation, translatedWorldExists :=
		translatedWorlds[worldKey]
	mu.RUnlock()

	if !coreWorldExists {
		log.Printf(
			"[ZONES] missing world metadata in core/worlds.json: %s",
			worldKey,
		)
	}

	worldName := worldKey
	if translatedWorldExists && worldTranslation.Name != "" {
		worldName = worldTranslation.Name
	} else {
		log.Printf(
			"[ZONES] missing translated world name: %s",
			worldKey,
		)
	}

	zones := make(
		map[string]ZoneInfo,
		len(coreFile.Zones),
	)

	for zoneKey, coreData := range coreFile.Zones {
		translation, translated := translations.Zones[zoneKey]

		name := "???"
		sub := ""

		if translated {
			if translation.Name != "" {
				name = translation.Name
			}
			sub = translation.Sub
		} else {
			log.Printf(
				"[ZONES] missing %s translation: %s",
				language,
				zoneKey,
			)
		}

		zones[zoneKey] = ZoneInfo{
			Name:       name,
			Sub:        sub,
			World:      worldName,
			Image:      worldCore.Image,
			Type:       coreData.Type,
			Parent:     coreData.Parent,
			Instance:   coreData.Instance,
			Repeatable: coreData.Repeatable,
		}
	}

	// Hilft dabei, Übersetzungen zu erkennen, für die keine Core-Zone existiert.
	for zoneKey := range translations.Zones {
		if _, exists := coreFile.Zones[zoneKey]; !exists {
			log.Printf(
				"[ZONES] translation without core zone: %s",
				zoneKey,
			)
		}
	}

	mu.Lock()
	cache[worldKey] = zones
	mu.Unlock()

	log.Printf(
		"[ZONES] loaded %s (%d zones)",
		worldKey,
		len(zones),
	)
}

// Speichert einen leeren Cache-Eintrag, damit fehlerhafte Dateien nicht bei
// jedem Resolve-Aufruf erneut heruntergeladen werden.
func cacheEmpty(worldKey string) {
	mu.Lock()
	cache[worldKey] = map[string]ZoneInfo{}
	mu.Unlock()
}

// extractWorldKey derives the world key from a zone key.
func extractWorldKey(zoneKey string) string {
	base := strings.SplitN(zoneKey, "/", 2)[0]

	if strings.Contains(zoneKey, "WC_Catacombs") {
		return "Catacombs"
	}

	if strings.Contains(zoneKey, "Selenopolis") {
		return "Selenopolis"
	}

	switch {
	case strings.HasPrefix(base, "G14_DM"):
		return "Darkmoor"

	case strings.HasPrefix(base, "G14"):
		return "Dungeons"

	case strings.HasPrefix(base, "DD"):
		return "Dungeons"

	case strings.HasPrefix(base, "Housing"):
		return "Housing"

	case strings.HasPrefix(base, "ThePhantomZoneWorld"):
		return "Minigames"
	}

	// Fallback: Prefix vor dem ersten Unterstrich.
	if i := strings.Index(base, "_"); i != -1 {
		return base[:i]
	}

	return base
}

// Resolve gibt sichtbare Informationen für einen Zone-Key zurück.
func Resolve(
	zoneKey string,
) (
	name string,
	sub string,
	world string,
	image string,
) {
	worldKey := extractWorldKey(zoneKey)

	loadWorld(worldKey)

	mu.RLock()
	worldZones, worldExists := cache[worldKey]
	z, zoneExists := worldZones[zoneKey]
	mu.RUnlock()

	if worldExists && zoneExists {
		return z.Name, z.Sub, z.World, z.Image
	}

	reportMissingZone(zoneKey)

	return "???", "", "???", "dungeons"
}
