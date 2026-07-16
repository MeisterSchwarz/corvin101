package dropdata

import (
	"encoding/json"
	"fmt"
	"log"
	"ravendex/rest"
	"sort"
	"strings"
	"sync"
)

const language = "de"

var (
	// Geladene Gegnerdaten, gruppiert nach Welt.
	worldCache = make(map[string]map[string]EnemyInfo)

	// Verhindert paralleles mehrfaches Laden derselben Welt.
	worldLoading = make(map[string]chan struct{})

	// Übersetzte Itemnamen: Item-ID -> übersetzter Name.
	itemNames = make(map[string]string)

	// Kategorie -> Index -> Item-ID.
	itemIDsByCategory = make(map[string]map[int]string)

	loadedItemFiles  = make(map[string]bool)
	loadingItemFiles = make(map[string]chan struct{})

	mu sync.RWMutex
)

var itemFiles = map[string]string{
	"RG": "reagents",
	"RN": "rings",
	"HT": "hats",
	"RB": "robes",
	"BT": "boots",
	"AT": "athames",
	"WD": "wands",
	"DK": "decks",
	"SN": "snacks",
	"JW": "jewels",
	"TC": "treasurecards",
}

// core/items/<category>.json
type coreItemMetadata struct {
	ID int `json:"id"`
}

type coreItemFile map[string]coreItemMetadata

// core/drops/<world>.json
//
// enemyID -> category -> item indexes
type coreDropFile map[string]map[string][]int

// core/enemies/<world>.json
type coreEnemyMetadata struct{}

type coreEnemyFile map[string]coreEnemyMetadata

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
	Items []ItemInfo `json:"items"`
}

func resolveEnemyDrops(
	worldKey string,
	enemyID string,
	dropsByCategory map[string][]int,
	itemTables map[string]map[int]string,
) []string {
	totalDrops := 0

	for _, indexes := range dropsByCategory {
		totalDrops += len(indexes)
	}

	dropIDs := make([]string, 0, totalDrops)

	for rawCategory, indexes := range dropsByCategory {
		category := strings.ToUpper(
			strings.TrimSpace(rawCategory),
		)

		itemsByIndex, exists := itemTables[category]
		if !exists {
			log.Printf(
				"[DROPS] missing item table for %s/%s category %s",
				worldKey,
				enemyID,
				category,
			)
			continue
		}

		for _, index := range indexes {
			itemID, exists := itemsByIndex[index]
			if !exists {
				log.Printf(
					"[DROPS] invalid item index for %s/%s: %s/%d",
					worldKey,
					enemyID,
					category,
					index,
				)
				continue
			}

			dropIDs = append(dropIDs, itemID)
		}
	}

	return dropIDs
}

func itemFileForID(itemID string) (string, bool) {
	prefix, _, found := strings.Cut(itemID, "_")
	if !found || prefix == "" {
		return "", false
	}

	fileName, ok := itemFiles[prefix]
	return fileName, ok
}

func loadItemFile(
	category string,
	fileName string,
) error {
	mu.Lock()

	if loadedItemFiles[category] {
		mu.Unlock()
		return nil
	}

	if wait, loading := loadingItemFiles[category]; loading {
		mu.Unlock()
		<-wait

		mu.RLock()
		loaded := loadedItemFiles[category]
		mu.RUnlock()

		if !loaded {
			return fmt.Errorf(
				"item category %s could not be loaded",
				category,
			)
		}

		return nil
	}

	wait := make(chan struct{})
	loadingItemFiles[category] = wait
	mu.Unlock()

	defer func() {
		mu.Lock()
		close(wait)
		delete(loadingItemFiles, category)
		mu.Unlock()
	}()

	corePath := fmt.Sprintf(
		"core/items/%s.json",
		fileName,
	)

	translationPath := fmt.Sprintf(
		"i18n/%s/items/%s.json",
		language,
		fileName,
	)

	coreRaw, err := rest.ReadFile(corePath)
	if err != nil {
		return fmt.Errorf("load %s: %w", corePath, err)
	}

	translationRaw, translationErr := rest.ReadFile(translationPath)
	if translationErr != nil {
		log.Printf(
			"[DROPS] optional item translation file %s unavailable: %v",
			translationPath,
			translationErr,
		)
	}

	var coreItems coreItemFile
	if err := json.Unmarshal(coreRaw, &coreItems); err != nil {
		return fmt.Errorf("parse %s: %w", corePath, err)
	}

	translations := make(map[string]string)

	if translationErr == nil {
		if err := json.Unmarshal(
			translationRaw,
			&translations,
		); err != nil {
			log.Printf(
				"[DROPS] invalid optional item translation file %s: %v",
				translationPath,
				err,
			)

			translations = make(map[string]string)
		}
	}

	indexToItemID := make(map[int]string, len(coreItems))

	for itemID, metadata := range coreItems {
		if metadata.ID < 0 {
			return fmt.Errorf(
				"negative item index %d for %q in %s",
				metadata.ID,
				itemID,
				corePath,
			)
		}

		if existingID, exists := indexToItemID[metadata.ID]; exists {
			return fmt.Errorf(
				"duplicate item index %d for %q and %q in %s",
				metadata.ID,
				existingID,
				itemID,
				corePath,
			)
		}

		indexToItemID[metadata.ID] = itemID
	}

	mu.Lock()
	defer mu.Unlock()

	if loadedItemFiles[category] {
		return nil
	}

	itemIDsByCategory[category] = indexToItemID

	for itemID := range coreItems {
		name := strings.TrimSpace(translations[itemID])

		if name == "" {
			log.Printf(
				"[DROPS] missing %s item translation: %s",
				language,
				itemID,
			)

			name = itemID
		}

		if existing, exists := itemNames[itemID]; exists &&
			existing != name {
			return fmt.Errorf(
				"duplicate item ID %q with conflicting translations",
				itemID,
			)
		}

		itemNames[itemID] = name
	}

	for itemID := range translations {
		if _, exists := coreItems[itemID]; !exists {
			log.Printf(
				"[DROPS] item translation without core item: %s",
				itemID,
			)
		}
	}

	loadedItemFiles[category] = true

	log.Printf(
		"[DROPS] loaded item category %s (%d items)",
		category,
		len(coreItems),
	)

	return nil
}

func loadRequiredItemFiles(dropFile coreDropFile) error {
	requiredCategories := make(map[string]struct{})

	for _, categories := range dropFile {
		for category := range categories {
			category = strings.ToUpper(
				strings.TrimSpace(category),
			)

			if category == "" {
				continue
			}

			requiredCategories[category] = struct{}{}
		}
	}

	for category := range requiredCategories {
		fileName, ok := itemFiles[category]
		if !ok {
			log.Printf(
				"[DROPS] unknown item category: %s",
				category,
			)
			continue
		}

		if err := loadItemFile(category, fileName); err != nil {
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

	loadedItemTables := make(
		map[string]map[int]string,
		len(itemIDsByCategory),
	)

	for category, sourceTable := range itemIDsByCategory {
		tableCopy := make(
			map[int]string,
			len(sourceTable),
		)

		for index, itemID := range sourceTable {
			tableCopy[index] = itemID
		}

		loadedItemTables[category] = tableCopy
	}

	mu.RUnlock()

	enemies := make(
		map[string]EnemyInfo,
		len(enemyFile),
	)

	for enemyID := range enemyFile {
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

		dropData, hasDropData := dropFile[enemyID]

		dropIDs := []string{}

		if hasDropData {
			dropIDs = resolveEnemyDrops(
				worldKey,
				enemyID,
				dropData,
				loadedItemTables,
			)
		}

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
			Items: items,
		}
	}

	for enemyID := range dropFile {
		if _, exists := enemyFile[enemyID]; !exists {
			log.Printf(
				"[DROPS] drop pool without core enemy: %s",
				enemyID,
			)
		}
	}

	for enemyID := range translations {
		if _, exists := enemyFile[enemyID]; !exists {
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
