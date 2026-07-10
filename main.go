package main

import (
	"log"
	"time"

	"wizlink/config"
	"wizlink/discord"
	"wizlink/enemy"
	"wizlink/logreader"
	"wizlink/state"
	"wizlink/system"
	"wizlink/tray"
	"wizlink/watchdog"
)

func main() {
	if !system.Lock() {
		return
	}
	defer system.Unlock()

	if err := config.Load(); err != nil {
		log.Println("[config] no config found, using defaults")
	}

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

				logreader.Stop()
				time.Sleep(300 * time.Millisecond)

				enemyTracker := enemy.NewEnemyTracker()

				logreader.Watch(
					logPath,
					discord.HandleLogLine,
					enemyTracker.HandleLogLine,
				)

			case <-watchdog.GameStopped:
				logreader.Stop()
				discord.Logout()
				state.Reset()
			}
		}
	}()

	select {}
}
