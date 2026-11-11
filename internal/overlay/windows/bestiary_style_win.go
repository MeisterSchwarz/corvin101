//go:build windows

package windows

import (
	"corvin101/internal/overlay/widget"
	"math"
	"time"
	"unsafe"
)

func panelInset(r widget.Rect, n int32) widget.Rect {
	return widget.Rect{Left: r.Left + n, Top: r.Top + n, Right: r.Right - n, Bottom: r.Bottom - n}
}

func panelRule(pixels unsafe.Pointer, width, height int32, r widget.Rect, color uint32) {
	fillPanelRect(pixels, width, height, widget.Rect{Left: r.Left, Top: r.Top, Right: r.Right, Bottom: r.Top + 1}, color)
	fillPanelRect(pixels, width, height, widget.Rect{Left: r.Left, Top: r.Bottom - 1, Right: r.Right, Bottom: r.Bottom}, color)
	fillPanelRect(pixels, width, height, widget.Rect{Left: r.Left, Top: r.Top, Right: r.Left + 1, Bottom: r.Bottom}, color)
	fillPanelRect(pixels, width, height, widget.Rect{Left: r.Right - 1, Top: r.Top, Right: r.Right, Bottom: r.Bottom}, color)
}

func panelGradient(pixels unsafe.Pointer, width, height int32, r widget.Rect, base [3]int32, depth int32, grain bool) {
	buffer := unsafe.Slice((*uint32)(pixels), int(width*height))
	for y := max(0, r.Top); y < min(height, r.Bottom); y++ {
		for x := max(0, r.Left); x < min(width, r.Right); x++ {
			dx := int64(2*(x-r.Left)-r.Width()) * 128 / int64(max(1, r.Width()))
			dy := int64(2*(y-r.Top)-r.Height()) * 128 / int64(max(1, r.Height()))
			shade := int32((dx*dx + dy*dy) * int64(depth) / 32768)
			noise := int32(0)
			if grain {
				noise = int32((uint32(x)*374761393^uint32(y)*668265263)%3) - 1
			}
			red := uint32(max(0, min(255, base[0]-shade+noise)))
			green := uint32(max(0, min(255, base[1]-shade+noise)))
			blue := uint32(max(0, min(255, base[2]-shade+noise)))
			buffer[int(y)*int(width)+int(x)] = 0xff000000 | red<<16 | green<<8 | blue
		}
	}
}

func drawBestiaryFrame(pixels unsafe.Pointer, width, height int32, bounds widget.Rect) {
	buffer := unsafe.Slice((*uint32)(pixels), int(width*height))
	radius := max(int32(8), height/45)
	for y := max(0, bounds.Top-radius); y < min(height, bounds.Bottom+radius); y++ {
		for x := max(0, bounds.Left-radius); x < min(width, bounds.Right+radius); x++ {
			distance := max(bounds.Left-x, x-bounds.Right+1, bounds.Top-y, y-bounds.Bottom+1)
			if distance <= 0 || distance >= radius {
				continue
			}
			alpha := uint32((radius - distance) * 110 / radius)
			index := int(y)*int(width) + int(x)
			old := buffer[index]
			inv := 255 - alpha
			buffer[index] = (alpha+(old>>24)*inv/255)<<24 | (((old>>16)&255)*inv/255)<<16 | (((old>>8)&255)*inv/255)<<8 | (old&255)*inv/255
		}
	}
	panelGradient(pixels, width, height, bounds, [3]int32{48, 39, 56}, 15, true)
	for i, color := range []uint32{0xff120e18, 0xff8b6736, 0xffe2bf73, 0xff685031, 0xff17131f, 0xff4f3e4b} {
		panelRule(pixels, width, height, panelInset(bounds, int32(i)), color)
	}
	length := max(int32(14), height/36)
	for _, corner := range []struct{ x, y, dx, dy int32 }{{bounds.Left + 7, bounds.Top + 7, 1, 1}, {bounds.Right - 8, bounds.Top + 7, -1, 1}, {bounds.Left + 7, bounds.Bottom - 8, 1, -1}, {bounds.Right - 8, bounds.Bottom - 8, -1, -1}} {
		for i := int32(0); i < length; i++ {
			x, y := corner.x+corner.dx*i, corner.y+corner.dy*i
			fillPanelRect(pixels, width, height, widget.Rect{Left: x, Top: corner.y, Right: x + 1, Bottom: corner.y + 1}, 0xffbd9552)
			fillPanelRect(pixels, width, height, widget.Rect{Left: corner.x, Top: y, Right: corner.x + 1, Bottom: y + 1}, 0xffbd9552)
		}
	}
}

func drawBestiaryRow(pixels unsafe.Pointer, width, height int32, row widget.Rect, selected, hovered bool) {
	row.Bottom -= 2
	base := [3]int32{43, 35, 50}
	if hovered {
		base = [3]int32{68, 53, 70}
	}
	if selected {
		base = [3]int32{92, 68, 44}
	}
	panelGradient(pixels, width, height, row, base, 8, false)
	if selected {
		panelRule(pixels, width, height, row, 0xff987744)
		marker := row
		marker.Right = marker.Left + 3
		fillPanelRect(pixels, width, height, marker, 0xffe1bb6c)
	}
}

// A slow ember pulse distinguishes combat opponents from the gold selection.
func drawCombatBestiaryRow(pixels unsafe.Pointer, width, height int32, row widget.Rect, selected, hovered bool, now time.Time) {
	row.Bottom -= 2
	pulse := (1 + math.Sin(float64(now.UnixMilli()%2400)*2*math.Pi/2400)) / 2
	base := [3]int32{77, 39, 34}
	if hovered {
		base = [3]int32{96, 51, 40}
	}
	if selected {
		base = [3]int32{106, 68, 38}
	}
	panelGradient(pixels, width, height, row, base, 9, false)
	red, green, blue := uint32(142+45*pulse), uint32(75+35*pulse), uint32(43+13*pulse)
	panelRule(pixels, width, height, row, 0xff000000|red<<16|green<<8|blue)
	marker := row
	marker.Right = marker.Left + 3
	fillPanelRect(pixels, width, height, marker, 0xff000000|uint32(195+35*pulse)<<16|uint32(104+40*pulse)<<8|52)
	if selected {
		panelRule(pixels, width, height, panelInset(row, 2), 0xffcda75c)
		fillPanelRect(pixels, width, height, marker, 0xffefd084)
	}
}

func drawLocationBestiaryRow(pixels unsafe.Pointer, width, height int32, row widget.Rect, selected, hovered bool, now time.Time) {
	row.Bottom -= 2
	pulse := (1 + math.Sin(float64(now.UnixMilli()%2800)*2*math.Pi/2800)) / 2
	base := [3]int32{30, 48, 78}
	if hovered {
		base = [3]int32{39, 64, 100}
	}
	if selected {
		base = [3]int32{48, 73, 110}
	}
	panelGradient(pixels, width, height, row, base, 9, false)
	color := uint32(0xff000000) | uint32(65+20*pulse)<<16 | uint32(110+30*pulse)<<8 | uint32(166+35*pulse)
	panelRule(pixels, width, height, row, color)
	marker := row
	marker.Right = marker.Left + 3
	fillPanelRect(pixels, width, height, marker, 0xff000000|uint32(95+30*pulse)<<16|uint32(156+30*pulse)<<8|uint32(215+25*pulse))
	if selected {
		panelRule(pixels, width, height, panelInset(row, 2), 0xffcda75c)
		fillPanelRect(pixels, width, height, marker, 0xffefd084)
	}
}
