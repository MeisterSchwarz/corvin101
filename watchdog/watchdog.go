package watchdog

import (
	"log"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Poll interval for process checks
const checkEvery = 5 * time.Second

// Tracked process names
var processNames = []string{
	"WizardGraphicalClient.exe",
	"Wizard101.exe",
}

var lastRunning bool

// Game lifecycle events
var GameStarted = make(chan struct{}, 1)
var GameStopped = make(chan struct{}, 1)

// launches watchdog loop
func Start() {
	go loop()
}

// periodically checks game state
func loop() {
	ticker := time.NewTicker(checkEvery)
	defer ticker.Stop()

	for range ticker.C {
		running := isWizardRunning()

		switch {
		case !lastRunning && running:
			notify(GameStarted)

		case lastRunning && !running:
			notify(GameStopped)
		}

		lastRunning = running
	}
}

// checks if any game process is active
func isWizardRunning() bool {
	for _, name := range processNames {
		if processExists(name) {
			return true
		}
	}
	return false
}

// checks for process via tasklist
func processExists(name string) bool {
	cmd := exec.Command(
		"tasklist",
		"/FI", "IMAGENAME eq "+name,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	out, err := cmd.Output()

	if err != nil {
		log.Printf("[Watchdog] tasklist failed: %v", err)
		return false
	}

	return strings.Contains(string(out), name)
}

// sends event without blocking
func notify(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}
