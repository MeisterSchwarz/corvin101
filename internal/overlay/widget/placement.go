package widget

type Anchor int

const (
	AnchorTopLeft Anchor = iota
	AnchorTopCenter
	AnchorTopRight
	AnchorCenterLeft
	AnchorCenter
	AnchorCenterRight
	AnchorBottomLeft
	AnchorBottomCenter
	AnchorBottomRight
)

type Placement struct {
	Anchor Anchor

	// Relative Position innerhalb des Wizard-Clients.
	//
	// 0.0 = linke/obere Kante
	// 1.0 = rechte/untere Kante
	X float64
	Y float64

	Width  int32
	Height int32
}

func (p Placement) Bounds(
	viewport Viewport,
) Rect {
	width := p.Width
	height := p.Height

	if width < 1 {
		width = 1
	}

	if height < 1 {
		height = 1
	}

	x := int32(
		p.X * float64(viewport.Width),
	)

	y := int32(
		p.Y * float64(viewport.Height),
	)

	left := x
	top := y

	switch p.Anchor {
	case AnchorTopCenter:
		left -= width / 2

	case AnchorTopRight:
		left -= width

	case AnchorCenterLeft:
		top -= height / 2

	case AnchorCenter:
		left -= width / 2
		top -= height / 2

	case AnchorCenterRight:
		left -= width
		top -= height / 2

	case AnchorBottomLeft:
		top -= height

	case AnchorBottomCenter:
		left -= width / 2
		top -= height

	case AnchorBottomRight:
		left -= width
		top -= height
	}

	return Rect{
		Left:   left,
		Top:    top,
		Right:  left + width,
		Bottom: top + height,
	}
}
