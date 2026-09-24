package widgets

import (
	"time"

	"corvin101/internal/game/events"
	"corvin101/internal/game/state"
	"corvin101/internal/overlay/widget"
)

const roundPopDuration = 260 * time.Millisecond

type Round struct {
	visible bool
	round   int

	changedAt time.Time
}

func NewRound() *Round {
	return &Round{}
}

func (w *Round) ID() string {
	return "combat.round"
}

func (w *Round) Update(snapshot state.Snapshot) {
	newRound := snapshot.Combat.Round

	// Nur bei einem tatsächlichen Rundenwechsel animieren.
	if newRound > 0 && newRound != w.round {
		w.round = newRound
		w.changedAt = time.Now()
	}

	// Ein zurückgesetzter Combat-State setzt auch das Widget zurück.
	if newRound <= 0 {
		w.round = 0
		w.changedAt = time.Time{}
	}

	w.visible =
		snapshot.Combat.Active &&
			snapshot.Combat.Phase == events.PhasePlanning &&
			snapshot.Combat.Round > 0
}

func (w *Round) Visible() bool {
	return w.visible
}

func (w *Round) Round() int {
	return w.round
}

func (w *Round) HoverBehavior() widget.HoverBehavior {
	return widget.HoverNone
}

func (w *Round) Placement() widget.Placement {
	return widget.Placement{
		Anchor: widget.AnchorBottomRight,

		X: 0.985,
		Y: 0.985,

		Width:  130,
		Height: 120,
	}
}

func (w *Round) Animating() bool {
	if w.changedAt.IsZero() {
		return false
	}

	return time.Since(w.changedAt) < roundPopDuration
}

func (w *Round) Scale() float64 {
	if w.changedAt.IsZero() {
		return 1
	}

	elapsed := time.Since(w.changedAt)
	if elapsed >= roundPopDuration {
		return 1
	}

	t := float64(elapsed) / float64(roundPopDuration)
	t = easeOutBack(t)

	return 1.14 - 0.14*t
}

func (w *Round) Highlight() float64 {
	if w.changedAt.IsZero() {
		return 0
	}

	elapsed := time.Since(w.changedAt)
	if elapsed >= roundPopDuration {
		return 0
	}

	t := float64(elapsed) / float64(roundPopDuration)

	return 1 - easeOutCubic(t)
}

func easeOutCubic(t float64) float64 {
	x := 1 - t
	return 1 - x*x*x
}

func easeOutBack(t float64) float64 {
	const c1 = 1.70158
	const c3 = c1 + 1

	x := t - 1

	return 1 + c3*x*x*x + c1*x*x
}
