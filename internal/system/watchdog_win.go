//go:build windows

package system

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const defaultWatchInterval = 5 * time.Second

var wizardProcessNames = []string{
	"WizardGraphicalClient.exe",
	"Wizard101.exe",
}

type GameEvent int

const (
	GameStarted GameEvent = iota
	GameStopped
)

type GameWatcher struct {
	interval time.Duration
	events   chan GameEvent
}

func NewGameWatcher() *GameWatcher {
	return &GameWatcher{
		interval: defaultWatchInterval,
		events:   make(chan GameEvent, 4),
	}
}

func (w *GameWatcher) Events() <-chan GameEvent {
	return w.events
}

func (w *GameWatcher) Start(ctx context.Context) error {
	running, err := isWizardRunning()
	if err != nil {
		return fmt.Errorf("check initial Wizard101 state: %w", err)
	}

	go w.loop(ctx, running)

	return nil
}

func (w *GameWatcher) loop(ctx context.Context, lastRunning bool) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	defer close(w.events)

	// Wenn Wizard101 bereits läuft, während Corvin gestartet wird,
	// sofort ein Start-Event auslösen.
	if lastRunning {
		w.send(ctx, GameStarted)
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			running, err := isWizardRunning()
			if err != nil {
				// Ein fehlgeschlagener Check darf nicht so behandelt
				// werden, als wäre Wizard101 beendet worden.
				continue
			}

			if running == lastRunning {
				continue
			}

			lastRunning = running

			if running {
				w.send(ctx, GameStarted)
			} else {
				w.send(ctx, GameStopped)
			}
		}
	}
}

func (w *GameWatcher) send(ctx context.Context, event GameEvent) {
	select {
	case w.events <- event:
	case <-ctx.Done():
	}
}

func isWizardRunning() (bool, error) {
	for _, processName := range wizardProcessNames {
		running, err := processExists(processName)
		if err != nil {
			return false, err
		}

		if running {
			return true, nil
		}
	}

	return false, nil
}

func processExists(processName string) (bool, error) {
	cmd := exec.Command(
		"tasklist",
		"/FI",
		"IMAGENAME eq "+processName,
		"/FO",
		"CSV",
		"/NH",
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf(
			"tasklist %s: %w",
			processName,
			err,
		)
	}

	result := strings.ToLower(string(output))
	target := strings.ToLower(processName)

	return strings.Contains(result, target), nil
}
