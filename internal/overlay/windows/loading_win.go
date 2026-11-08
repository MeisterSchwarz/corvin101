//go:build windows

package windows

import (
	"encoding/binary"
	"math"
	"sync"
	"time"
	"unsafe"

	"corvin101/internal/overlay/theme"
	"corvin101/internal/overlay/widget"
	"corvin101/public"
)

var ravenLoadOnce sync.Once
var loadingRaven *schoolIcon

// The bundled ICO contains a 32-bit, bottom-up BGRA bitmap.
func loadLoadingRaven() *schoolIcon {
	ravenLoadOnce.Do(func() {
		data := public.TrayIcon
		if len(data) < 22 {
			return
		}
		offset := int(binary.LittleEndian.Uint32(data[18:22]))
		if offset < 0 || offset+40 > len(data) {
			return
		}
		header := data[offset:]
		w, h := int(binary.LittleEndian.Uint32(header[4:8])), int(binary.LittleEndian.Uint32(header[8:12]))/2
		headerSize := int(binary.LittleEndian.Uint32(header[:4]))
		if w <= 0 || h <= 0 || w > 1024 || h > 1024 || headerSize < 40 || binary.LittleEndian.Uint16(header[14:16]) != 32 || binary.LittleEndian.Uint32(header[16:20]) != 0 || headerSize+w*h*4 > len(header) {
			return
		}
		icon := &schoolIcon{width: w, height: h, pixels: make([]uint32, w*h)}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				pos := headerSize + ((h-1-y)*w+x)*4
				icon.pixels[y*w+x] = binary.LittleEndian.Uint32(header[pos : pos+4])
			}
		}
		loadingRaven = icon
	})
	return loadingRaven
}

func drawEnemyLoading(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, started time.Time) {
	fontSize := max(int32(24), min(width, height)/25)
	centerX, centerY := (bounds.Left+bounds.Right)/2, (bounds.Top+bounds.Bottom)/2
	elapsed := time.Since(started).Seconds()
	baseSize := max(int32(64), height/9)
	iconSize := float64(baseSize) * (1 + 0.035*math.Sin(elapsed*2*math.Pi/1.2))
	raven := loadLoadingRaven()
	if raven != nil {
		drawFloatingIcon(pixels, width, height, smoothIcon(raven, int(baseSize)+8), float64(centerX), float64(centerY)-float64(baseSize)/2, iconSize)
	}
	text := "Gegner werden geladen …"
	textWidth := panelFontTextWidth(hdc, text, fontSize, theme.DisplayFontFamily)
	line := widget.Rect{Left: max(bounds.Left+16, centerX-textWidth/2), Top: centerY + fontSize/2, Right: min(bounds.Right-16, centerX+textWidth/2+6), Bottom: centerY + fontSize*2}
	drawBestiaryText(hdc, pixels, width, height, line, text, fontSize, true)
}
