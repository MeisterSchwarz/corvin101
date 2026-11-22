//go:build windows

package windows

import (
	"bytes"
	"corvin101/assets"
	"log"
	"sync"
)

var challengeIcons struct {
	once             sync.Once
	active, inactive *schoolIcon
}

func challengeModeIcon(active bool) *schoolIcon {
	challengeIcons.once.Do(func() {
		icon, err := decodeIcon(bytes.NewReader(assets.Challenge))
		if err != nil {
			log.Printf("[overlay] challenge icon: %v", err)
			return
		}
		challengeIcons.active = normalizeToggleIcon(icon)
		gray := &schoolIcon{width: challengeIcons.active.width, height: challengeIcons.active.height, pixels: append([]uint32(nil), challengeIcons.active.pixels...)}
		for i, p := range gray.pixels {
			r, g, b := (p>>16)&255, (p>>8)&255, p&255
			// Map the red disc to neutral gray while retaining white and black.
			v := (r + g + b) / 3
			gray.pixels[i] = (p & 0xff000000) | (v << 16) | (v << 8) | v
		}
		challengeIcons.inactive = gray
	})
	if active {
		return challengeIcons.active
	}
	return challengeIcons.inactive
}
