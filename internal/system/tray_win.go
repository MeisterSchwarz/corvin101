//go:build windows

package system

import (
	"context"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wmDestroy = 0x0002
	wmCommand = 0x0111
	wmUser    = 0x0400

	wmTrayIcon = wmUser + 1

	nimAdd    = 0x00000000
	nimDelete = 0x00000002

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	wmRButtonUp = 0x0205

	mfString    = 0x00000000
	tpmRightBtn = 0x0002

	trayExitID = 1001

	imageIcon      = 1
	lrLoadFromFile = 0x0010
	lrDefaultSize  = 0x0040
)

var (
	trayUser32 = windows.NewLazySystemDLL("user32.dll")

	trayShell32 = windows.NewLazySystemDLL("shell32.dll")

	trayKernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procTrayRegisterClassExW = trayUser32.NewProc("RegisterClassExW")

	procTrayCreateWindowExW = trayUser32.NewProc("CreateWindowExW")

	procTrayDefWindowProcW = trayUser32.NewProc("DefWindowProcW")

	procTrayGetMessageW = trayUser32.NewProc("GetMessageW")

	procTrayTranslateMessage = trayUser32.NewProc("TranslateMessage")

	procTrayDispatchMessageW = trayUser32.NewProc("DispatchMessageW")

	procTrayDestroyWindow = trayUser32.NewProc("DestroyWindow")

	procTrayPostQuitMessage = trayUser32.NewProc("PostQuitMessage")

	procTrayCreatePopupMenu = trayUser32.NewProc("CreatePopupMenu")

	procTrayAppendMenuW = trayUser32.NewProc("AppendMenuW")

	procTrayTrackPopupMenu = trayUser32.NewProc("TrackPopupMenu")

	procTrayDestroyMenu = trayUser32.NewProc("DestroyMenu")

	procTrayGetCursorPos = trayUser32.NewProc("GetCursorPos")

	procTraySetForegroundWindow = trayUser32.NewProc("SetForegroundWindow")

	procTrayLoadImageW = trayUser32.NewProc("LoadImageW")

	procTrayShellNotifyIconW = trayShell32.NewProc("Shell_NotifyIconW")

	procTrayGetModuleHandleW = trayKernel32.NewProc("GetModuleHandleW")
)

type trayPoint struct {
	X int32
	Y int32
}

type trayMessage struct {
	Hwnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      trayPoint
}

type trayWndClassEx struct {
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

type notifyIconData struct {
	Size             uint32
	Hwnd             windows.Handle
	ID               uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             windows.Handle
	Tip              [128]uint16
	State            uint32
	StateMask        uint32
	Info             [256]uint16
	TimeoutOrVersion uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	GuidItem         windows.GUID
	BalloonIcon      windows.Handle
}

type Tray struct {
	hwnd windows.Handle
	icon windows.Handle

	onExit func()
}

var activeTray *Tray

func NewTray(
	onExit func(),
) *Tray {
	return &Tray{
		onExit: onExit,
	}
}

func (t *Tray) Run(
	ctx context.Context,
	iconPath string,
) error {
	activeTray = t

	instance, _, _ :=
		procTrayGetModuleHandleW.Call(0)

	className, err :=
		windows.UTF16PtrFromString(
			"CorvinTrayWindow",
		)

	if err != nil {
		return err
	}

	class := trayWndClassEx{
		Size: uint32(
			unsafe.Sizeof(
				trayWndClassEx{},
			),
		),
		WndProc: syscall.NewCallback(
			trayWindowProc,
		),
		Instance: windows.Handle(
			instance,
		),
		ClassName: className,
	}

	result, _, registerErr :=
		procTrayRegisterClassExW.Call(
			uintptr(
				unsafe.Pointer(
					&class,
				),
			),
		)

	if result == 0 {
		return fmt.Errorf(
			"register tray window class: %v",
			registerErr,
		)
	}

	windowName, err :=
		windows.UTF16PtrFromString(
			"Corvin Tray",
		)

	if err != nil {
		return err
	}

	hwnd, _, createErr :=
		procTrayCreateWindowExW.Call(
			0,
			uintptr(
				unsafe.Pointer(
					className,
				),
			),
			uintptr(
				unsafe.Pointer(
					windowName,
				),
			),
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			instance,
			0,
		)

	if hwnd == 0 {
		return fmt.Errorf(
			"create tray window: %v",
			createErr,
		)
	}

	t.hwnd =
		windows.Handle(hwnd)

	if err := t.loadIcon(
		iconPath,
	); err != nil {
		procTrayDestroyWindow.Call(hwnd)
		return err
	}

	if err := t.addIcon(); err != nil {
		procTrayDestroyWindow.Call(hwnd)
		return err
	}

	go func() {
		<-ctx.Done()

		if t.hwnd != 0 {
			procTrayDestroyWindow.Call(
				uintptr(t.hwnd),
			)
		}
	}()

	var message trayMessage

	for {
		result, _, _ :=
			procTrayGetMessageW.Call(
				uintptr(
					unsafe.Pointer(
						&message,
					),
				),
				0,
				0,
				0,
			)

		if int32(result) <= 0 {
			break
		}

		procTrayTranslateMessage.Call(
			uintptr(
				unsafe.Pointer(
					&message,
				),
			),
		)

		procTrayDispatchMessageW.Call(
			uintptr(
				unsafe.Pointer(
					&message,
				),
			),
		)
	}

	t.removeIcon()

	return nil
}

func (t *Tray) loadIcon(
	path string,
) error {
	iconPath, err :=
		windows.UTF16PtrFromString(
			path,
		)

	if err != nil {
		return err
	}

	icon, _, loadErr :=
		procTrayLoadImageW.Call(
			0,
			uintptr(
				unsafe.Pointer(
					iconPath,
				),
			),
			imageIcon,
			0,
			0,
			lrLoadFromFile|
				lrDefaultSize,
		)

	if icon == 0 {
		return fmt.Errorf(
			"load tray icon %q: %v",
			path,
			loadErr,
		)
	}

	t.icon =
		windows.Handle(icon)

	return nil
}

func (t *Tray) addIcon() error {
	data := notifyIconData{
		Size: uint32(
			unsafe.Sizeof(
				notifyIconData{},
			),
		),
		Hwnd: t.hwnd,
		ID:   1,
		Flags: nifMessage |
			nifIcon |
			nifTip,
		CallbackMessage: wmTrayIcon,
		Icon:            t.icon,
	}

	copy(
		data.Tip[:],
		windows.StringToUTF16(
			"Corvin",
		),
	)

	result, _, callErr :=
		procTrayShellNotifyIconW.Call(
			nimAdd,
			uintptr(
				unsafe.Pointer(
					&data,
				),
			),
		)

	if result == 0 {
		return fmt.Errorf(
			"add tray icon: %v",
			callErr,
		)
	}

	return nil
}

func (t *Tray) removeIcon() {
	if t.hwnd == 0 {
		return
	}

	data := notifyIconData{
		Size: uint32(
			unsafe.Sizeof(
				notifyIconData{},
			),
		),
		Hwnd: t.hwnd,
		ID:   1,
	}

	procTrayShellNotifyIconW.Call(
		nimDelete,
		uintptr(
			unsafe.Pointer(
				&data,
			),
		),
	)
}

func trayWindowProc(
	hwnd uintptr,
	message uint32,
	wParam uintptr,
	lParam uintptr,
) uintptr {
	switch message {
	case wmTrayIcon:
		if uint32(lParam) ==
			wmRButtonUp {

			showTrayMenu(
				windows.Handle(
					hwnd,
				),
			)
		}

		return 0

	case wmCommand:
		command :=
			uint32(
				wParam & 0xffff,
			)

		if command == trayExitID {
			if activeTray != nil &&
				activeTray.onExit != nil {

				activeTray.onExit()
			}
		}

		return 0

	case wmDestroy:
		procTrayPostQuitMessage.Call(0)
		return 0
	}

	result, _, _ :=
		procTrayDefWindowProcW.Call(
			hwnd,
			uintptr(message),
			wParam,
			lParam,
		)

	return result
}

func showTrayMenu(
	hwnd windows.Handle,
) {
	menu, _, _ :=
		procTrayCreatePopupMenu.Call()

	if menu == 0 {
		return
	}

	defer procTrayDestroyMenu.Call(
		menu,
	)

	exitText, err :=
		windows.UTF16PtrFromString(
			"Corvin beenden",
		)

	if err != nil {
		return
	}

	procTrayAppendMenuW.Call(
		menu,
		mfString,
		trayExitID,
		uintptr(
			unsafe.Pointer(
				exitText,
			),
		),
	)

	var cursor trayPoint

	procTrayGetCursorPos.Call(
		uintptr(
			unsafe.Pointer(
				&cursor,
			),
		),
	)

	// Wichtig für korrektes Verhalten des Popup-Menüs.
	procTraySetForegroundWindow.Call(
		uintptr(hwnd),
	)

	procTrayTrackPopupMenu.Call(
		menu,
		tpmRightBtn,
		uintptr(cursor.X),
		uintptr(cursor.Y),
		0,
		uintptr(hwnd),
		0,
	)
}
