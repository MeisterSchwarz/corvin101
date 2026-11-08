//go:build windows

package windows

import (
	"corvin101/internal/data/enemies"
	"corvin101/internal/overlay/widget"
	"fmt"
	"sort"
	"unsafe"
)

func drawDamageColumns(hdc uintptr, pixels unsafe.Pointer, width, height int32, bounds widget.Rect, top, fontSize int32, affinities map[enemies.School]int) int32 {
	var columns [2][]enemies.School
	for school, value := range affinities {
		if value > 0 {
			columns[0] = append(columns[0], school)
		} else {
			columns[1] = append(columns[1], school)
		}
	}
	for i := range columns {
		sort.Slice(columns[i], func(a, b int) bool { return columns[i][a] < columns[i][b] })
	}
	rowHeight := max(int32(26), fontSize+12)
	iconSize := min(rowHeight-4, fontSize+6)
	columnWidth := min(bounds.Width()/2, fontSize*6)
	rows := max(len(columns[0]), len(columns[1]))
	if rows > 0 {
		x := bounds.Left + columnWidth - 8
		fillPanelRect(pixels, width, height, widget.Rect{Left: x, Top: top, Right: x + 1, Bottom: min(bounds.Bottom, top+int32(rows)*rowHeight)}, 0xff6f563a)
	}
	if rows == 0 {
		line := bounds
		line.Top = top
		line.Bottom = top + rowHeight
		drawPanelLine(hdc, pixels, width, height, line, "Keine Schadensmodifikatoren hinterlegt", fontSize)
		rows = 1
	}
	for column, schools := range columns {
		left := bounds.Left + int32(column)*columnWidth
		for i, school := range schools {
			y := top + int32(i)*rowHeight
			if y+rowHeight > bounds.Bottom {
				break
			}
			valueBounds := widget.Rect{Left: left + iconSize + 6, Top: y, Right: left + columnWidth - 4, Bottom: y + rowHeight}
			value := fmt.Sprintf("(%+d%%)", affinities[school])
			if affinities[school] == 0 {
				value = "(0%)"
			}
			drawPanelLine(hdc, pixels, width, height, valueBounds, value, fontSize)
			drawSchoolIcon(pixels, width, height, school, left+iconSize/2, y+(rowHeight-iconSize)/2, iconSize)
		}
	}
	return top + int32(rows)*rowHeight + fontSize
}
