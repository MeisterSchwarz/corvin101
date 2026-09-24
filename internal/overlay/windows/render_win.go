//go:build windows

package windows

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"

	"corvin101/internal/overlay/theme"
	"corvin101/internal/overlay/widget"
	"corvin101/internal/overlay/widgets"
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

	fontWeightBold = 700
)

func (w *Window) render() {
	w.mu.RLock()

	hwnd := w.hwnd
	width := w.width
	height := w.height
	cursor := w.cursor
	cursorKnown := w.cursorKnown
	manager := w.manager

	var entries []widget.Entry
	if manager != nil {
		entries = manager.Entries(
			widget.Viewport{
				Width:  width,
				Height: height,
			},
			cursor,
			cursorKnown,
		)
	}

	w.mu.RUnlock()

	if hwnd == 0 || width <= 0 || height <= 0 {
		return
	}

	screenDC, _, _ := user32.NewProc("GetDC").Call(0)
	if screenDC == 0 {
		return
	}
	defer user32.NewProc("ReleaseDC").Call(0, screenDC)

	memoryDC, _, _ := procCreateCompatibleDC.Call(screenDC)
	if memoryDC == 0 {
		return
	}
	defer procDeleteDC.Call(memoryDC)

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

	bitmap, _, _ := procCreateDIBSection.Call(
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
	defer procDeleteObject.Call(bitmap)

	oldBitmap, _, _ := procSelectObject.Call(memoryDC, bitmap)
	defer procSelectObject.Call(memoryDC, oldBitmap)

	for _, entry := range entries {
		switch current := entry.Widget.(type) {
		case *widgets.Round:
			drawRoundWidgetARGB(
				memoryDC,
				pixels,
				width,
				height,
				entry.Bounds,
				current.Round(),
				current.Scale(),
				current.Highlight(),
			)
		}
	}

	source := point{X: 0, Y: 0}
	windowSize := size{CX: width, CY: height}

	blend := blendFunction{
		BlendOp:             acSrcOver,
		SourceConstantAlpha: 255,
		AlphaFormat:         acSrcAlpha,
	}

	procUpdateLayeredWindow.Call(
		uintptr(hwnd),
		screenDC,
		0,
		uintptr(unsafe.Pointer(&windowSize)),
		memoryDC,
		uintptr(unsafe.Pointer(&source)),
		0,
		uintptr(unsafe.Pointer(&blend)),
		ulwAlpha,
	)
}

func drawRoundWidgetARGB(
	hdc uintptr,
	pixels unsafe.Pointer,
	windowWidth int32,
	windowHeight int32,
	bounds widget.Rect,
	round int,
	animationScale float64,
	highlight float64,
) {
	scaleX := float64(windowWidth) / 1920.0
	scaleY := float64(windowHeight) / 1080.0
	uiScale := min(scaleX, scaleY)

	if uiScale < 0.65 {
		uiScale = 0.65
	}

	fontSize := int32(
		theme.RoundFontSize *
			uiScale *
			animationScale,
	)

	text := fmt.Sprintf("%d", round)

	drawOutlinedText(
		hdc,
		pixels,
		windowWidth,
		windowHeight,
		bounds,
		text,
		fontSize,
		uiScale,
		highlight,
	)
}

func drawOutlinedText(
	hdc uintptr,
	pixels unsafe.Pointer,
	windowWidth int32,
	windowHeight int32,
	bounds widget.Rect,
	text string,
	fontSize int32,
	scale float64,
	highlight float64,
) {
	textUTF16, err := windows.UTF16FromString(text)
	if err != nil || len(textUTF16) == 0 {
		return
	}

	fontName, err := windows.UTF16PtrFromString(theme.DisplayFontFamily)
	if err != nil {
		return
	}

	font, _, _ := procCreateFontW.Call(
		uintptr(-fontSize),
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
		uintptr(unsafe.Pointer(fontName)),
	)
	if font == 0 {
		return
	}
	defer procDeleteObject.Call(font)

	oldFont, _, _ := procSelectObject.Call(hdc, font)
	defer procSelectObject.Call(hdc, oldFont)

	procSetBkMode.Call(hdc, transparentBackground)

	outline := max(
		int32(2),
		int32(float64(theme.RoundOutlineSize)*scale),
	)

	shadowX := int32(float64(theme.RoundShadowX) * scale)
	shadowY := int32(float64(theme.RoundShadowY) * scale)

	// Schatten.
	drawTextPass(
		hdc,
		textUTF16,
		bounds,
		shadowX,
		shadowY,
		theme.RoundShadowColor,
	)

	// Outline.
	for y := -outline; y <= outline; y++ {
		for x := -outline; x <= outline; x++ {
			if x == 0 && y == 0 {
				continue
			}

			if x*x+y*y > outline*outline {
				continue
			}

			drawTextPass(
				hdc,
				textUTF16,
				bounds,
				x,
				y,
				theme.RoundOutlineColor,
			)
		}
	}

	// Goldene Zahl. Während des Bounce kurz heller.
	mainColor := interpolateColor(
		theme.RoundColor,
		theme.RoundHighlightColor,
		highlight,
	)

	drawTextPass(
		hdc,
		textUTF16,
		bounds,
		0,
		0,
		mainColor,
	)

	ensureTextAlpha(
		pixels,
		windowWidth,
		windowHeight,
		bounds,
	)
}

func drawTextPass(
	hdc uintptr,
	text []uint16,
	bounds widget.Rect,
	offsetX int32,
	offsetY int32,
	color theme.RGB,
) {
	procSetTextColor.Call(
		hdc,
		rgb(color.R, color.G, color.B),
	)

	textRect := rect{
		Left:   bounds.Left + offsetX,
		Top:    bounds.Top + offsetY,
		Right:  bounds.Right + offsetX,
		Bottom: bounds.Bottom + offsetY,
	}

	procDrawTextW.Call(
		hdc,
		uintptr(unsafe.Pointer(&text[0])),
		uintptr(len(text)-1),
		uintptr(unsafe.Pointer(&textRect)),
		dtCenter|dtVCenter|dtSingleLine,
	)
}

func ensureTextAlpha(
	pixels unsafe.Pointer,
	width int32,
	height int32,
	bounds widget.Rect,
) {
	if pixels == nil || width <= 0 || height <= 0 {
		return
	}

	// Etwas Platz für Outline, Schatten und Bounce.
	padding := int32(20)

	left := max(bounds.Left-padding, int32(0))
	top := max(bounds.Top-padding, int32(0))
	right := min(bounds.Right+padding, width)
	bottom := min(bounds.Bottom+padding, height)

	if left >= right || top >= bottom {
		return
	}

	buffer := unsafe.Slice(
		(*uint32)(pixels),
		int(width*height),
	)

	for y := top; y < bottom; y++ {
		offset := int(y * width)

		for x := left; x < right; x++ {
			index := offset + int(x)
			pixel := buffer[index]

			if pixel&0x00FFFFFF == 0 {
				continue
			}

			buffer[index] = 0xFF000000 | (pixel & 0x00FFFFFF)
		}
	}
}

func interpolateColor(
	from theme.RGB,
	to theme.RGB,
	t float64,
) theme.RGB {
	if t < 0 {
		t = 0
	}

	if t > 1 {
		t = 1
	}

	return theme.RGB{
		R: interpolateByte(from.R, to.R, t),
		G: interpolateByte(from.G, to.G, t),
		B: interpolateByte(from.B, to.B, t),
	}
}

func interpolateByte(
	from byte,
	to byte,
	t float64,
) byte {
	value := float64(from) +
		(float64(to)-float64(from))*t

	return byte(value)
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
