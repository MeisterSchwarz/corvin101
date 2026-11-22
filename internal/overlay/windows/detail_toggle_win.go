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
	once                         sync.Once
	backpack, resistance, cheats *schoolIcon
}

// Caller holds w.mu; hit testing shares the exact drawing layout.
func (w *Window) detailTabAtLocked(p widget.Point) int {
	if !w.panelOpen || w.worldMenuOpen || w.panelLoading || w.selectedEnemy == "" {
		return -1
	}
	hasCheats := w.hasSelectedCheatsLocked()
	count := 2
	if hasCheats {
		count = 3
	}
	for i := 0; i < count; i++ {
		if detailTabRect(w.width, w.height, i, hasCheats).Contains(p) {
			return i
		}
	}
	return -1
}

func detailToggleIcon(dropsView bool) *schoolIcon {
	detailToggleIcons.once.Do(func() {
		for path, target := range map[string]**schoolIcon{"items/backpack.png": &detailToggleIcons.backpack, "items/resistance.png": &detailToggleIcons.resistance, "items/cheats.png": &detailToggleIcons.cheats} {
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
		return detailToggleIcons.resistance
	}
	return detailToggleIcons.backpack
}

func detailToggleRect(width, height int32, dropsView bool) widget.Rect {
	index := 0
	if dropsView {
		index = 1
	}
	return detailTabRect(width, height, index, false)
}

func detailTabRect(width, height int32, index int, hasCheats bool) widget.Rect {
	_, bounds := enemyPanelColumns(width, height)
	bounds = panelInset(bounds, max(10, height/60))
	fontSize := max(int32(12), min(width, height)/50)
	top := bounds.Top
	size := max(int32(44), fontSize*3)
	count := int32(2)
	if hasCheats {
		count = 3
		if index == 2 {
			index = 0
		} else {
			index++
		}
	}
	left := bounds.Right - count*size - (count-1)*8 + int32(index)*(size+8)
	return widget.Rect{Left: left, Top: top, Right: left + size, Bottom: top + size}
}

func drawDetailTabs(pixels unsafe.Pointer, width, height int32, dropsView, cheatsView, hasCheats bool, cursor widget.Point) {
	detailToggleIcon(false)
	icons := []*schoolIcon{detailToggleIcons.resistance, detailToggleIcons.backpack}
	if hasCheats {
		icons = append(icons, detailToggleIcons.cheats)
	}
	for i, icon := range icons {
		button := detailTabRect(width, height, i, hasCheats)
		size := button.Width() * 86 / 100
		if button.Contains(cursor) {
			size = button.Width()
		}
		drawIconImage(pixels, width, height, icon, (button.Left+button.Right)/2, button.Top+(button.Height()-size)/2, size)
	}
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
