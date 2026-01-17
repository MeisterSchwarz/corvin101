//go:build windows

package system

import (
	"os"
	"os/exec"
	"syscall"
)

func Restart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(
		"cmd",
		"/C",
		"start",
		"",
		exe,
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	os.Exit(0)
	return nil
}
