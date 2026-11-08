//go:build windows

package windows

import (
	"math"
	"unsafe"
)

// Floating-point bounds and alpha-aware bilinear sampling avoid whole-pixel
// jumps in the raven's size and center during its breathing animation.
func drawFloatingIcon(target unsafe.Pointer, width, height int32, icon *schoolIcon, cx, cy, size float64) {
	if icon == nil || size <= 0 {
		return
	}
	buffer := unsafe.Slice((*uint32)(target), int(width*height))
	left, top := cx-size/2, cy-size/2
	for y := max(0, int(math.Floor(top))); y < min(int(height), int(math.Ceil(top+size))); y++ {
		sy := (float64(y)+0.5-top)*float64(icon.height)/size - 0.5
		for x := max(0, int(math.Floor(left))); x < min(int(width), int(math.Ceil(left+size))); x++ {
			sx := (float64(x)+0.5-left)*float64(icon.width)/size - 0.5
			ix, iy := int(math.Floor(sx)), int(math.Floor(sy))
			fx, fy := sx-float64(ix), sy-float64(iy)
			var a, r, g, b float64
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					xx, yy := ix+dx, iy+dy
					if xx < 0 || xx >= icon.width || yy < 0 || yy >= icon.height {
						continue
					}
					wx, wy := 1-fx, 1-fy
					if dx == 1 {
						wx = fx
					}
					if dy == 1 {
						wy = fy
					}
					p := icon.pixels[yy*icon.width+xx]
					weight := wx * wy * float64(p>>24)
					a += weight
					r += weight * float64((p>>16)&255)
					g += weight * float64((p>>8)&255)
					b += weight * float64(p&255)
				}
			}
			if a > 0 {
				p := uint32(math.Round(a))<<24 | uint32(math.Round(r/a))<<16 | uint32(math.Round(g/a))<<8 | uint32(math.Round(b/a))
				index := y*int(width) + x
				buffer[index] = alphaBlendPixel(buffer[index], p)
			}
		}
	}
}
