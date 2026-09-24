package widget

type Point struct {
	X int32
	Y int32
}

type Rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

func (r Rect) Width() int32 {
	return r.Right - r.Left
}

func (r Rect) Height() int32 {
	return r.Bottom - r.Top
}

func (r Rect) Contains(point Point) bool {
	return point.X >= r.Left &&
		point.X < r.Right &&
		point.Y >= r.Top &&
		point.Y < r.Bottom
}

type Viewport struct {
	Width  int32
	Height int32
}
