//go:build windows

package windows

import (
	"corvin101/assets"
	"corvin101/internal/data/enemies"
	"corvin101/internal/overlay/widget"
	"fmt"
	"log"
	"sync"
	"unsafe"
)

var itemIcons struct {
	once         sync.Once
	treasurecard *schoolIcon
	gold         *schoolIcon
	reagent      *schoolIcon
}

func treasurecardIcon() *schoolIcon {
	itemIcons.once.Do(func() {
		for path, target := range map[string]**schoolIcon{"items/tc.png": &itemIcons.treasurecard, "items/gold.png": &itemIcons.gold, "items/rg.png": &itemIcons.reagent} {
			file, err := assets.Items.Open(path)
			if err != nil {
				log.Printf("[overlay] item icon %s: %v", path, err)
				continue
			}
			icon, err := decodeIcon(file)
			file.Close()
			if err != nil {
				log.Printf("[overlay] item icon %s: %v", path, err)
				continue
			}
			*target = icon
		}
	})
	return itemIcons.treasurecard
}

// Each category uses the same heading, icon slot and content spacing. Item IDs
// stay in Info even though this first version only renders the display names.
func drawEnemyDrops(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, top, fontSize int32, enemy enemies.EnemyInfo, selected string) {
	rowHeight := max(int32(26), fontSize*2)
	if enemy.DropsUnavailable {
		line := bounds
		line.Top = top
		line.Bottom = top + rowHeight
		if line.Bottom <= bounds.Bottom {
			drawPanelLine(hdc, pixels, width, height, line, "Dropdaten derzeit nicht verfügbar", fontSize)
		}
		return
	}
	if len(enemy.DropCategories) == 0 {
		line := bounds
		line.Top = top
		line.Bottom = top + rowHeight
		if line.Bottom <= bounds.Bottom {
			drawPanelLine(hdc, pixels, width, height, line, "Noch keine Drops hinterlegt", fontSize)
		}
		return
	}
	for _, category := range enemy.DropCategories {
		if category.Gold == nil {
			continue
		}
		line := widget.Rect{Left: bounds.Left, Top: top, Right: bounds.Right, Bottom: top + rowHeight}
		if line.Bottom > bounds.Bottom {
			return
		}
		treasurecardIcon() // Load the embedded category icons once.
		iconSize := rowHeight - 2
		drawIconImage(pixels, width, height, itemIcons.gold, line.Left+iconSize/2, top, iconSize)
		line.Left += iconSize + 8
		text := fmt.Sprintf("%d–%d", category.Gold.Min, category.Gold.Max)
		if category.Gold.Min == category.Gold.Max {
			text = fmt.Sprintf("%d", category.Gold.Min)
		}
		drawBestiaryText(hdc, pixels, width, height, line, text, fontSize+3, false)
		top += rowHeight + fontSize/2
	}
	treasurecardIcon()
	for i, icon := range []*schoolIcon{itemIcons.treasurecard, itemIcons.reagent} {
		button := dropCategoryRect(bounds, top, rowHeight, i)
		drawIconImage(pixels, width, height, icon, (button.Left+button.Right)/2, button.Top+2, rowHeight-4)
		if selected == "reagents" && i == 1 || selected != "reagents" && i == 0 {
			fillPanelRect(pixels, width, height, widget.Rect{Left: button.Left, Right: button.Right, Top: button.Bottom, Bottom: button.Bottom + 2}, 0xffd3ad55)
		}
	}
	top += rowHeight + 8
	fillPanelRect(pixels, width, height, widget.Rect{Left: bounds.Left, Right: bounds.Right, Top: top - 4, Bottom: top - 3}, 0xff6f563a)
	if selected == "" {
		selected = "treasurecards"
	}
	for _, category := range enemy.DropCategories {
		if category.Key != selected {
			continue
		}
		for _, item := range category.Items {
			line := widget.Rect{Left: bounds.Left + 6, Top: top, Right: bounds.Right, Bottom: top + rowHeight}
			if line.Bottom > bounds.Bottom {
				return
			}
			drawPanelLine(hdc, pixels, width, height, line, item.Name, fontSize)
			top += rowHeight
		}
		return
	}
	drawPanelLine(hdc, pixels, width, height, widget.Rect{Left: bounds.Left + 6, Right: bounds.Right, Top: top, Bottom: top + rowHeight}, "Noch keine Drops hinterlegt", fontSize)
}

func dropCategoryRect(bounds widget.Rect, top, rowHeight int32, index int) widget.Rect {
	left := bounds.Left + int32(index)*(rowHeight+14)
	return widget.Rect{Left: left, Right: left + rowHeight, Top: top, Bottom: top + rowHeight}
}

// Caller holds w.mu. Match the detail header and optional gold row.
func (w *Window) dropCategoryAtLocked(p widget.Point) string {
	if !w.detailDrops || w.selectedEnemy == "" {
		return ""
	}
	_, bounds := enemyPanelColumns(w.width, w.height)
	bounds = panelInset(bounds, max(10, w.height/60))
	fontSize := max(int32(12), min(w.width, w.height)/50)
	rowHeight := max(int32(26), fontSize*2)
	top := bounds.Top + fontSize*5
	for _, enemy := range w.panelEntriesLocked() {
		if enemy.Key != w.selectedEnemy {
			continue
		}
		if enemy.DropsUnavailable || len(enemy.DropCategories) == 0 {
			return ""
		}
		for _, category := range enemy.DropCategories {
			if category.Gold != nil {
				top += rowHeight + fontSize/2
			}
		}
		for i, key := range []string{"treasurecards", "reagents"} {
			if dropCategoryRect(bounds, top, rowHeight, i).Contains(p) {
				return key
			}
		}
		break
	}
	return ""
}
