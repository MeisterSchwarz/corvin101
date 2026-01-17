//go:build windows

package system

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

const shortcutName = "Wizard101RPC.lnk"

// returns the Windows startup directory
func startupFolder() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return ""
	}

	return filepath.Join(
		appData,
		"Microsoft",
		"Windows",
		"Start Menu",
		"Programs",
		"Startup",
	)
}

// checks if the startup shortcut exists
func IsAutostartEnabled() bool {
	path := filepath.Join(startupFolder(), shortcutName)
	_, err := os.Stat(path)
	return err == nil
}

// creates a startup shortcut via PowerShell
func EnableAutostart() error {
	startup := startupFolder()
	if startup == "" {
		return errors.New("APPDATA not set")
	}

	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	linkPath := filepath.Join(startup, shortcutName)
	workDir := filepath.Dir(exePath)

	psCmd := "$s=(New-Object -COM WScript.Shell).CreateShortcut('" + linkPath + "');" +
		"$s.TargetPath='" + exePath + "';" +
		"$s.WorkingDirectory='" + workDir + "';" +
		"$s.Save()"

	cmd := exec.Command(
		"powershell",
		"-NoProfile",
		"-Command",
		psCmd,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	return cmd.Run()
}

// removes startup shortcut
func DisableAutostart() error {
	path := filepath.Join(startupFolder(), shortcutName)

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
