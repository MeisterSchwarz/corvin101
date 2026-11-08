//go:build windows

package windows

import "math"

// Filter covered source pixels in premultiplied alpha to avoid dark fringes.
// Cache each display size; the per-frame draw only copies the filtered bitmap.
func smoothIcon(source *schoolIcon, size int) *schoolIcon {
	if source.width == size && source.height == size {
		return source
	}
	if cached, ok := source.resized.Load(size); ok {
		return cached.(*schoolIcon)
	}
	result := &schoolIcon{width: size, height: size, pixels: make([]uint32, size*size)}
	sx, sy := float64(source.width)/float64(size), float64(source.height)/float64(size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			cx, cy := (float64(x)+0.5)*sx, (float64(y)+0.5)*sy
			x0, x1 := math.Max(0, cx-math.Max(1, sx)/2), math.Min(float64(source.width), cx+math.Max(1, sx)/2)
			y0, y1 := math.Max(0, cy-math.Max(1, sy)/2), math.Min(float64(source.height), cy+math.Max(1, sy)/2)
			var weight, alpha, red, green, blue float64
			for iy := int(y0); iy < int(math.Ceil(y1)); iy++ {
				for ix := int(x0); ix < int(math.Ceil(x1)); ix++ {
					w := (math.Min(x1, float64(ix+1)) - math.Max(x0, float64(ix))) * (math.Min(y1, float64(iy+1)) - math.Max(y0, float64(iy)))
					p := source.pixels[iy*source.width+ix]
					a := float64(p >> 24)
					weight += w
					alpha += a * w
					red += float64((p>>16)&255) * a * w
					green += float64((p>>8)&255) * a * w
					blue += float64(p&255) * a * w
				}
			}
			if alpha > 0 && weight > 0 {
				result.pixels[y*size+x] = uint32(math.Round(alpha/weight))<<24 | uint32(math.Round(red/alpha))<<16 | uint32(math.Round(green/alpha))<<8 | uint32(math.Round(blue/alpha))
			}
		}
	}
	actual, _ := source.resized.LoadOrStore(size, result)
	return actual.(*schoolIcon)
}
