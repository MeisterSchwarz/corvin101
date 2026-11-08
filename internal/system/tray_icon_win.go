//go:build windows

package system

import (
	"encoding/binary"
	"fmt"
	"unsafe"

	"corvin101/public"
	"golang.org/x/sys/windows"
)

func (t *Tray) loadIcon() error {
	data, err := trayIconImage(public.TrayIcon)
	if err != nil {
		return err
	}
	icon, _, loadErr := trayUser32.NewProc("CreateIconFromResourceEx").Call(
		uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), 1, 0x00030000, 32, 32, 0,
	)
	if icon == 0 {
		return fmt.Errorf("load embedded tray icon: %v", loadErr)
	}
	t.icon = windows.Handle(icon)
	return nil
}

// ICO files contain a directory followed by individual image resources.
func trayIconImage(data []byte) ([]byte, error) {
	if len(data) < 6 || binary.LittleEndian.Uint16(data[:2]) != 0 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		return nil, fmt.Errorf("invalid embedded ICO header")
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count == 0 || 6+count*16 > len(data) {
		return nil, fmt.Errorf("invalid embedded ICO directory")
	}
	bestOffset, bestLength, bestScore := 0, 0, int(^uint(0)>>1)
	for i := 0; i < count; i++ {
		entry := data[6+i*16 : 6+(i+1)*16]
		width, height := int(entry[0]), int(entry[1])
		if width == 0 {
			width = 256
		}
		if height == 0 {
			height = 256
		}
		length, offset := uint64(binary.LittleEndian.Uint32(entry[8:12])), uint64(binary.LittleEndian.Uint32(entry[12:16]))
		if length == 0 || offset < uint64(6+count*16) || offset+length > uint64(len(data)) {
			return nil, fmt.Errorf("invalid embedded ICO image bounds")
		}
		score := (absIconSize(width-32)+absIconSize(height-32))*1000 - int(binary.LittleEndian.Uint16(entry[6:8]))
		if score < bestScore {
			bestOffset, bestLength, bestScore = int(offset), int(length), score
		}
	}
	return data[bestOffset : bestOffset+bestLength], nil
}

func absIconSize(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
