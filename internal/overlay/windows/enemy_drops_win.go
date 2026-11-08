//go:build windows

package windows

import (
	"corvin101/assets"
	"corvin101/internal/data/enemies"
	"corvin101/internal/overlay/widget"
	"log"
	"sync"
	"unsafe"
)

var itemIcons struct {
	once         sync.Once
	treasurecard *schoolIcon
}

func treasurecardIcon() *schoolIcon {
	itemIcons.once.Do(func() {
		file, err := assets.Items.Open("items/tc.png")
		if err != nil {
			log.Printf("[overlay] item icon: %v", err)
			return
		}
		defer file.Close()
		itemIcons.treasurecard, err = decodeIcon(file)
		if err != nil {
			log.Printf("[overlay] item icon: %v", err)
		}
	})
	return itemIcons.treasurecard
}

// Each category uses the same heading, icon slot and content spacing. Item IDs
// stay in Info even though this first version only renders the display names.
func drawEnemyDrops(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, top, fontSize int32, enemy enemies.EnemyInfo) {
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
		heading := widget.Rect{Left: bounds.Left, Top: top, Right: bounds.Right, Bottom: top + rowHeight}
		if heading.Bottom > bounds.Bottom {
			return
		}
		iconSize := rowHeight - 4
		if category.Key == "treasurecards" {
			drawIconImage(pixels, width, height, treasurecardIcon(), heading.Left+iconSize/2, heading.Top, iconSize)
			heading.Left += iconSize + 8
		}
		drawPanelLine(hdc, pixels, width, height, heading, category.Name, fontSize)
		rule := heading
		rule.Left = bounds.Left
		rule.Top = rule.Bottom - 1
		fillPanelRect(pixels, width, height, rule, 0xff6f563a)
		top = heading.Bottom + 4
		for _, item := range category.Items {
			line := widget.Rect{Left: bounds.Left + 6, Top: top, Right: bounds.Right, Bottom: top + rowHeight}
			if line.Bottom > bounds.Bottom {
				return
			}
			drawPanelLine(hdc, pixels, width, height, line, item.Name, fontSize)
			top += rowHeight
		}
		top += fontSize
	}
}
