package widgets

import (
	"context"
	"log"
	"sort"
	"sync"

	"corvin101/internal/data/enemies"
	"corvin101/internal/game/events"
	"corvin101/internal/game/state"
	"corvin101/internal/overlay/widget"
)

const maxEnemyAffinitySlots = 4

type EnemyAffinityGroup struct {
	Value   int
	Schools []enemies.School
}

type EnemyAffinityEntry struct {
	Slot      int
	SubCircle int
	MobID     string
	Name      string

	Groups []EnemyAffinityGroup
}

type EnemyAffinities struct {
	mu sync.RWMutex

	repository *enemies.Repository
	invalidate func()

	visible bool
	entries []EnemyAffinityEntry

	loaded  map[string]enemies.EnemyInfo
	missing map[string]bool
	loading map[string]bool
}

func NewEnemyAffinities(
	repository *enemies.Repository,
	invalidate func(),
) *EnemyAffinities {
	return &EnemyAffinities{
		repository: repository,
		invalidate: invalidate,

		loaded:  make(map[string]enemies.EnemyInfo),
		missing: make(map[string]bool),
		loading: make(map[string]bool),
	}
}

func (w *EnemyAffinities) ID() string {
	return "combat.enemy_affinities"
}

func (w *EnemyAffinities) Update(
	snapshot state.Snapshot,
) {
	if !snapshot.Combat.Active {
		w.mu.Lock()
		w.visible = false
		w.entries = nil
		w.mu.Unlock()

		return
	}

	participants := make(
		[]state.ParticipantSnapshot,
		0,
		len(snapshot.Combat.Participants),
	)

	for _, participant := range snapshot.Combat.Participants {
		log.Printf(
			"[affinity] participant id=%s sub=%d kind=%s mob=%q",
			participant.ID,
			participant.SubCircle,
			participant.Kind,
			participant.MobID,
		)
		if participant.Kind != events.ParticipantMob {
			continue
		}

		if participant.MobID == "" {
			continue
		}

		participants = append(
			participants,
			participant,
		)

		w.ensureLoaded(
			snapshot.Game.ZoneKey,
			participant.MobID,
		)
	}

	sort.Slice(
		participants,
		func(i int, j int) bool {
			return participants[i].SubCircle <
				participants[j].SubCircle
		},
	)

	if len(participants) > maxEnemyAffinitySlots {
		participants =
			participants[:maxEnemyAffinitySlots]
	}

	entries := make(
		[]EnemyAffinityEntry,
		0,
		len(participants),
	)

	w.mu.RLock()

	for _, participant := range participants {
		if participant.SubCircle < 0 ||
			participant.SubCircle >= maxEnemyAffinitySlots {

			continue
		}

		info, ok :=
			w.loaded[participant.MobID]

		if !ok {
			continue
		}

		groups := groupAffinities(
			info.Affinities,
		)

		if len(groups) == 0 {
			continue
		}

		entries = append(
			entries,
			EnemyAffinityEntry{
				Slot:      participant.SubCircle,
				SubCircle: participant.SubCircle,
				MobID:     participant.MobID,
				Name:      info.Name,
				Groups:    groups,
			},
		)
	}

	w.mu.RUnlock()

	w.mu.Lock()

	w.entries = entries

	w.visible =
		snapshot.Combat.Phase == events.PhasePlanning &&
			len(entries) > 0

	w.mu.Unlock()
}

func (w *EnemyAffinities) Visible() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.visible
}

func (w *EnemyAffinities) HoverBehavior() widget.HoverBehavior {
	return widget.HoverNone
}

func (w *EnemyAffinities) Placement() widget.Placement {
	return widget.Placement{
		Anchor: widget.AnchorTopLeft,
		X:      0,
		Y:      0,
		Width:  1,
		Height: 1,
	}
}

func (w *EnemyAffinities) Entries() []EnemyAffinityEntry {
	w.mu.RLock()
	defer w.mu.RUnlock()

	result := make(
		[]EnemyAffinityEntry,
		len(w.entries),
	)

	for i, entry := range w.entries {
		result[i] = EnemyAffinityEntry{
			Slot:      entry.Slot,
			SubCircle: entry.SubCircle,
			MobID:     entry.MobID,
			Name:      entry.Name,
			Groups:    cloneGroups(entry.Groups),
		}
	}

	return result
}

func (w *EnemyAffinities) ensureLoaded(
	zoneKey string,
	mobID string,
) {
	if zoneKey == "" ||
		mobID == "" ||
		w.repository == nil {

		return
	}

	w.mu.Lock()

	if _, ok := w.loaded[mobID]; ok {
		w.mu.Unlock()
		return
	}

	if w.missing[mobID] {
		w.mu.Unlock()
		return
	}

	if w.loading[mobID] {
		w.mu.Unlock()
		return
	}

	w.loading[mobID] = true

	w.mu.Unlock()

	go w.load(
		zoneKey,
		mobID,
	)
}

func (w *EnemyAffinities) load(
	zoneKey string,
	mobID string,
) {
	info, found, err :=
		w.repository.Resolve(
			context.Background(),
			zoneKey,
			mobID,
		)

	log.Printf(
		"[affinity] resolved zone=%q mob=%q found=%v name=%q affinities=%v err=%v",
		zoneKey,
		mobID,
		found,
		info.Name,
		info.Affinities,
		err,
	)

	w.mu.Lock()

	delete(
		w.loading,
		mobID,
	)

	if err != nil {
		w.mu.Unlock()

		log.Printf(
			"[overlay] resolve enemy %q in zone %q: %v",
			mobID,
			zoneKey,
			err,
		)

		return
	}

	if !found {
		w.missing[mobID] = true
		w.mu.Unlock()
		return
	}

	w.loaded[mobID] = info

	invalidate := w.invalidate

	w.mu.Unlock()

	if invalidate != nil {
		invalidate()
	}
}

func groupAffinities(
	affinities map[enemies.School]int,
) []EnemyAffinityGroup {
	if len(affinities) == 0 {
		return nil
	}

	byValue := make(
		map[int][]enemies.School,
	)

	for school, value := range affinities {
		if value == 0 {
			continue
		}

		byValue[value] = append(
			byValue[value],
			school,
		)
	}

	if len(byValue) == 0 {
		return nil
	}

	values := make(
		[]int,
		0,
		len(byValue),
	)

	for value := range byValue {
		values = append(
			values,
			value,
		)
	}

	sort.Slice(values, func(i, j int) bool {
		left := values[i]
		right := values[j]

		// Resistenzen links, Boosts rechts.
		if left >= 0 && right < 0 {
			return true
		}

		if left < 0 && right >= 0 {
			return false
		}

		// Innerhalb beider Bereiche nach Stärke:
		// +15, +30, +60
		// -15, -30, -60
		if left >= 0 {
			return left < right
		}

		return left > right
	})

	schoolOrder := map[enemies.School]int{
		enemies.SchoolFire:    0,
		enemies.SchoolIce:     1,
		enemies.SchoolStorm:   2,
		enemies.SchoolMyth:    3,
		enemies.SchoolLife:    4,
		enemies.SchoolDeath:   5,
		enemies.SchoolBalance: 6,
		enemies.SchoolSun:     7,
		enemies.SchoolMoon:    8,
		enemies.SchoolStar:    9,
		enemies.SchoolShadow:  10,
		enemies.SchoolAny:     11,
	}

	result := make(
		[]EnemyAffinityGroup,
		0,
		len(values),
	)

	for _, value := range values {
		schools := append(
			[]enemies.School(nil),
			byValue[value]...,
		)

		sort.Slice(
			schools,
			func(i int, j int) bool {
				return schoolOrder[schools[i]] <
					schoolOrder[schools[j]]
			},
		)

		result = append(
			result,
			EnemyAffinityGroup{
				Value:   value,
				Schools: schools,
			},
		)
	}

	return result
}

func cloneGroups(
	source []EnemyAffinityGroup,
) []EnemyAffinityGroup {
	if len(source) == 0 {
		return nil
	}

	result := make(
		[]EnemyAffinityGroup,
		len(source),
	)

	for i, group := range source {
		result[i] = EnemyAffinityGroup{
			Value: group.Value,

			Schools: append(
				[]enemies.School(nil),
				group.Schools...,
			),
		}
	}

	return result
}
