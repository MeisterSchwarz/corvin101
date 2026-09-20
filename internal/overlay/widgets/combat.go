package widgets

import (
	"corvin101/internal/game/events"
	"corvin101/internal/game/state"
)

type Combat struct {
	Visible bool
	Round   int
}

func (w *Combat) Update(snapshot state.Snapshot) {
	w.Visible =
		snapshot.Game.Mode == state.ModeBattle &&
			snapshot.Combat.Active &&
			snapshot.Combat.Phase == events.PhasePlanning

	w.Round = snapshot.Combat.Round
}
