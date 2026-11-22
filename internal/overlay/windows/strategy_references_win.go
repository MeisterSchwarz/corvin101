//go:build windows

package windows

import (
	"corvin101/internal/data/enemies"
	"corvin101/internal/overlay/theme"
	"corvin101/internal/overlay/widget"
	"regexp"
	"strings"
	"unsafe"
)

type strategyEnemyLink struct {
	Bounds       widget.Rect
	Family, Mode string
	Head         string
}

var strategyTokenRe = regexp.MustCompile(`\{[A-Za-z0-9_]+\}|[^\s{}]+|\s+`)

func drawStrategyParagraph(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, top, fontSize int32, note enemies.CheatNote, colors ...theme.RGB) (int32, []strategyEnemyLink) {
	baseColor := theme.RGB{R: 213, G: 210, B: 200}
	if len(colors) > 0 {
		baseColor = colors[0]
	}
	if len(note.Enemies) == 0 {
		return drawWrappedCheatText(hdc, pixels, width, height, bounds, top, fontSize, note.Text, baseColor), nil
	}
	refs := map[string]enemies.CheatEnemyReference{}
	for _, ref := range note.Enemies {
		refs["{"+ref.Key+"}"] = ref
	}
	x, y := bounds.Left, top
	lineHeight := fontSize + fontSize/2 + 2
	iconSize := fontSize + 3
	var links []strategyEnemyLink
	for _, token := range strategyTokenRe.FindAllString(note.Text, -1) {
		ref, isRef := refs[token]
		text := token
		if isRef {
			text = ref.Label
		}
		wordWidth := panelTextWidth(hdc, text, fontSize)
		if isRef && len(ref.Schools) > 0 {
			wordWidth += panelTextWidth(hdc, " ()", fontSize) + int32(len(ref.Schools))*(iconSize+3)
		}
		space := strings.TrimSpace(token) == ""
		if strings.Contains(token, "\n") {
			x = bounds.Left
			y += lineHeight
			continue
		}
		if x+wordWidth > bounds.Right-6 && x > bounds.Left {
			x = bounds.Left
			y += lineHeight
			if space {
				continue
			}
		}
		if x == bounds.Left && space {
			continue
		}
		area := widget.Rect{Left: x, Right: min(bounds.Right-4, x+wordWidth+1), Top: y, Bottom: y + lineHeight}
		if y >= bounds.Top && area.Bottom <= bounds.Bottom {
			color := baseColor
			if isRef {
				color = theme.RGB{R: 239, G: 205, B: 126}
			}
			label := area
			label.Right = min(area.Right, x+panelTextWidth(hdc, text, fontSize)+1)
			drawPanelTextColor(hdc, pixels, width, height, label, text, fontSize, theme.UIFontFamily, color)
			if isRef {
				if len(ref.Schools) > 0 {
					left := label.Right + 3
					parenWidth := panelTextWidth(hdc, "(", fontSize)
					p := area
					p.Left = left
					p.Right = left + parenWidth + 1
					drawPanelLine(hdc, pixels, width, height, p, "(", fontSize)
					left += parenWidth + 2
					for _, school := range ref.Schools {
						drawSchoolIcon(pixels, width, height, school, left+iconSize/2, y+(lineHeight-iconSize)/2, iconSize)
						left += iconSize + 3
					}
					p.Left = left
					p.Right = left + parenWidth + 1
					drawPanelLine(hdc, pixels, width, height, p, ")", fontSize)
				}
				links = append(links, strategyEnemyLink{Bounds: area, Family: ref.FamilyID, Mode: ref.Mode, Head: ref.HeadID})
			}
		}
		x += wordWidth
	}
	return y + lineHeight, links
}

// Caller holds w.mu.
func (w *Window) strategyLinkAtLocked(p widget.Point) *strategyEnemyLink {
	if !w.panelOpen || w.panelLoading || w.worldMenuOpen || !w.detailCheats || w.strategyLinksEnemy != w.selectedEnemy {
		return nil
	}
	for i := range w.strategyLinks {
		if w.strategyLinks[i].Bounds.Contains(p) {
			return &w.strategyLinks[i]
		}
	}
	return nil
}
