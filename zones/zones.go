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

// Cache mit bereits geladenen Zonen pro Welt.
var (
	// Welt-Key -> (Zone-Key -> ZoneInfo)
	cache = make(map[string]map[string]ZoneInfo)

	// Stellt sicher, dass jede Welt nur einmal geladen wird.
	worldOnce = make(map[string]*sync.Once)

	// Welt-Metadaten aus core/worlds.json.
	coreWorlds map[string]coreWorld

	// Übersetzte Weltnamen aus i18n/<lang>/worlds.json.
	translatedWorlds map[string]translatedWorld

	// Lädt die Welt-Metadaten genau einmal.
	worldsOnce sync.Once
	worldsErr  error

	// Schützt alle gemeinsam genutzten Maps.
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

// Übersetzungen einer Welt.
// Der JSON-Inhalt besteht direkt aus einer Map von Zone-Key auf Übersetzung.
type translatedZoneFile map[string]translatedZone

// Lokalisierter Name einer Zone inklusive optionalem Unterbereich.
type translatedZone struct {
	Name string
	Sub  string
}

// Unterstützt beide Übersetzungsformate:
//
// "Basislager"
//
// sowie:
//
// ["Die Säulenhalle", "Außenbereich"]
func (z *translatedZone) UnmarshalJSON(data []byte) error {
	var name string

	if err := json.Unmarshal(data, &name); err == nil {
		z.Name = name
		z.Sub = ""
		return nil
	}

	var values []string
	if err := json.Unmarshal(data, &values); err != nil {
		return fmt.Errorf(
			"translation must be a string or string array: %w",
			err,
		)
	}

	switch len(values) {
	case 1:
		z.Name = values[0]
		z.Sub = ""
		return nil

	case 2:
		z.Name = values[0]
		z.Sub = values[1]
		return nil

	default:
		return fmt.Errorf(
			"translation array must contain 1 or 2 strings, got %d",
			len(values),
		)
	}
}

// Vollständig zusammengeführte Informationen einer Zone.
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

// Lädt die Welt-Metadaten einmalig
func loadWorldMetadata() error {
	worldsOnce.Do(func() {
		corePath := "core/worlds.json"
		translationPath := fmt.Sprintf(
			"i18n/%s/worlds.json",
			language,
		)

		coreRaw, err := rest.ReadFile(corePath)
		if err != nil {
			worldsErr = fmt.Errorf("load %s: %w", corePath, err)
			return
		}

		translationRaw, err := rest.ReadFile(translationPath)
		if err != nil {
			worldsErr = fmt.Errorf("load %s: %w", translationPath, err)
			return
		}

		var loadedCoreWorlds map[string]coreWorld
		if err := json.Unmarshal(coreRaw, &loadedCoreWorlds); err != nil {
			worldsErr = fmt.Errorf("parse %s: %w", corePath, err)
			return
		}

		var loadedTranslatedWorlds map[string]translatedWorld
		if err := json.Unmarshal(
			translationRaw,
			&loadedTranslatedWorlds,
		); err != nil {
			worldsErr = fmt.Errorf(
				"parse %s: %w",
				translationPath,
				err,
			)
			return
		}

		coreWorlds = loadedCoreWorlds
		translatedWorlds = loadedTranslatedWorlds

		log.Printf(
			"[ZONES] loaded world metadata (%d worlds)",
			len(coreWorlds),
		)
	})

	return worldsErr
}

// Lädt die Zonen einer Welt einmalig
func loadWorld(worldKey string) {
	mu.Lock()

	once, exists := worldOnce[worldKey]
	if !exists {
		once = &sync.Once{}
		worldOnce[worldKey] = once
	}

	mu.Unlock()

	once.Do(func() {
		zones, err := readWorld(worldKey)
		if err != nil {
			log.Printf(
				"[ZONES] failed loading %s: %v",
				worldKey,
				err,
			)

			zones = map[string]ZoneInfo{}
		}

		mu.Lock()
		cache[worldKey] = zones
		mu.Unlock()
	})
}

// Liest und kombiniert alle Daten einer Welt aus Core- und Übersetzungsdateien.
func readWorld(worldKey string) (map[string]ZoneInfo, error) {
	if err := loadWorldMetadata(); err != nil {
		return nil, fmt.Errorf("load world metadata: %w", err)
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
		return nil, fmt.Errorf("load %s: %w", corePath, err)
	}

	translationRaw, err := rest.ReadFile(translationPath)
	if err != nil {
		return nil, fmt.Errorf(
			"load %s: %w",
			translationPath,
			err,
		)
	}

	var coreFile coreZoneFile
	if err := json.Unmarshal(coreRaw, &coreFile); err != nil {
		return nil, fmt.Errorf("parse %s: %w", corePath, err)
	}

	var translations translatedZoneFile
	if err := json.Unmarshal(
		translationRaw,
		&translations,
	); err != nil {
		return nil, fmt.Errorf(
			"parse %s: %w",
			translationPath,
			err,
		)
	}

	mu.RLock()
	worldCore, coreWorldExists := coreWorlds[worldKey]
	worldTranslation, translatedWorldExists :=
		translatedWorlds[worldKey]
	mu.RUnlock()

	if !coreWorldExists {
		return nil, fmt.Errorf(
			"missing world metadata in core/worlds.json: %s",
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
		translation, translated := translations[zoneKey]

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

	for zoneKey := range translations {
		if _, exists := coreFile.Zones[zoneKey]; !exists {
			log.Printf(
				"[ZONES] translation without core zone: %s",
				zoneKey,
			)
		}
	}

	log.Printf(
		"[ZONES] loaded %s (%d zones)",
		worldKey,
		len(zones),
	)

	return zones, nil
}

// Ermittelt den Welt-Key aus einem Zone-Key
func extractWorldKey(zoneKey string) string {
	base, _, _ := strings.Cut(zoneKey, "/")

	switch {
	case strings.Contains(zoneKey, "WC_Catacombs"):
		return "Catacombs"

	case strings.Contains(zoneKey, "Selenopolis"):
		return "Selenopolis"

	case strings.HasPrefix(base, "G14"):
		return "Dungeons"

	case strings.HasPrefix(base, "Housing"):
		return "Housing"

	case strings.HasPrefix(base, "ThePhantomZoneWorld"):
		return "Minigames"

	default:
		return base
	}
}

// Public Wrapper für extractWorldKey
func WorldKey(zoneKey string) string {
	return extractWorldKey(zoneKey)
}

// Liefert die aufgelösten Informationen zu einem Zone-Key
func Resolve(zoneKey string) (ZoneInfo, bool) {
	if zoneKey == "" {
		return ZoneInfo{}, false
	}

	worldKey := extractWorldKey(zoneKey)
	loadWorld(worldKey)

	mu.RLock()
	info, exists := cache[worldKey][zoneKey]
	mu.RUnlock()

	return info, exists
}

// ResolveOrFallback liefert Zone-Informationen oder einen einfachen Fallback,
// falls die Zone unbekannt ist
func ResolveOrFallback(zoneKey string) ZoneInfo {
	if info, ok := Resolve(zoneKey); ok {
		return info
	}

	return ZoneInfo{
		Name:  zoneKey,
		World: zoneKey,
		Image: "dungeons",
	}
}
