package widget

import "corvin101/internal/game/state"

type HoverBehavior int

const (
	HoverNone HoverBehavior = iota
	HoverDim
)

type Widget interface {
	ID() string

	Update(snapshot state.Snapshot)

	Visible() bool

	Placement() Placement

	HoverBehavior() HoverBehavior
}
