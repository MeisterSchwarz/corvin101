//go:build windows

package windows

import (
	"log"
	"sync"
	"unsafe"

	"corvin101/assets"
	"corvin101/internal/overlay/widget"
)

var detailToggleIcons struct {
	once            sync.Once
	backpack, enemy *schoolIcon
}

func detailToggleIcon(dropsView bool) *schoolIcon {
	detailToggleIcons.once.Do(func() {
		for path, target := range map[string]**schoolIcon{"items/backpack.png": &detailToggleIcons.backpack, "items/enemy.png": &detailToggleIcons.enemy} {
			file, err := assets.Items.Open(path)
			if err != nil {
				log.Printf("[overlay] button icon %s: %v", path, err)
				continue
			}
			icon, err := decodeIcon(file)
			file.Close()
			if err != nil {
				log.Printf("[overlay] button icon %s: %v", path, err)
				continue
			}
			*target = normalizeToggleIcon(icon)
		}
	})
	if dropsView {
		return detailToggleIcons.enemy
	}
	return detailToggleIcons.backpack
}

func detailToggleRect(width, height int32, dropsView bool) widget.Rect {
	_, bounds := enemyPanelColumns(width, height)
	bounds = panelInset(bounds, max(10, height/60))
	fontSize := max(int32(12), min(width, height)/50)
	top := bounds.Top
	size := max(int32(44), fontSize*3)
	return widget.Rect{Left: bounds.Right - size, Top: top, Right: bounds.Right, Bottom: top + size}
}

func drawDetailToggle(hdc uintptr, pixels unsafe.Pointer, width, height int32, dropsView bool, cursor widget.Point) {
	button := detailToggleRect(width, height, dropsView)
	size := button.Width() * 86 / 100
	if button.Contains(cursor) {
		size = button.Width()
	}
	drawIconImage(pixels, width, height, detailToggleIcon(dropsView), (button.Left+button.Right)/2, button.Top+(button.Height()-size)/2, size)
}

// Equalize the visible silhouettes rather than the transparent PNG canvases.
func normalizeToggleIcon(source *schoolIcon) *schoolIcon {
	left, top, right, bottom := source.width, source.height, 0, 0
	for y := 0; y < source.height; y++ {
		for x := 0; x < source.width; x++ {
			if source.pixels[y*source.width+x]>>24 > 8 {
				left = min(left, x)
				top = min(top, y)
				right = max(right, x+1)
				bottom = max(bottom, y+1)
			}
		}
	}
	if right <= left || bottom <= top {
		return source
	}
	w, h := right-left, bottom-top
	side := max(w, h) + 4
	result := &schoolIcon{width: side, height: side, pixels: make([]uint32, side*side)}
	dx, dy := (side-w)/2, (side-h)/2
	for y := 0; y < h; y++ {
		copy(result.pixels[(y+dy)*side+dx:(y+dy)*side+dx+w], source.pixels[(y+top)*source.width+left:(y+top)*source.width+right])
	}
	return result
}
