package tray

import (
	_ "embed"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"runtime"

	"wizlink/config"
	"wizlink/filesystem"
	"wizlink/logreader"
	"wizlink/system"

	"github.com/getlantern/systray"
	"github.com/sqweek/dialog"
)

const webUIURL = "http://127.0.0.1:8101"

//go:embed wizlink.ico
var iconData []byte

// Run starts the system tray application.
func Run() {
	systray.Run(onReady, onExit)
}

// onReady initializes the tray menu and its event handlers.
func onReady() {
	systray.SetIcon(iconData)
	systray.SetTitle("Wizard101 RPC")
	systray.SetTooltip("Wizard101 Discord Rich Presence")

	selectLogItem := addSelectLogItem()

	webUIItem := systray.AddMenuItem(
		"Web-UI öffnen",
		webUIURL,
	)

	systray.AddSeparator()

	autostartItem := systray.AddMenuItem(
		"Autostart aktivieren",
		"",
	)
	updateAutostartTitle(autostartItem)

	systray.AddSeparator()

	quitItem := systray.AddMenuItem("Beenden", "")

	go handleMenuEvents(
		selectLogItem,
		webUIItem,
		autostartItem,
		quitItem,
	)
}

// addSelectLogItem adds the manual log-path menu item when automatic
// path detection fails.
func addSelectLogItem() *systray.MenuItem {
	if _, ok := logreader.ResolveLogPath(); ok {
		return nil
	}

	return systray.AddMenuItem(
		"Pfad manuell auswählen",
		"Automatische Pfadsuche fehlgeschlagen",
	)
}

// handleMenuEvents processes all tray menu interactions.
func handleMenuEvents(
	selectLogItem *systray.MenuItem,
	webUIItem *systray.MenuItem,
	autostartItem *systray.MenuItem,
	quitItem *systray.MenuItem,
) {
	for {
		select {
		case <-clicked(selectLogItem):
			handleLogPathSelection(selectLogItem)

		case <-webUIItem.ClickedCh:
			if err := openURL(webUIURL); err != nil {
				log.Printf("[tray] Web-UI konnte nicht geöffnet werden: %v", err)
			}

		case <-autostartItem.ClickedCh:
			toggleAutostart(autostartItem)

		case <-quitItem.ClickedCh:
			systray.Quit()
			return
		}
	}
}

// handleLogPathSelection lets the user select the Wizard101 installation
// directory and stores the discovered log path.
func handleLogPathSelection(item *systray.MenuItem) {
	path, err := dialog.File().
		Title("Datei im Wizard101-Ordner auswählen").
		Filter("Wizard101 Dateien", "ico").
		Load()
	if err != nil {
		return
	}

	installDir := filepath.Dir(path)
	logPath := filepath.Join(
		installDir,
		"Bin",
		"WizardClient.log",
	)

	if !filesystem.FileExists(logPath) {
		log.Printf(
			"[tray] keine WizardClient.log im ausgewählten Ordner: %s",
			installDir,
		)
		return
	}

	config.AppConfig.InstallDir = installDir

	if err := config.Save(); err != nil {
		log.Printf("[tray] Konfiguration konnte nicht gespeichert werden: %v", err)
		return
	}

	item.SetTitle("Log-Pfad gesetzt")
	item.Disable()

	system.Restart()
}

// clicked returns the click channel of a menu item.
// A nil item produces a permanently blocked channel.
func clicked(item *systray.MenuItem) <-chan struct{} {
	if item == nil {
		return nil
	}

	return item.ClickedCh
}

// openURL opens a URL in the operating system's default browser.
func openURL(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)

	case "darwin":
		cmd = exec.Command("open", url)

	case "linux":
		cmd = exec.Command("xdg-open", url)

	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// toggleAutostart enables or disables application autostart.
func toggleAutostart(item *systray.MenuItem) {
	var err error

	if system.IsAutostartEnabled() {
		err = system.DisableAutostart()
	} else {
		err = system.EnableAutostart()
	}

	if err != nil {
		log.Printf("[tray] Autostart konnte nicht geändert werden: %v", err)
		return
	}

	updateAutostartTitle(item)
}

// updateAutostartTitle updates the menu label to reflect the current state.
func updateAutostartTitle(item *systray.MenuItem) {
	if system.IsAutostartEnabled() {
		item.SetTitle("Autostart deaktivieren")
		return
	}

	item.SetTitle("Autostart aktivieren")
}

// onExit is called after the system tray has stopped.
func onExit() {
	log.Println("[tray] beendet")
}
