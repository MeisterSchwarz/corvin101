//go:build windows

package system

import "testing"

func TestEmbeddedTrayIconWithoutPublicDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	tray := &Tray{}
	if err := tray.loadIcon(); err != nil {
		t.Fatal(err)
	}
	if tray.icon == 0 {
		t.Fatal("embedded icon was not created")
	}
	defer trayUser32.NewProc("DestroyIcon").Call(uintptr(tray.icon))
}

func TestTrayIconRejectsMalformedData(t *testing.T) {
	for _, data := range [][]byte{nil, {0, 0, 1, 0, 1, 0}, {0, 0, 1, 0, 1, 0, 32, 32, 0, 0, 1, 0, 32, 0, 255, 255, 255, 255, 255, 255, 255, 255}} {
		if _, err := trayIconImage(data); err == nil {
			t.Fatal("malformed ICO accepted")
		}
	}
}
