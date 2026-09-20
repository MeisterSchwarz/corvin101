//go:build windows

package windows

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	biRGB = 0

	dibRGBColors = 0

	acSrcOver  = 0
	acSrcAlpha = 1

	transparentBackground = 1

	dtCenter     = 0x00000001
	dtVCenter    = 0x00000004
	dtSingleLine = 0x00000020

	fontWeightNormal = 400
	fontWeightBold   = 700
)

func (w *Window) render() {
	w.mu.RLock()

	hwnd := w.hwnd
	width := w.width
	height := w.height
	combat := w.combat

	w.mu.RUnlock()

	if hwnd == 0 ||
		width <= 0 ||
		height <= 0 {

		return
	}

	screenDC, _, _ :=
		user32.NewProc("GetDC").Call(0)

	if screenDC == 0 {
		return
	}

	defer user32.NewProc("ReleaseDC").Call(
		0,
		screenDC,
	)

	memoryDC, _, _ :=
		procCreateCompatibleDC.Call(
			screenDC,
		)

	if memoryDC == 0 {
		return
	}

	defer procDeleteDC.Call(
		memoryDC,
	)

	info := bitmapInfo{
		Header: bitmapInfoHeader{
			Size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
			Width:       width,
			Height:      -height,
			Planes:      1,
			BitCount:    32,
			Compression: biRGB,
		},
	}

	var pixels unsafe.Pointer

	bitmap, _, _ :=
		procCreateDIBSection.Call(
			memoryDC,
			uintptr(unsafe.Pointer(&info)),
			dibRGBColors,
			uintptr(unsafe.Pointer(&pixels)),
			0,
			0,
		)

	if bitmap == 0 {
		return
	}

	defer procDeleteObject.Call(
		bitmap,
	)

	oldBitmap, _, _ :=
		procSelectObject.Call(
			memoryDC,
			bitmap,
		)

	defer procSelectObject.Call(
		memoryDC,
		oldBitmap,
	)

	// Neue DIBSections sind bereits mit 0 initialisiert:
	//
	// A=0 R=0 G=0 B=0
	//
	// Das gesamte Overlay ist damit zunächst transparent.

	if combat.Visible {
		drawCombatWidgetARGB(
			memoryDC,
			pixels,
			width,
			height,
			combat.Round,
		)
	}

	source := point{
		X: 0,
		Y: 0,
	}

	windowSize := size{
		CX: width,
		CY: height,
	}

	blend := blendFunction{
		BlendOp:             acSrcOver,
		SourceConstantAlpha: 255,
		AlphaFormat:         acSrcAlpha,
	}

	procUpdateLayeredWindow.Call(
		uintptr(hwnd),
		screenDC,
		0,
		uintptr(
			unsafe.Pointer(
				&windowSize,
			),
		),
		memoryDC,
		uintptr(
			unsafe.Pointer(
				&source,
			),
		),
		0,
		uintptr(
			unsafe.Pointer(
				&blend,
			),
		),
		ulwAlpha,
	)
}

func drawCombatWidgetARGB(
	hdc uintptr,
	pixels unsafe.Pointer,
	windowWidth int32,
	windowHeight int32,
	round int,
) {
	scaleX :=
		float64(windowWidth) /
			1920.0

	scaleY :=
		float64(windowHeight) /
			1080.0

	scale := min(
		scaleX,
		scaleY,
	)

	if scale < 0.65 {
		scale = 0.65
	}

	widgetWidth :=
		int32(
			150 * scale,
		)

	widgetHeight :=
		int32(
			58 * scale,
		)

	centerX :=
		float64(windowWidth) * 0.5

	topY :=
		float64(windowHeight) * 0.67

	left :=
		int32(centerX) -
			widgetWidth/2

	top :=
		int32(topY)

	right :=
		left +
			widgetWidth

	bottom :=
		top +
			widgetHeight

	// Halbtransparenter anthrazitfarbener Hintergrund.
	fillARGBRect(
		pixels,
		windowWidth,
		windowHeight,
		left,
		top,
		right,
		bottom,
		190,
		24,
		24,
		28,
	)

	// Goldene Akzentlinie.
	accentHeight :=
		max(
			int32(2),
			int32(3*scale),
		)

	fillARGBRect(
		pixels,
		windowWidth,
		windowHeight,
		left,
		top,
		right,
		top+accentHeight,
		255,
		201,
		168,
		62,
	)

	drawRoundText(
		hdc,
		left,
		top,
		right,
		bottom,
		round,
		scale,
	)
}

func drawRoundText(
	hdc uintptr,
	left int32,
	top int32,
	right int32,
	bottom int32,
	round int,
	scale float64,
) {
	text :=
		fmt.Sprintf(
			"RUNDE  %d",
			round,
		)

	textUTF16, err :=
		windows.UTF16FromString(
			text,
		)

	if err != nil {
		return
	}

	fontName, err :=
		windows.UTF16PtrFromString(
			"Segoe UI",
		)

	if err != nil {
		return
	}

	fontHeight :=
		int32(
			-20 * scale,
		)

	font, _, _ :=
		procCreateFontW.Call(
			uintptr(fontHeight),
			0,
			0,
			0,
			fontWeightBold,
			0,
			0,
			0,
			1,
			0,
			0,
			5,
			0,
			uintptr(
				unsafe.Pointer(
					fontName,
				),
			),
		)

	if font == 0 {
		return
	}

	defer procDeleteObject.Call(
		font,
	)

	oldFont, _, _ :=
		procSelectObject.Call(
			hdc,
			font,
		)

	defer procSelectObject.Call(
		hdc,
		oldFont,
	)

	procSetBkMode.Call(
		hdc,
		transparentBackground,
	)

	procSetTextColor.Call(
		hdc,
		rgb(
			255,
			255,
			255,
		),
	)

	textRect := rect{
		Left:   left,
		Top:    top,
		Right:  right,
		Bottom: bottom,
	}

	procDrawTextW.Call(
		hdc,
		uintptr(
			unsafe.Pointer(
				&textUTF16[0],
			),
		),
		uintptr(
			len(textUTF16)-1,
		),
		uintptr(
			unsafe.Pointer(
				&textRect,
			),
		),
		dtCenter|
			dtVCenter|
			dtSingleLine,
	)
}

func fillARGBRect(
	pixels unsafe.Pointer,
	width int32,
	height int32,
	left int32,
	top int32,
	right int32,
	bottom int32,
	alpha byte,
	red byte,
	green byte,
	blue byte,
) {
	if pixels == nil {
		return
	}

	left = max(left, 0)
	top = max(top, 0)

	right = min(
		right,
		width,
	)

	bottom = min(
		bottom,
		height,
	)

	if left >= right ||
		top >= bottom {

		return
	}

	// UpdateLayeredWindow erwartet premultiplied alpha.
	a := uint32(alpha)

	r :=
		uint32(red) *
			a /
			255

	g :=
		uint32(green) *
			a /
			255

	b :=
		uint32(blue) *
			a /
			255

	color :=
		a<<24 |
			r<<16 |
			g<<8 |
			b

	pixelCount :=
		int(width * height)

	buffer :=
		unsafe.Slice(
			(*uint32)(pixels),
			pixelCount,
		)

	for y := top; y < bottom; y++ {
		offset :=
			int(y * width)

		for x := left; x < right; x++ {
			buffer[offset+int(x)] = color
		}
	}
}

func rgb(
	red byte,
	green byte,
	blue byte,
) uintptr {
	return uintptr(red) |
		uintptr(green)<<8 |
		uintptr(blue)<<16
}
