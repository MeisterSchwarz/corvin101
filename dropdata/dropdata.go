package dropdata

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"wizlink/rest"
)

const language = "de"

var (
	// Geladene Gegnerdaten, gruppiert nach Welt.
	worldCache = make(map[string]map[string]EnemyInfo)

	// Verhindert paralleles mehrfaches Laden derselben Welt.
	worldLoading = make(map[string]chan struct{})

	// Itemübersetzungen werden einmalig geladen.
	itemNames        = make(map[string]string)
	loadedItemFiles  = make(map[string]bool)
	loadingItemFiles = make(map[string]chan struct{})

	mu sync.RWMutex
)

var itemFiles = map[string]string{
	"rg": "reagents",
	"ht": "hats",
	"rb": "robes",
	"bt": "boots",
	"at": "athames",
	"wd": "wands",
	"dk": "decks",
	"sn": "snacks",
	"jw": "jewels",
	"tc": "treasurecards",
}

// core/enemies/<world>.json
type coreEnemyFile struct {
	Enemies map[string]coreEnemy `json:"enemies"`
}

type coreEnemy struct {
	Zones []string `json:"zones,omitempty"`
}

// core/drops/<world>.json
type coreDropFile struct {
	Drops map[string][]string `json:"drops"`
}

// i18n/<lang>/enemies/<world>.json
type translatedEnemyFile map[string]string

// Aufgelöstes Item für die Weboberfläche.
type ItemInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Vollständig aufgelöste Gegnerdaten.
type EnemyInfo struct {
	ID    string     `json:"id"`
	Name  string     `json:"name"`
	Zones []string   `json:"zones,omitempty"`
	Items []ItemInfo `json:"items"`
}

func itemFileForID(itemID string) (string, bool) {
	prefix, _, found := strings.Cut(itemID, "_")
	if !found || prefix == "" {
		return "", false
	}

	fileName, ok := itemFiles[prefix]
	return fileName, ok
}

func loadItemFile(fileName string) error {
	mu.Lock()

	if loadedItemFiles[fileName] {
		mu.Unlock()
		return nil
	}

	if wait, loading := loadingItemFiles[fileName]; loading {
		mu.Unlock()
		<-wait

		mu.RLock()
		loaded := loadedItemFiles[fileName]
		mu.RUnlock()

		if !loaded {
			return fmt.Errorf(
				"item file %s could not be loaded",
				fileName,
			)
		}

		return nil
	}

	wait := make(chan struct{})
	loadingItemFiles[fileName] = wait
	mu.Unlock()

	defer func() {
		mu.Lock()
		close(wait)
		delete(loadingItemFiles, fileName)
		mu.Unlock()
	}()

	path := fmt.Sprintf(
		"i18n/%s/items/%s.json",
		language,
		fileName,
	)

	raw, err := rest.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load %s: %w", path, err)
	}

	var loadedItems map[string]string

	if err := json.Unmarshal(raw, &loadedItems); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	mu.Lock()
	defer mu.Unlock()

	if loadedItemFiles[fileName] {
		return nil
	}

	for itemID, name := range loadedItems {
		if existing, exists := itemNames[itemID]; exists &&
			existing != name {
			return fmt.Errorf(
				"duplicate item ID %q in %s",
				itemID,
				path,
			)
		}

		itemNames[itemID] = name
	}

	loadedItemFiles[fileName] = true

	log.Printf(
		"[DROPS] loaded item file %s (%d items)",
		fileName,
		len(loadedItems),
	)

	return nil
}

func loadRequiredItemFiles(dropFile coreDropFile) error {
	requiredFiles := make(map[string]struct{})

	for _, itemIDs := range dropFile.Drops {
		for _, itemID := range itemIDs {
			fileName, ok := itemFileForID(itemID)
			if !ok {
				log.Printf(
					"[DROPS] unknown item prefix: %s",
					itemID,
				)
				continue
			}

			requiredFiles[fileName] = struct{}{}
		}
	}

	for fileName := range requiredFiles {
		if err := loadItemFile(fileName); err != nil {
			return err
		}
	}

	return nil
}

func loadWorld(worldKey string) {
	if worldKey == "" {
		return
	}

	mu.Lock()

	if _, alreadyLoaded := worldCache[worldKey]; alreadyLoaded {
		mu.Unlock()
		return
	}

	if wait, loading := worldLoading[worldKey]; loading {
		mu.Unlock()
		<-wait
		return
	}

	wait := make(chan struct{})
	worldLoading[worldKey] = wait
	mu.Unlock()

	defer func() {
		mu.Lock()
		close(wait)
		delete(worldLoading, worldKey)
		mu.Unlock()
	}()

	coreEnemyPath := fmt.Sprintf(
		"core/enemies/%s.json",
		worldKey,
	)

	coreDropPath := fmt.Sprintf(
		"core/drops/%s.json",
		worldKey,
	)

	translationPath := fmt.Sprintf(
		"i18n/%s/enemies/%s.json",
		language,
		worldKey,
	)

	coreEnemyRaw, err := rest.ReadFile(coreEnemyPath)
	if err != nil {
		log.Printf(
			"[DROPS] missing enemy file %s: %v",
			coreEnemyPath,
			err,
		)

		cacheEmpty(worldKey)
		return
	}

	coreDropRaw, err := rest.ReadFile(coreDropPath)
	if err != nil {
		log.Printf(
			"[DROPS] missing drop file %s: %v",
			coreDropPath,
			err,
		)

		cacheEmpty(worldKey)
		return
	}

	translationRaw, err := rest.ReadFile(translationPath)
	if err != nil {
		log.Printf(
			"[DROPS] missing enemy translation file %s: %v",
			translationPath,
			err,
		)

		cacheEmpty(worldKey)
		return
	}

	var enemyFile coreEnemyFile
	if err := json.Unmarshal(coreEnemyRaw, &enemyFile); err != nil {
		log.Printf(
			"[DROPS] invalid enemy file %s: %v",
			coreEnemyPath,
			err,
		)

		cacheEmpty(worldKey)
		return
	}

	var dropFile coreDropFile
	if err := json.Unmarshal(coreDropRaw, &dropFile); err != nil {
		log.Printf(
			"[DROPS] invalid drop file %s: %v",
			coreDropPath,
			err,
		)

		cacheEmpty(worldKey)
		return
	}

	var translations translatedEnemyFile
	if err := json.Unmarshal(
		translationRaw,
		&translations,
	); err != nil {
		log.Printf(
			"[DROPS] invalid enemy translation file %s: %v",
			translationPath,
			err,
		)

		cacheEmpty(worldKey)
		return
	}

	if err := loadRequiredItemFiles(dropFile); err != nil {
		log.Printf(
			"[DROPS] failed loading item translations for %s: %v",
			worldKey,
			err,
		)

		cacheEmpty(worldKey)
		return
	}

	mu.RLock()

	loadedItemNames := make(
		map[string]string,
		len(itemNames),
	)

	for itemID, name := range itemNames {
		loadedItemNames[itemID] = name
	}

	mu.RUnlock()

	enemies := make(
		map[string]EnemyInfo,
		len(enemyFile.Enemies),
	)

	for enemyID, coreData := range enemyFile.Enemies {
		name := enemyID

		if translatedName, exists := translations[enemyID]; exists {
			if translatedName != "" {
				name = translatedName
			}
		} else {
			log.Printf(
				"[DROPS] missing %s enemy translation: %s",
				language,
				enemyID,
			)
		}

		dropIDs := dropFile.Drops[enemyID]

		items := make(
			[]ItemInfo,
			0,
			len(dropIDs),
		)

		for _, itemID := range dropIDs {
			itemName := loadedItemNames[itemID]

			if itemName == "" {
				log.Printf(
					"[DROPS] missing item translation: %s",
					itemID,
				)

				itemName = itemID
			}

			items = append(items, ItemInfo{
				ID:   itemID,
				Name: itemName,
			})
		}

		sort.Slice(items, func(i, j int) bool {
			return items[i].Name < items[j].Name
		})

		enemies[enemyID] = EnemyInfo{
			ID:    enemyID,
			Name:  name,
			Zones: append([]string(nil), coreData.Zones...),
			Items: items,
		}
	}

	for enemyID := range dropFile.Drops {
		if _, exists := enemyFile.Enemies[enemyID]; !exists {
			log.Printf(
				"[DROPS] drop pool without core enemy: %s",
				enemyID,
			)
		}
	}

	for enemyID := range translations {
		if _, exists := enemyFile.Enemies[enemyID]; !exists {
			log.Printf(
				"[DROPS] translation without core enemy: %s",
				enemyID,
			)
		}
	}

	mu.Lock()
	worldCache[worldKey] = enemies
	mu.Unlock()

	log.Printf(
		"[DROPS] loaded %s (%d enemies)",
		worldKey,
		len(enemies),
	)
}

func cacheEmpty(worldKey string) {
	mu.Lock()
	worldCache[worldKey] = map[string]EnemyInfo{}
	mu.Unlock()
}

func Resolve(
	worldKey string,
	enemyID string,
) (EnemyInfo, bool) {
	loadWorld(worldKey)

	mu.RLock()
	defer mu.RUnlock()

	worldEnemies, worldExists := worldCache[worldKey]
	if !worldExists {
		return EnemyInfo{}, false
	}

	info, enemyExists := worldEnemies[enemyID]
	if !enemyExists {
		return EnemyInfo{}, false
	}

	info.Zones = append([]string(nil), info.Zones...)
	info.Items = append([]ItemInfo(nil), info.Items...)

	return info, true
}

func ResolveOrFallback(
	worldKey string,
	enemyID string,
) EnemyInfo {
	info, ok := Resolve(worldKey, enemyID)
	if ok {
		return info
	}

	return EnemyInfo{
		ID:    enemyID,
		Name:  enemyID,
		Items: []ItemInfo{},
	}
}
