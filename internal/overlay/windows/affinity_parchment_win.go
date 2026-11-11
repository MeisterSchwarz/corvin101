//go:build windows

package windows

import (
	"corvin101/internal/overlay/widget"
	"math"
	"unsafe"
)

// A softly feathered, irregular sheet rather than a framed rectangular panel.
func drawAffinityParchment(pixels unsafe.Pointer, width, height int32, bounds widget.Rect, scale float64) {
	buffer := unsafe.Slice((*uint32)(pixels), int(width*height))
	edge := math.Max(1, 2*scale)
	for y := max(0, bounds.Top); y < min(height, bounds.Bottom); y++ {
		for x := max(0, bounds.Left); x < min(width, bounds.Right); x++ {
			lx, ly := float64(x-bounds.Left), float64(y-bounds.Top)
			insetX := 1.5*scale + (math.Sin(ly/(5*scale))+math.Sin(ly/(11*scale)))*0.6*scale
			insetY := 1.5*scale + (math.Sin(lx/(7*scale))+math.Cos(lx/(13*scale)))*0.6*scale
			distance := math.Min(math.Min(lx-insetX, float64(bounds.Width()-1)-lx-insetX), math.Min(ly-insetY, float64(bounds.Height()-1)-ly-insetY))
			if distance <= 0 {
				continue
			}
			opacity := math.Min(1, distance/edge)
			shade := int32(math.Max(0, 1-distance/(8*scale)) * 25)
			noise := int32((uint32(x)*374761393^uint32(y)*668265263)%5) - 2
			r, g, b := uint32(224-shade+noise), uint32(197-shade+noise), uint32(143-shade+noise)
			source := uint32(240*opacity)<<24 | r<<16 | g<<8 | b
			index := int(y)*int(width) + int(x)
			buffer[index] = alphaBlendPixel(buffer[index], source)
		}
	}
}

// GDI clears alpha on text pixels. Keep the parchment's existing transparency.
func ensureAffinityTextAlpha(pixels unsafe.Pointer, width, height int32, bounds widget.Rect) {
	buffer := unsafe.Slice((*uint32)(pixels), int(width*height))
	for y := max(0, bounds.Top-4); y < min(height, bounds.Bottom+4); y++ {
		for x := max(0, bounds.Left-4); x < min(width, bounds.Right+4); x++ {
			index := int(y)*int(width) + int(x)
			p := buffer[index]
			if p>>24 == 0 && p&0xffffff != 0 {
				buffer[index] = p | 0xff000000
			}
		}
	}
}
