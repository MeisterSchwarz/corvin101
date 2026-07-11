package web

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"wizlink/dropdata"
	"wizlink/enemy"
	"wizlink/filesystem"
	"wizlink/zones"
)

type Server struct {
	tracker *enemy.EnemyTracker

	pendingNamesPath string
	pendingZonesPath string

	pendingMu    sync.RWMutex
	pendingNames map[string]string
	seenZones    map[string]map[string]struct{}
}

type CombatEnemy struct {
	ID          string              `json:"enemyId"`
	Name        string              `json:"enemyName"`
	NameMissing bool                `json:"nameMissing"`
	Count       int                 `json:"count"`
	Items       []dropdata.ItemInfo `json:"items"`
}

type CombatResponse struct {
	DuelID string `json:"duelId"`

	ZoneKey      string `json:"zoneKey"`
	Zone         string `json:"zone"`
	Sub          string `json:"sub"`
	World        string `json:"world"`
	ZoneResolved bool   `json:"zoneResolved"`

	Won     bool          `json:"won"`
	Ended   bool          `json:"ended"`
	Active  bool          `json:"active"`
	Enemies []CombatEnemy `json:"enemies"`
}

type EnemyNameRequest struct {
	EnemyID   string `json:"enemyId"`
	EnemyName string `json:"enemyName"`
}

func NewServer(tracker *enemy.EnemyTracker) *Server {
	server := &Server{
		tracker: tracker,

		pendingNamesPath: filesystem.PendingEnemyNamesPath(),
		pendingZonesPath: filesystem.PendingEnemyZonesPath(),

		pendingNames: make(map[string]string),
		seenZones:    make(map[string]map[string]struct{}),
	}

	server.loadPendingNames()
	server.loadPendingZones()

	return server
}

func (s *Server) Start(address string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/enemies", s.handleEnemies)
	mux.HandleFunc("/api/enemy-name", s.handleEnemyName)

	log.Printf("[WEB] listening on http://%s", address)
	log.Printf(
		"[WEB] pending enemy names: %s",
		s.pendingNamesPath,
	)
	log.Printf(
		"[WEB] pending enemy zones: %s",
		s.pendingZonesPath,
	)

	return http.ListenAndServe(address, mux)
}

func (s *Server) handleIndex(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)

	if _, err := w.Write([]byte(indexHTML)); err != nil {
		log.Printf("[WEB] write index response: %v", err)
	}
}

func (s *Server) handleEnemies(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	snapshot := s.tracker.Snapshot()

	// Neue Gegner-Zonen-Zuordnungen werden automatisch gespeichert.
	s.recordEnemyZones(
		snapshot.Enemies,
		snapshot.DuelZoneKey,
	)

	currentZoneInfo := zones.ZoneInfo{}
	currentZoneResolved := false

	if snapshot.CurrentZoneKey != "" {
		currentZoneInfo, currentZoneResolved =
			zones.Resolve(snapshot.CurrentZoneKey)

		if !currentZoneResolved {
			currentZoneInfo = zones.ZoneInfo{
				Name: snapshot.CurrentZoneKey,
				World: zones.WorldKey(
					snapshot.CurrentZoneKey,
				),
			}
		}
	}

	duelWorldKey := zones.WorldKey(
		snapshot.DuelZoneKey,
	)

	enemies := make(
		[]CombatEnemy,
		0,
		len(snapshot.Enemies),
	)

	for enemyID, count := range snapshot.Enemies {
		info := dropdata.ResolveOrFallback(
			duelWorldKey,
			enemyID,
		)

		name := strings.TrimSpace(info.Name)
		nameMissing := name == "" ||
			name == enemyID ||
			name == info.ID

		// Lokal eingetragene Übersetzungen direkt verwenden.
		if pendingName, ok := s.pendingEnemyName(enemyID); ok {
			name = pendingName
			nameMissing = false
		}

		if name == "" {
			name = enemyID
		}

		enemies = append(enemies, CombatEnemy{
			ID:          enemyID,
			Name:        name,
			NameMissing: nameMissing,
			Count:       count,
			Items:       info.Items,
		})
	}

	sort.Slice(enemies, func(i, j int) bool {
		return enemies[i].Name < enemies[j].Name
	})

	response := CombatResponse{
		DuelID: snapshot.DuelID,

		ZoneKey:      snapshot.CurrentZoneKey,
		Zone:         currentZoneInfo.Name,
		Sub:          currentZoneInfo.Sub,
		World:        currentZoneInfo.World,
		ZoneResolved: currentZoneResolved,

		Won:     snapshot.Won,
		Ended:   snapshot.Ended,
		Active:  snapshot.Active,
		Enemies: enemies,
	}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf(
			"[WEB] encode enemies response: %v",
			err,
		)
	}
}

func (s *Server) handleEnemyName(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	var request EnemyNameRequest

	decoder := json.NewDecoder(
		http.MaxBytesReader(w, r.Body, 8<<10),
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		http.Error(
			w,
			"invalid JSON request",
			http.StatusBadRequest,
		)
		return
	}

	request.EnemyID = strings.TrimSpace(
		request.EnemyID,
	)
	request.EnemyName = strings.TrimSpace(
		request.EnemyName,
	)

	if request.EnemyID == "" {
		http.Error(
			w,
			"enemyId is required",
			http.StatusBadRequest,
		)
		return
	}

	if request.EnemyName == "" {
		http.Error(
			w,
			"enemyName is required",
			http.StatusBadRequest,
		)
		return
	}

	if utf8.RuneCountInString(
		request.EnemyName,
	) > 120 {
		http.Error(
			w,
			"enemyName is too long",
			http.StatusBadRequest,
		)
		return
	}

	snapshot := s.tracker.Snapshot()

	if _, exists :=
		snapshot.Enemies[request.EnemyID]; !exists {
		http.Error(
			w,
			"enemy is not part of the current combat",
			http.StatusConflict,
		)
		return
	}

	// Nur die Namensdatei wird verändert.
	if err := SetPendingEnemyName(
		s.pendingNamesPath,
		request.EnemyID,
		request.EnemyName,
	); err != nil {
		log.Printf(
			"[WEB] save enemy name %q: %v",
			request.EnemyID,
			err,
		)

		http.Error(
			w,
			"could not save enemy name",
			http.StatusInternalServerError,
		)
		return
	}

	s.pendingMu.Lock()
	s.pendingNames[request.EnemyID] =
		request.EnemyName
	s.pendingMu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) recordEnemyZones(
	enemies map[string]int,
	zoneKey string,
) {
	zoneKey = strings.TrimSpace(zoneKey)

	if zoneKey == "" || len(enemies) == 0 {
		return
	}

	for enemyID := range enemies {
		if s.zoneWasSeen(enemyID, zoneKey) {
			continue
		}

		// Nur die Zonendatei wird verändert.
		if err := AddPendingEnemyZone(
			s.pendingZonesPath,
			enemyID,
			zoneKey,
		); err != nil {
			log.Printf(
				"[WEB] save enemy zone %q -> %q: %v",
				enemyID,
				zoneKey,
				err,
			)
			continue
		}

		s.pendingMu.Lock()
		s.markZoneSeenLocked(enemyID, zoneKey)
		s.pendingMu.Unlock()

		log.Printf(
			"[WEB] recorded enemy zone %q -> %q",
			enemyID,
			zoneKey,
		)
	}
}

func (s *Server) pendingEnemyName(
	enemyID string,
) (string, bool) {
	s.pendingMu.RLock()
	defer s.pendingMu.RUnlock()

	name, ok := s.pendingNames[enemyID]
	return name, ok
}

func (s *Server) zoneWasSeen(
	enemyID string,
	zoneKey string,
) bool {
	s.pendingMu.RLock()
	defer s.pendingMu.RUnlock()

	zonesForEnemy := s.seenZones[enemyID]
	if zonesForEnemy == nil {
		return false
	}

	_, exists := zonesForEnemy[zoneKey]
	return exists
}

func (s *Server) markZoneSeenLocked(
	enemyID string,
	zoneKey string,
) {
	if s.seenZones[enemyID] == nil {
		s.seenZones[enemyID] =
			make(map[string]struct{})
	}

	s.seenZones[enemyID][zoneKey] = struct{}{}
}

func (s *Server) loadPendingNames() {
	names := make(map[string]string)

	if err := readJSONIfExists(
		s.pendingNamesPath,
		&names,
	); err != nil {
		log.Printf(
			"[WEB] load pending enemy names: %v",
			err,
		)
		return
	}

	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()

	for enemyID, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		s.pendingNames[enemyID] = name
	}
}

func (s *Server) loadPendingZones() {
	zonesFile := EnemyZonesFile{
		Enemies: make(map[string]EnemyZones),
	}

	if err := readJSONIfExists(
		s.pendingZonesPath,
		&zonesFile,
	); err != nil {
		log.Printf(
			"[WEB] load pending enemy zones: %v",
			err,
		)
		return
	}

	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()

	for enemyID, entry := range zonesFile.Enemies {
		for _, zoneKey := range entry.Zones {
			zoneKey = strings.TrimSpace(zoneKey)
			if zoneKey == "" {
				continue
			}

			s.markZoneSeenLocked(
				enemyID,
				zoneKey,
			)
		}
	}
}
