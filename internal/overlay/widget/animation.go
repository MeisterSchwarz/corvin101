package widget

// Animated kann von Widgets implementiert werden, die zeitabhängig
// gerendert werden müssen.
//
// Solange Animating true liefert, fordert das Overlay regelmäßig
// neue Frames an.
type Animated interface {
	Animating() bool
}
