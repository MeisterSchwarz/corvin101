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
	"corvin101/internal/overlay/widget"
	"corvin101/internal/overlay/widgets"
)

const (
	overlayClassName = "CorvinOverlayWindow"
	overlayTitle     = "Corvin Overlay"

	wmDestroy = 0x0002

	wsPopup = 0x80000000

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
	user32 = windows.NewLazySystemDLL(
		"user32.dll",
	)

	gdi32 = windows.NewLazySystemDLL(
		"gdi32.dll",
	)

	kernel32 = windows.NewLazySystemDLL(
		"kernel32.dll",
	)

	// ------------------------------------------------------------
	// Window creation / lifecycle
	// ------------------------------------------------------------

	procRegisterClassExW = user32.NewProc(
		"RegisterClassExW",
	)

	procCreateWindowExW = user32.NewProc(
		"CreateWindowExW",
	)

	procDefWindowProcW = user32.NewProc(
		"DefWindowProcW",
	)

	procDestroyWindow = user32.NewProc(
		"DestroyWindow",
	)

	procDispatchMessageW = user32.NewProc(
		"DispatchMessageW",
	)

	procGetMessageW = user32.NewProc(
		"GetMessageW",
	)

	procTranslateMessage = user32.NewProc(
		"TranslateMessage",
	)

	procPostQuitMessage = user32.NewProc(
		"PostQuitMessage",
	)

	procShowWindow = user32.NewProc(
		"ShowWindow",
	)

	procSetWindowPos = user32.NewProc(
		"SetWindowPos",
	)

	// ------------------------------------------------------------
	// Cursor / coordinates
	// ------------------------------------------------------------

	procGetCursorPos = user32.NewProc(
		"GetCursorPos",
	)

	procScreenToClient = user32.NewProc(
		"ScreenToClient",
	)

	// ------------------------------------------------------------
	// Layered window rendering
	// ------------------------------------------------------------

	procUpdateLayeredWindow = user32.NewProc(
		"UpdateLayeredWindow",
	)

	// ------------------------------------------------------------
	// GDI
	// ------------------------------------------------------------

	procCreateCompatibleDC = gdi32.NewProc(
		"CreateCompatibleDC",
	)

	procDeleteDC = gdi32.NewProc(
		"DeleteDC",
	)

	procCreateDIBSection = gdi32.NewProc(
		"CreateDIBSection",
	)

	procSelectObject = gdi32.NewProc(
		"SelectObject",
	)

	procDeleteObject = gdi32.NewProc(
		"DeleteObject",
	)

	procSetBkMode = gdi32.NewProc(
		"SetBkMode",
	)

	procSetTextColor = gdi32.NewProc(
		"SetTextColor",
	)

	procCreateFontW = gdi32.NewProc(
		"CreateFontW",
	)

	procDrawTextW = user32.NewProc(
		"DrawTextW",
	)

	// ------------------------------------------------------------
	// Wizard101 window discovery
	// ------------------------------------------------------------

	procEnumWindows = user32.NewProc(
		"EnumWindows",
	)

	procGetWindowThreadProcessId = user32.NewProc(
		"GetWindowThreadProcessId",
	)

	procIsWindowVisible = user32.NewProc(
		"IsWindowVisible",
	)

	procGetClientRect = user32.NewProc(
		"GetClientRect",
	)

	procClientToScreen = user32.NewProc(
		"ClientToScreen",
	)

	// ------------------------------------------------------------
	// Kernel
	// ------------------------------------------------------------

	procGetModuleHandleW = kernel32.NewProc(
		"GetModuleHandleW",
	)
)

// Window ist die Windows-spezifische Implementierung
// des Corvin-Overlays.
//
// Das eigentliche Widget-System kennt keine Win32-Details.
// Window verbindet lediglich:
//
//   - WidgetManager
//   - Win32 Window
//   - Cursor
//   - Rendering
//   - Wizard101 Window Tracking
type Window struct {
	mu sync.RWMutex

	hwnd windows.Handle

	width  int32
	height int32

	// Zentrale Registry aller Overlay-Widgets.
	manager *widget.Manager

	// Referenz auf das Round/Round-Widget.
	//
	// Der Manager besitzt die eigentliche Widget-Liste.
	// Die Referenz ist praktisch, falls Windows-spezifischer
	// Code später gezielt auf dieses Widget zugreifen muss.
	round *widgets.Round

	// Cursorposition relativ zum Overlay bzw.
	// Wizard101-Client.
	//
	// Sie wird in input_win.go über GetCursorPos +
	// ScreenToClient ermittelt.
	cursor widget.Point

	cursorKnown bool
}

// point entspricht Win32 POINT.
type point struct {
	X int32
	Y int32
}

// size entspricht Win32 SIZE.
type size struct {
	CX int32
	CY int32
}

// rect entspricht Win32 RECT.
type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

// blendFunction entspricht Win32 BLENDFUNCTION.
type blendFunction struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

// bitmapInfoHeader entspricht BITMAPINFOHEADER.
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

// bitmapInfo entspricht der für CreateDIBSection
// benötigten BITMAPINFO-Struktur.
type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]uint32
}

// wndClassEx entspricht Win32 WNDCLASSEXW.
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

// msg entspricht Win32 MSG.
type msg struct {
	Hwnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

// activeWindow wird momentan vom globalen Win32
// WindowProc verwendet.
//
// Für ein einzelnes Overlay-Fenster ist das ausreichend.
// Sollten wir irgendwann mehrere echte HWNDs erzeugen,
// würden wir das durch HWND -> *Window Mapping ersetzen.
var activeWindow *Window

// New erstellt die Windows-Overlay-Instanz und registriert
// die initialen Widgets.
//
// Neue Widgets werden perspektivisch einfach hier bzw.
// über eine eigene Widget-Registry dem Manager hinzugefügt.
func New() *Window {
	round :=
		widgets.NewRound()

	manager :=
		widget.NewManager(
			round,
		)

	return &Window{
		manager: manager,
		round:   round,
	}
}

// Update reicht den aktuellen Game-State an alle Widgets
// weiter und rendert anschließend einen neuen Frame.
func (w *Window) Update(
	snapshot state.Snapshot,
) {
	w.mu.Lock()

	if w.manager != nil {
		w.manager.Update(
			snapshot,
		)
	}

	w.mu.Unlock()

	w.render()
}

// Run erstellt das transparente Win32-Overlay und startet
// die Windows Message Loop.
func (w *Window) Run(
	ctx context.Context,
) error {
	loadOverlayFonts()
	activeWindow = w

	instance, _, _ :=
		procGetModuleHandleW.Call(0)

	className, err :=
		windows.UTF16PtrFromString(
			overlayClassName,
		)

	if err != nil {
		return err
	}

	class := wndClassEx{
		Size: uint32(
			unsafe.Sizeof(
				wndClassEx{},
			),
		),

		WndProc: syscall.NewCallback(
			windowProc,
		),

		Instance: windows.Handle(
			instance,
		),

		ClassName: className,
	}

	result, _, registerErr :=
		procRegisterClassExW.Call(
			uintptr(
				unsafe.Pointer(
					&class,
				),
			),
		)

	if result == 0 {
		return fmt.Errorf(
			"register overlay window class: %v",
			registerErr,
		)
	}

	title, err :=
		windows.UTF16PtrFromString(
			overlayTitle,
		)

	if err != nil {
		return err
	}

	// Wichtig:
	//
	// WS_EX_TRANSPARENT
	//     Das Overlay soll Mausinteraktion nicht blockieren.
	//
	// WS_EX_LAYERED
	//     Erlaubt per-pixel Alpha über UpdateLayeredWindow.
	//
	// WS_EX_NOACTIVATE
	//     Das Overlay bekommt keinen Fokus.
	//
	// WS_EX_TOOLWINDOW
	//     Kein eigener Taskbar-Eintrag.
	//
	// WS_EX_TOPMOST
	//     Bleibt für diesen Refactor zunächst bestehen.
	//     Die Z-Order relativ zu Wizard101 bauen wir separat
	//     im nächsten Schritt um.
	hwnd, _, createErr :=
		procCreateWindowExW.Call(
			wsExTransparent|
				wsExToolWindow|
				wsExLayered|
				wsExNoActivate,

			uintptr(
				unsafe.Pointer(
					className,
				),
			),

			uintptr(
				unsafe.Pointer(
					title,
				),
			),

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

	w.hwnd =
		windows.Handle(
			hwnd,
		)

	w.mu.Unlock()

	// Fenster anzeigen, ohne Wizard101 den Fokus
	// wegzunehmen.
	procShowWindow.Call(
		hwnd,
		swShowNoActivate,
	)

	// Overlay an Position und Größe des Wizard101-
	// Clientbereichs koppeln.
	go w.followGameWindow(ctx)
	go w.runAnimationLoop(ctx)

	// Cursor unabhängig von Window-Mouse-Messages verfolgen.
	//
	// Dadurch können wir Hover erkennen, obwohl das Overlay
	// selbst weiterhin vollständig click-through ist.
	go w.trackCursor(
		ctx,
	)

	// Context-Abbruch zerstört das Overlay-Fenster.
	go func() {
		<-ctx.Done()

		w.mu.RLock()

		currentHWND :=
			w.hwnd

		w.mu.RUnlock()

		if currentHWND != 0 {
			procDestroyWindow.Call(
				uintptr(
					currentHWND,
				),
			)
		}
	}()

	var message msg

	for {
		result, _, getMessageErr :=
			procGetMessageW.Call(
				uintptr(
					unsafe.Pointer(
						&message,
					),
				),
				0,
				0,
				0,
			)

		// GetMessage:
		//
		// > 0 = normale Nachricht
		//   0 = WM_QUIT
		//  -1 = Fehler
		switch int32(result) {
		case 0:
			return nil

		case -1:
			return fmt.Errorf(
				"GetMessageW failed: %v",
				getMessageErr,
			)
		}

		procTranslateMessage.Call(
			uintptr(
				unsafe.Pointer(
					&message,
				),
			),
		)

		procDispatchMessageW.Call(
			uintptr(
				unsafe.Pointer(
					&message,
				),
			),
		)
	}
}

// windowProc ist die globale Win32 Window Procedure.
//
// Das Overlay benötigt momentan nur sehr wenige Messages,
// da Rendering und Positionierung aktiv von unserer Anwendung
// gesteuert werden.
func windowProc(
	hwnd uintptr,
	message uint32,
	wParam uintptr,
	lParam uintptr,
) uintptr {
	switch message {
	case wmDestroy:
		if activeWindow != nil {
			activeWindow.mu.Lock()

			if uintptr(
				activeWindow.hwnd,
			) == hwnd {

				activeWindow.hwnd = 0
			}

			activeWindow.mu.Unlock()
		}

		procPostQuitMessage.Call(
			0,
		)

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
