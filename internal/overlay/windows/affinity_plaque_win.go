//go:build windows

package windows

import (
	"corvin101/internal/overlay/widget"
	"corvin101/internal/overlay/widgets"
	"unsafe"
)

type affinityGroupLayout struct {
	bounds  widget.Rect
	columns int
}
type affinityPlaqueLayout struct {
	groups                                             []affinityGroupLayout
	width, height, padding, gap, iconSize, valueHeight int32
}

func layoutAffinityPlaque(groups []widgets.EnemyAffinityGroup, scale float64) affinityPlaqueLayout {
	l := affinityPlaqueLayout{padding: max(5, int32(7*scale)), gap: max(2, int32(4*scale)), iconSize: max(18, int32(28*scale)), valueHeight: max(16, int32(22*scale))}
	var widths [2]int32
	for i, g := range groups {
		columns := max(1, min(3, len(g.Schools)))
		w := max(int32(45*scale), int32(columns)*l.iconSize+int32(columns-1)*l.gap)
		widths[i%2] = max(widths[i%2], w)
	}
	top := l.padding
	for i := 0; i < len(groups); i += 2 {
		rowHeight := int32(0)
		for j := i; j < min(i+2, len(groups)); j++ {
			columns := max(1, min(3, len(groups[j].Schools)))
			rows := max(1, (len(groups[j].Schools)+columns-1)/columns)
			h := l.valueHeight + int32(rows)*l.iconSize + int32(rows-1)*l.gap
			left := l.padding
			if j%2 == 1 {
				left += widths[0] + l.gap*2
			}
			l.groups = append(l.groups, affinityGroupLayout{bounds: widget.Rect{Left: left, Top: top, Right: left + widths[j%2], Bottom: top + h}, columns: columns})
			rowHeight = max(rowHeight, h)
		}
		top += rowHeight + l.gap
	}
	l.width = l.padding*2 + widths[0]
	if len(groups) > 1 {
		l.width += l.gap*2 + widths[1]
	}
	l.height = top - l.gap + l.padding
	return l
}

func enemyAffinitySlotBounds(width, height int32, slot int, groups []widgets.EnemyAffinityGroup, scale float64) widget.Rect {
	if slot < 0 || slot >= 4 || len(groups) == 0 {
		return widget.Rect{}
	}
	layout := layoutAffinityPlaque(groups, scale)
	left := int32(float64(width)*(0.156+float64(slot)*0.223) + 8*scale)
	left = max(0, min(left, width-layout.width-4))
	top := int32(float64(height) * 0.018)
	return widget.Rect{Left: left, Top: top, Right: left + layout.width, Bottom: top + layout.height}
}

func drawEnemyAffinityGroups(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, groups []widgets.EnemyAffinityGroup, scale float64) {
	if len(groups) == 0 {
		return
	}
	layout := layoutAffinityPlaque(groups, scale)
	drawAffinityParchment(pixels, width, height, bounds, scale)
	for i, group := range groups {
		l := layout.groups[i]
		b := l.bounds
		b.Left += bounds.Left
		b.Right += bounds.Left
		b.Top += bounds.Top
		b.Bottom += bounds.Top
		value := b
		value.Bottom = value.Top + layout.valueHeight
		drawAffinityValue(hdc, pixels, width, height, value, group.Value, scale)
		for index, school := range group.Schools {
			row, col := index/l.columns, index%l.columns
			count := min(l.columns, len(group.Schools)-row*l.columns)
			contentWidth := int32(count)*layout.iconSize + int32(count-1)*layout.gap
			left := b.Left + (b.Width()-contentWidth)/2 + int32(col)*(layout.iconSize+layout.gap)
			top := value.Bottom + int32(row)*(layout.iconSize+layout.gap)
			drawSchoolIcon(pixels, width, height, school, left+layout.iconSize/2, top, layout.iconSize)
		}
	}
}
