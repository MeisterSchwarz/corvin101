package web

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"ravendex/dropdata"
	"ravendex/enemy"
	"ravendex/filesystem"
	"ravendex/zones"
)

type Server struct {
	tracker  *enemy.EnemyTracker
	language string

	pendingMu sync.RWMutex

	// Schlüssel: "<language>/<world>"
	// Wert: enemyID -> übersetzter Name
	pendingNames map[string]map[string]string

	// Merkt sich, welche Sprach-/Welt-Dateien bereits geladen wurden.
	loadedNameScopes map[string]struct{}
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

// NewServer erwartet die aktuell eingestellte Sprache, beispielsweise "de" oder "en".
func NewServer(
	tracker *enemy.EnemyTracker,
	language string,
) *Server {
	language = strings.TrimSpace(language)

	return &Server{
		tracker:  tracker,
		language: language,

		pendingNames:     make(map[string]map[string]string),
		loadedNameScopes: make(map[string]struct{}),
	}
}

func (s *Server) Start(address string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/enemies", s.handleEnemies)
	mux.HandleFunc("/api/enemy-name", s.handleEnemyName)

	log.Printf("[WEB] listening on http://%s", address)
	log.Printf("[WEB] pending language: %s", s.language)

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

	duelWorldKey := zones.WorldKey(snapshot.DuelZoneKey)

	if duelWorldKey != "" {
		if err := s.ensurePendingNamesLoaded(
			s.language,
			duelWorldKey,
		); err != nil {
			log.Printf(
				"[WEB] load pending enemy names for language %q, world %q: %v",
				s.language,
				duelWorldKey,
				err,
			)
		}
	}

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

		// Lokal eingetragene Übersetzungen nur für die aktuelle
		// Sprache und Welt verwenden.
		if duelWorldKey != "" {
			if pendingName, ok := s.pendingEnemyName(
				s.language,
				duelWorldKey,
				enemyID,
			); ok {
				name = pendingName
				nameMissing = false
			}
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

	world := zones.WorldKey(snapshot.DuelZoneKey)

	if world == "" {
		http.Error(
			w,
			"could not determine current world",
			http.StatusConflict,
		)
		return
	}

	if err := s.ensurePendingNamesLoaded(
		s.language,
		world,
	); err != nil {
		log.Printf(
			"[WEB] load pending enemy names for language %q, world %q: %v",
			s.language,
			world,
			err,
		)

		http.Error(
			w,
			"could not load enemy names",
			http.StatusInternalServerError,
		)
		return
	}

	pendingNamesPath := filesystem.PendingEnemyNamesPath(
		s.language,
		world,
	)

	// Nur die zur Sprache und Welt gehörende Namensdatei wird verändert.
	if err := SetPendingEnemyName(
		pendingNamesPath,
		request.EnemyID,
		request.EnemyName,
	); err != nil {
		log.Printf(
			"[WEB] save enemy name %q for language %q, world %q: %v",
			request.EnemyID,
			s.language,
			world,
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

	scope := pendingScope(
		s.language,
		world,
	)

	if s.pendingNames[scope] == nil {
		s.pendingNames[scope] =
			make(map[string]string)
	}

	s.pendingNames[scope][request.EnemyID] =
		request.EnemyName

	s.pendingMu.Unlock()

	log.Printf(
		"[WEB] saved enemy name %q -> %q for language %q, world %q",
		request.EnemyID,
		request.EnemyName,
		s.language,
		world,
	)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) pendingEnemyName(
	language string,
	world string,
	enemyID string,
) (string, bool) {
	scope := pendingScope(
		language,
		world,
	)

	s.pendingMu.RLock()
	defer s.pendingMu.RUnlock()

	namesForScope := s.pendingNames[scope]
	if namesForScope == nil {
		return "", false
	}

	name, ok := namesForScope[enemyID]
	return name, ok
}

// ensurePendingNamesLoaded lädt eine Sprach-/Welt-Datei genau einmal.
//
// Das Laden geschieht unter dem Write-Lock. Dadurch können nicht zwei
// Requests dieselbe Datei gleichzeitig initialisieren.
func (s *Server) ensurePendingNamesLoaded(
	language string,
	world string,
) error {
	language = strings.TrimSpace(language)
	world = strings.TrimSpace(world)

	if language == "" || world == "" {
		return nil
	}

	scope := pendingScope(
		language,
		world,
	)

	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()

	if _, loaded := s.loadedNameScopes[scope]; loaded {
		return nil
	}

	names := make(map[string]string)

	path := filesystem.PendingEnemyNamesPath(
		language,
		world,
	)

	if err := readJSONIfExists(path, &names); err != nil {
		return err
	}

	cleanNames := make(map[string]string)

	for enemyID, name := range names {
		enemyID = strings.TrimSpace(enemyID)
		name = strings.TrimSpace(name)

		if enemyID == "" || name == "" {
			continue
		}

		cleanNames[enemyID] = name
	}

	s.pendingNames[scope] = cleanNames
	s.loadedNameScopes[scope] = struct{}{}

	log.Printf(
		"[WEB] loaded %d pending enemy names for language %q, world %q from %s",
		len(cleanNames),
		language,
		world,
		path,
	)

	return nil
}

func pendingScope(
	language string,
	world string,
) string {
	return strings.TrimSpace(language) +
		"/" +
		strings.TrimSpace(world)
}
