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

	if hwnd == 0 ||
		width <= 0 ||
		height <= 0 {

		return
	}

	screenDC, _, _ :=
		user32.NewProc(
			"GetDC",
		).Call(0)

	if screenDC == 0 {
		return
	}

	defer user32.NewProc(
		"ReleaseDC",
	).Call(
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
			Size: uint32(
				unsafe.Sizeof(
					bitmapInfoHeader{},
				),
			),

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
			uintptr(
				unsafe.Pointer(
					&info,
				),
			),
			dibRGBColors,
			uintptr(
				unsafe.Pointer(
					&pixels,
				),
			),
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

	for _, entry := range entries {
		switch current :=
			entry.Widget.(type) {

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

		case *widgets.EnemyAffinities:
			drawEnemyAffinitiesARGB(
				memoryDC,
				pixels,
				width,
				height,
				current.Entries(),
				cursor,
				cursorKnown,
			)
		}
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

func drawEnemyAffinitiesARGB(
	hdc uintptr,
	pixels unsafe.Pointer,
	windowWidth int32,
	windowHeight int32,
	entries []widgets.EnemyAffinityEntry,
	cursor widget.Point,
	cursorKnown bool,
) {
	if len(entries) == 0 {
		return
	}

	scaleX := float64(windowWidth) / 1920.0
	scaleY := float64(windowHeight) / 1080.0
	uiScale := min(scaleX, scaleY)

	if uiScale < 0.65 {
		uiScale = 0.65
	}

	for _, enemy := range entries {
		slot := enemy.Slot

		if slot < 0 ||
			slot >= 4 ||
			len(enemy.Groups) == 0 {

			continue
		}

		bounds :=
			enemyAffinitySlotBounds(
				windowWidth,
				windowHeight,
				slot,
				len(enemy.Groups),
				uiScale,
			)

		if cursorKnown &&
			bounds.Contains(cursor) {

			continue
		}

		drawEnemyAffinityGroups(
			hdc,
			pixels,
			windowWidth,
			windowHeight,
			bounds,
			enemy.Groups,
			uiScale,
		)
	}
}

func drawEnemyAffinityGroups(
	hdc uintptr,
	pixels unsafe.Pointer,
	windowWidth int32,
	windowHeight int32,
	bounds widget.Rect,
	groups []widgets.EnemyAffinityGroup,
	scale float64,
) {
	if len(groups) == 0 {
		return
	}

	groupCount := int32(len(groups))

	availableWidth :=
		bounds.Right -
			bounds.Left

	if availableWidth <= 0 {
		return
	}

	columnWidth := int32(52.0 * scale)
	contentWidth := columnWidth * groupCount

	if contentWidth > availableWidth {
		columnWidth =
			availableWidth /
				groupCount

		contentWidth =
			columnWidth *
				groupCount
	}

	startX :=
		bounds.Left +
			(availableWidth-contentWidth)/2

	valueHeight := int32(22.0 * scale)
	iconSize := int32(32.0 * scale)
	iconGap := int32(-6.0 * scale)

	for index, group := range groups {
		columnLeft :=
			startX +
				int32(index)*
					columnWidth

		columnRight :=
			columnLeft +
				columnWidth

		centerX :=
			columnLeft +
				columnWidth/2

		valueBounds := widget.Rect{
			Left:   columnLeft,
			Top:    bounds.Top,
			Right:  columnRight,
			Bottom: bounds.Top + valueHeight,
		}

		drawAffinityValue(
			hdc,
			pixels,
			windowWidth,
			windowHeight,
			valueBounds,
			group.Value,
			scale,
		)

		iconTop :=
			valueBounds.Bottom -
				int32(2.0*scale)

		for _, school := range group.Schools {
			drawSchoolIcon(
				pixels,
				windowWidth,
				windowHeight,
				school,
				centerX,
				iconTop,
				iconSize,
			)

			iconTop +=
				iconSize +
					iconGap
		}
	}

}

func drawAffinityValue(
	hdc uintptr,
	pixels unsafe.Pointer,
	windowWidth int32,
	windowHeight int32,
	bounds widget.Rect,
	value int,
	scale float64,
) {
	text :=
		fmt.Sprintf(
			"%+d%%",
			value,
		)

	textUTF16, err :=
		windows.UTF16FromString(
			text,
		)

	if err != nil ||
		len(textUTF16) == 0 {

		return
	}

	fontName, err :=
		windows.UTF16PtrFromString(
			theme.UIFontFamily,
		)

	if err != nil {
		return
	}

	fontSize :=
		int32(
			theme.AffinityFontSize *
				scale,
		)

	font, _, _ :=
		procCreateFontW.Call(
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

	outline :=
		max(
			int32(1),
			int32(
				float64(
					theme.AffinityOutlineSize,
				)*scale,
			),
		)

	for y := -outline; y <= outline; y++ {
		for x := -outline; x <= outline; x++ {
			if x == 0 &&
				y == 0 {

				continue
			}

			if x*x+y*y >
				outline*outline {

				continue
			}

			drawTextPass(
				hdc,
				textUTF16,
				bounds,
				x,
				y,
				theme.AffinityOutlineColor,
			)
		}
	}

	drawTextPass(
		hdc,
		textUTF16,
		bounds,
		0,
		0,
		theme.AffinityResistColor,
	)

	ensureTextAlpha(
		pixels,
		windowWidth,
		windowHeight,
		bounds,
	)
}

func enemyAffinitySlotBounds(
	width int32,
	height int32,
	slot int,
	groupCount int,
	scale float64,
) widget.Rect {
	if slot < 0 ||
		slot >= 4 ||
		groupCount <= 0 {

		return widget.Rect{}
	}

	// Kalibrierung des Wizard101 Enemy-HUDs.
	//
	// firstHPRight:
	// rechte Kante der ersten Enemy-Healthbar.
	//
	// enemySpacing:
	// horizontaler Abstand von einer Healthbar
	// zur nächsten.
	//
	// gap:
	// Abstand zwischen Healthbar und Affinity-Block.
	const (
		firstHPRight = 0.156
		enemySpacing = 0.223
	)

	gap := 8.0 * scale

	hpRight :=
		float64(width) *
			(firstHPRight +
				float64(slot)*enemySpacing)

	left :=
		int32(hpRight + gap)

	top :=
		int32(
			float64(height) *
				0.018,
		)

	columnWidth :=
		int32(
			42.0 *
				scale,
		)

	padding :=
		int32(
			5.0 *
				scale,
		)

	contentWidth :=
		int32(groupCount) *
			columnWidth

	slotHeight :=
		int32(
			145.0 *
				scale,
		)

	return widget.Rect{
		Left: left -
			padding,

		Top: top,

		Right: left +
			contentWidth +
			padding,

		Bottom: top +
			slotHeight,
	}
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
	scaleX :=
		float64(windowWidth) /
			1920.0

	scaleY :=
		float64(windowHeight) /
			1080.0

	uiScale := min(
		scaleX,
		scaleY,
	)

	if uiScale < 0.65 {
		uiScale = 0.65
	}

	fontSize :=
		int32(
			theme.RoundFontSize *
				uiScale *
				animationScale,
		)

	text :=
		fmt.Sprintf(
			"%d",
			round,
		)

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
	textUTF16, err :=
		windows.UTF16FromString(
			text,
		)

	if err != nil ||
		len(textUTF16) == 0 {

		return
	}

	fontName, err :=
		windows.UTF16PtrFromString(
			theme.DisplayFontFamily,
		)

	if err != nil {
		return
	}

	font, _, _ :=
		procCreateFontW.Call(
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

	outline :=
		max(
			int32(2),
			int32(
				float64(
					theme.RoundOutlineSize,
				)*scale,
			),
		)

	shadowX :=
		int32(
			float64(
				theme.RoundShadowX,
			) * scale,
		)

	shadowY :=
		int32(
			float64(
				theme.RoundShadowY,
			) * scale,
		)

	drawTextPass(
		hdc,
		textUTF16,
		bounds,
		shadowX,
		shadowY,
		theme.RoundShadowColor,
	)

	for y := -outline; y <= outline; y++ {
		for x := -outline; x <= outline; x++ {
			if x == 0 &&
				y == 0 {

				continue
			}

			if x*x+y*y >
				outline*outline {

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

	mainColor :=
		interpolateColor(
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
		rgb(
			color.R,
			color.G,
			color.B,
		),
	)

	textRect := rect{
		Left: bounds.Left +
			offsetX,

		Top: bounds.Top +
			offsetY,

		Right: bounds.Right +
			offsetX,

		Bottom: bounds.Bottom +
			offsetY,
	}

	procDrawTextW.Call(
		hdc,
		uintptr(
			unsafe.Pointer(
				&text[0],
			),
		),
		uintptr(
			len(text)-1,
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

func ensureTextAlpha(
	pixels unsafe.Pointer,
	width int32,
	height int32,
	bounds widget.Rect,
) {
	if pixels == nil ||
		width <= 0 ||
		height <= 0 {

		return
	}

	padding := int32(20)

	left :=
		max(
			bounds.Left-padding,
			int32(0),
		)

	top :=
		max(
			bounds.Top-padding,
			int32(0),
		)

	right :=
		min(
			bounds.Right+padding,
			width,
		)

	bottom :=
		min(
			bounds.Bottom+padding,
			height,
		)

	if left >= right ||
		top >= bottom {

		return
	}

	buffer :=
		unsafe.Slice(
			(*uint32)(pixels),
			int(width*height),
		)

	for y := top; y < bottom; y++ {
		offset :=
			int(
				y *
					width,
			)

		for x := left; x < right; x++ {
			index :=
				offset +
					int(x)

			pixel :=
				buffer[index]

			if pixel&0x00FFFFFF == 0 {
				continue
			}

			buffer[index] =
				0xFF000000 |
					(pixel & 0x00FFFFFF)
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
		R: interpolateByte(
			from.R,
			to.R,
			t,
		),

		G: interpolateByte(
			from.G,
			to.G,
			t,
		),

		B: interpolateByte(
			from.B,
			to.B,
			t,
		),
	}
}

func interpolateByte(
	from byte,
	to byte,
	t float64,
) byte {
	value :=
		float64(from) +
			(float64(to)-
				float64(from))*
				t

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
