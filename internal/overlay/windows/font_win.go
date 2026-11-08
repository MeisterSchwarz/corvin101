//go:build windows

package windows

import (
	"log"
	"sync"
	"unsafe"

	"corvin101/assets"
)

var (
	fontLoadOnce      sync.Once
	overlayFontHandle uintptr
)

func loadOverlayFonts() {
	fontLoadOnce.Do(func() {
		var count uint32
		handle, _, err := gdi32.NewProc("AddFontMemResourceEx").Call(
			uintptr(unsafe.Pointer(&assets.DisplayFont[0])), uintptr(len(assets.DisplayFont)), 0, uintptr(unsafe.Pointer(&count)),
		)
		if handle == 0 || count == 0 {
			log.Printf("[overlay] load embedded font: %v", err)
			return
		}
		// Keep the private font registered for the process lifetime.
		overlayFontHandle = handle
	})
}
