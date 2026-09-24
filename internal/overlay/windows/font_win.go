//go:build windows

package windows

import (
	"path/filepath"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	frPrivate = 0x10
)

var (
	gdi32Fonts = windows.NewLazySystemDLL(
		"gdi32.dll",
	)

	procAddFontResourceExW = gdi32Fonts.NewProc(
		"AddFontResourceExW",
	)

	fontLoadOnce sync.Once
)

func loadOverlayFonts() {
	fontLoadOnce.Do(
		func() {
			path :=
				filepath.Join(
					"assets",
					"fonts",
					"shermlock.ttf",
				)

			pathUTF16, err :=
				windows.UTF16PtrFromString(
					path,
				)

			if err != nil {
				return
			}

			procAddFontResourceExW.Call(
				uintptr(
					unsafe.Pointer(
						pathUTF16,
					),
				),
				frPrivate,
				0,
			)
		},
	)
}
