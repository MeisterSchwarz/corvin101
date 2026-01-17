package tray

import (
	_ "embed"
	"log"
	"os"
	"path/filepath"

	"wizard101rpc/config"
	"wizard101rpc/filesystem"
	"wizard101rpc/logreader"
	"wizard101rpc/system"
	"wizard101rpc/zones"

	"github.com/getlantern/systray"
	"github.com/sqweek/dialog"
)

//go:embed wizard101rpc.ico
var iconData []byte

// starts system tray
func Run() {
	systray.Run(onReady, onExit)
}

// initializes tray UI and handlers
func onReady() {
	systray.SetIcon(iconData)
	systray.SetTitle("Wizard101 RPC")
	systray.SetTooltip("Wizard101 Discord Rich Presence")

	var mSelectLog *systray.MenuItem
	if _, ok := logreader.ResolveLogPath(); !ok {
		systray.AddSeparator()
		mSelectLog = systray.AddMenuItem(
			"Pfad manuell auswählen",
			"Automatische Pfadsuche fehlgeschlagen",
		)
	}

	mContribute := systray.AddMenuItemCheckbox(
		"Fehlende Übersetzungen sammeln",
		"Unbekannte Zonen loggen",
		false,
	)
	systray.AddSeparator()
	mAutostart := systray.AddMenuItem("Autostart aktivieren", "")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Beenden", "")

	updateAutostartTitle(mAutostart)

	// Event loop
	go func() {
		for {
			select {

			case <-clicked(mSelectLog):
				path, err := dialog.File().
					Title("Datei im Wizard101-Ordner auswählen").
					Filter("Wizard101 Dateien", "ico").
					Load()
				if err != nil {
					continue
				}

				dir := filepath.Dir(path)

				candidate := filepath.Join(dir, "Bin", "WizardClient.log")
				if !filesystem.FileExists(candidate) {
					log.Println("[tray] kein WizardClient.log im ausgewählten Ordner")
					continue
				}

				config.AppConfig.InstallDir = dir
				if err := config.Save(); err != nil {
					log.Println("[tray] failed to save config:", err)
					continue
				}
				mSelectLog.SetTitle("Log-Pfad gesetzt")
				mSelectLog.Disable()

				system.Restart()

			case <-mAutostart.ClickedCh:
				toggleAutostart(mAutostart)
			case <-mContribute.ClickedCh:
				enabled := !mContribute.Checked()
				zones.SetContributing(enabled)

				if enabled {
					mContribute.Check()
				} else {
					mContribute.Uncheck()
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				os.Exit(0)
			}
		}
	}()
}

// helper: nil-safe wrapper for mSelectLog
func clicked(item *systray.MenuItem) <-chan struct{} {
	if item == nil {
		return nil
	}
	return item.ClickedCh
}

// exit function
func onExit() {
	log.Println("Tray exited")
}

// toggle autostart
func toggleAutostart(item *systray.MenuItem) {
	var err error

	if system.IsAutostartEnabled() {
		err = system.DisableAutostart()
	} else {
		err = system.EnableAutostart()
	}

	if err != nil {
		log.Println("Autostart toggle failed:", err)
		return
	}

	updateAutostartTitle(item)
}

// updates menu label based on state
func updateAutostartTitle(item *systray.MenuItem) {
	if system.IsAutostartEnabled() {
		item.SetTitle("Autostart deaktivieren")
	} else {
		item.SetTitle("Autostart aktivieren")
	}
}
