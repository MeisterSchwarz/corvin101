//go:build windows

package windows

import (
	"corvin101/internal/overlay/theme"
	"corvin101/internal/overlay/widget"
	"golang.org/x/sys/windows"
	"unsafe"
)

func drawBestiaryText(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, text string, fontSize int32, prominent bool) {
	family, _ := windows.UTF16PtrFromString(theme.DisplayFontFamily)
	font, _, _ := procCreateFontW.Call(uintptr(-fontSize), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(family)))
	if font == 0 {
		return
	}
	defer procDeleteObject.Call(font)
	old, _, _ := procSelectObject.Call(hdc, font)
	defer procSelectObject.Call(hdc, old)
	value, err := windows.UTF16FromString(text)
	if err != nil {
		return
	}
	procSetBkMode.Call(hdc, transparentBackground)
	outline := int32(1)
	if prominent {
		outline = max(2, fontSize/14)
	}
	pass := func(dx, dy int32, color theme.RGB) {
		procSetTextColor.Call(hdc, rgb(color.R, color.G, color.B))
		area := rect{Left: bounds.Left + dx, Top: bounds.Top + dy, Right: bounds.Right + dx, Bottom: bounds.Bottom + dy}
		procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&value[0])), uintptr(len(value)-1), uintptr(unsafe.Pointer(&area)), dtVCenter|dtSingleLine|0x8000|0x800)
	}
	for y := -outline; y <= outline; y++ {
		for x := -outline; x <= outline; x++ {
			if x*x+y*y <= outline*outline {
				pass(x, y, theme.RoundOutlineColor)
			}
		}
	}
	color := theme.RoundColor
	if !prominent {
		color = theme.RGB{R: 231, G: 200, B: 110}
	}
	pass(0, 0, color)
	ensureTextAlpha(pixels, width, height, bounds)
}

func drawWorldArrow(pixels unsafe.Pointer, width, height int32, left, top, size int32, open bool) {
	for y := int32(0); y < size/2; y++ {
		span := size - 2*y
		yy := top + y
		if open {
			yy = top + size/2 - y
		}
		fillPanelRect(pixels, width, height, widget.Rect{Left: left + y, Top: yy, Right: left + y + span, Bottom: yy + 1}, 0xffffd326)
	}
}
