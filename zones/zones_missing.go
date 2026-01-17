package zones

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"wizard101rpc/filesystem"
	"wizard101rpc/state"

	"github.com/go-toast/toast"
)

// Missing zone log entry
type MissingZoneEntry struct {
	Time time.Time `json:"time"`
	Zone string    `json:"zone"`
}

var (
	missingMu sync.Mutex
	seen      = map[string]bool{} // session-scoped deduplication
)

// shows a Windows toast notification
func NotifyMissingZone(zone string) {
	notification := toast.Notification{
		AppID:   "Wizard101 RPC",
		Title:   "Unbekanntes Gebiet",
		Message: zone,
	}

	_ = notification.Push()
}

// logs and notifies about unknown zones
func reportMissingZone(zoneKey string) {
	if !IsContributing() {
		return
	}

	sessionKey := state.SessionStart().String() + ":" + zoneKey

	missingMu.Lock()
	if seen[sessionKey] {
		missingMu.Unlock()
		return
	}
	seen[sessionKey] = true
	missingMu.Unlock()

	NotifyMissingZone(zoneKey)

	entry := MissingZoneEntry{
		Time: time.Now().UTC(),
		Zone: zoneKey,
	}

	appendMissingZone(entry)
}

// writes entry to foile
func appendMissingZone(entry MissingZoneEntry) {

	path := filepath.Join(
		filesystem.AppDataDir(),
		"missing_zones.log",
	)

	f, err := os.OpenFile(
		path,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return
	}
	defer f.Close()

	if raw, err := json.Marshal(entry); err == nil {
		f.Write(raw)
		f.Write([]byte("\n"))
	}
}
