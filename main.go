package main

import (
	"log"
	"time"

	"ravendex/config"
	"ravendex/discord"
	"ravendex/enemy"
	"ravendex/logreader"
	"ravendex/state"
	"ravendex/system"
	"ravendex/tray"
	"ravendex/watchdog"
	"ravendex/web"
)

func main() {
	if !system.Lock() {
		return
	}
	defer system.Unlock()

	if err := config.Load(); err != nil {
		log.Println("[config] no config found, using defaults")
	}

	enemyTracker := enemy.NewEnemyTracker()

	server := web.NewServer(enemyTracker, "de")
	go func() {
		if err := server.Start("127.0.0.1:8101"); err != nil {
			log.Println("[web]", err)
		}
	}()

	go tray.Run()
	watchdog.Start()

	go func() {
		for {
			select {
			case <-watchdog.GameStarted:
				if err := discord.EnsureConnected(); err != nil {
					log.Println("[main] Discord connect failed:", err)
					continue
				}

				logPath, ok := logreader.ResolveLogPath()
				if !ok {
					continue
				}

				state.StartSession()

				lines := logreader.ReadAllLines(logPath)
				discord.InitFromLog(lines)

				// Bereits bekannte Zone aus dem eingelesenen Log übernehmen.
				syncTrackerZone(enemyTracker)

				logreader.Stop()
				time.Sleep(300 * time.Millisecond)

				logreader.Watch(
					logPath,
					func(line string) {
						handleGameLogLine(enemyTracker, line)
					},
				)

			case <-watchdog.GameStopped:
				logreader.Stop()
				discord.Logout()
				state.Reset()
				enemyTracker.Reset()
			}
		}
	}()

	select {}
}

func handleGameLogLine(
	tracker *enemy.EnemyTracker,
	line string,
) {
	// Aktualisiert unter anderem state.LastZone().
	discord.HandleLogLine(line)

	// Die Welt muss gesetzt werden, bevor eine mögliche Combat-Zeile
	// vom EnemyTracker verarbeitet wird.
	syncTrackerZone(tracker)

	tracker.HandleLogLine(line)
}

func syncTrackerZone(tracker *enemy.EnemyTracker) {
	zoneKey := state.LastZone()
	if zoneKey == "" {
		return
	}

	tracker.SetZoneKey(zoneKey)
}
