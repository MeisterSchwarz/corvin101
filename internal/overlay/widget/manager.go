package widget

import "corvin101/internal/game/state"

type Entry struct {
	Widget Widget

	Bounds Rect

	Hovered bool
}

type Manager struct {
	widgets []Widget
}

func NewManager(
	widgets ...Widget,
) *Manager {
	result := &Manager{
		widgets: make(
			[]Widget,
			0,
			len(widgets),
		),
	}

	for _, item := range widgets {
		if item == nil {
			continue
		}

		result.widgets = append(
			result.widgets,
			item,
		)
	}

	return result
}

func (m *Manager) Add(
	item Widget,
) {
	if item == nil {
		return
	}

	m.widgets = append(
		m.widgets,
		item,
	)
}

func (m *Manager) Update(
	snapshot state.Snapshot,
) {
	for _, item := range m.widgets {
		item.Update(snapshot)
	}
}

func (m *Manager) Animating() bool {
	for _, current := range m.widgets {
		animated, ok := current.(Animated)
		if !ok {
			continue
		}

		if animated.Animating() {
			return true
		}
	}

	return false
}

func (m *Manager) Entries(
	viewport Viewport,
	cursor Point,
	cursorKnown bool,
) []Entry {
	result := make(
		[]Entry,
		0,
		len(m.widgets),
	)

	for _, item := range m.widgets {
		if item == nil ||
			!item.Visible() {

			continue
		}

		bounds :=
			item.Placement().
				Bounds(viewport)

		hovered := false

		if cursorKnown &&
			item.HoverBehavior() ==
				HoverDim {

			hovered =
				bounds.Contains(cursor)
		}

		result = append(
			result,
			Entry{
				Widget:  item,
				Bounds:  bounds,
				Hovered: hovered,
			},
		)
	}

	return result
}
