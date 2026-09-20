//go:build windows

package windows

import (
	"context"
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"corvin101/internal/game/state"
	"corvin101/internal/overlay/widgets"
)

const (
	overlayClassName = "CorvinOverlayWindow"
	overlayTitle     = "Corvin Overlay"

	wmDestroy = 0x0002

	wsPopup = 0x80000000

	wsExTopmost     = 0x00000008
	wsExTransparent = 0x00000020
	wsExToolWindow  = 0x00000080
	wsExLayered     = 0x00080000
	wsExNoActivate  = 0x08000000

	swHide           = 0
	swShowNoActivate = 4

	swpNoActivate = 0x0010
	swpShowWindow = 0x0040

	ulwAlpha = 0x00000002
)

var (
	user32 = windows.NewLazySystemDLL("user32.dll")
	gdi32  = windows.NewLazySystemDLL("gdi32.dll")

	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procShowWindow       = user32.NewProc("ShowWindow")
	procSetWindowPos     = user32.NewProc("SetWindowPos")

	procUpdateLayeredWindow = user32.NewProc("UpdateLayeredWindow")

	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procSetBkMode          = gdi32.NewProc("SetBkMode")
	procSetTextColor       = gdi32.NewProc("SetTextColor")
	procCreateFontW        = gdi32.NewProc("CreateFontW")

	procDrawTextW = user32.NewProc("DrawTextW")

	procEnumWindows = user32.NewProc("EnumWindows")

	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")

	procIsWindowVisible = user32.NewProc("IsWindowVisible")
	procGetClientRect   = user32.NewProc("GetClientRect")
	procClientToScreen  = user32.NewProc("ClientToScreen")

	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
)

type Window struct {
	mu sync.RWMutex

	hwnd windows.Handle

	width  int32
	height int32

	combat widgets.Combat
}

type point struct {
	X int32
	Y int32
}

type size struct {
	CX int32
	CY int32
}

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

type msg struct {
	Hwnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type blendFunction struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]uint32
}

var activeWindow *Window

func New() *Window {
	return &Window{}
}

func (w *Window) Update(snapshot state.Snapshot) {
	w.mu.Lock()
	w.combat.Update(snapshot)
	w.mu.Unlock()

	w.render()
}

func (w *Window) Run(ctx context.Context) error {
	activeWindow = w

	instance, _, _ := procGetModuleHandleW.Call(0)

	className, err := windows.UTF16PtrFromString(
		overlayClassName,
	)
	if err != nil {
		return err
	}

	class := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   syscall.NewCallback(windowProc),
		Instance:  windows.Handle(instance),
		ClassName: className,
	}

	result, _, registerErr :=
		procRegisterClassExW.Call(
			uintptr(unsafe.Pointer(&class)),
		)

	if result == 0 {
		return fmt.Errorf(
			"register overlay window class: %v",
			registerErr,
		)
	}

	title, err := windows.UTF16PtrFromString(
		overlayTitle,
	)
	if err != nil {
		return err
	}

	hwnd, _, createErr :=
		procCreateWindowExW.Call(
			wsExTopmost|
				wsExTransparent|
				wsExToolWindow|
				wsExLayered|
				wsExNoActivate,
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(title)),
			wsPopup,
			0,
			0,
			1,
			1,
			0,
			0,
			instance,
			0,
		)

	if hwnd == 0 {
		return fmt.Errorf(
			"create overlay window: %v",
			createErr,
		)
	}

	w.mu.Lock()
	w.hwnd = windows.Handle(hwnd)
	w.mu.Unlock()

	procShowWindow.Call(
		hwnd,
		swShowNoActivate,
	)

	go w.followGameWindow(ctx)

	go func() {
		<-ctx.Done()

		w.mu.RLock()
		hwnd := w.hwnd
		w.mu.RUnlock()

		if hwnd != 0 {
			procDestroyWindow.Call(
				uintptr(hwnd),
			)
		}
	}()

	var message msg

	for {
		result, _, _ :=
			procGetMessageW.Call(
				uintptr(unsafe.Pointer(&message)),
				0,
				0,
				0,
			)

		if int32(result) <= 0 {
			break
		}

		procTranslateMessage.Call(
			uintptr(unsafe.Pointer(&message)),
		)

		procDispatchMessageW.Call(
			uintptr(unsafe.Pointer(&message)),
		)
	}

	w.mu.Lock()
	w.hwnd = 0
	w.mu.Unlock()

	return nil
}

func windowProc(
	hwnd uintptr,
	message uint32,
	wParam uintptr,
	lParam uintptr,
) uintptr {
	switch message {
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}

	result, _, _ :=
		procDefWindowProcW.Call(
			hwnd,
			uintptr(message),
			wParam,
			lParam,
		)

	return result
}
