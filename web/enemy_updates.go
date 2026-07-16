package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var enemyUpdateMu sync.Mutex

// SetPendingEnemyName speichert oder aktualisiert eine deutsche
// Gegnerübersetzung in der separaten Namensdatei.
//
// Dateiformat:
//
//	{
//	  "Skeleton-Pirate-L01": "Skelettpirat"
//	}
func SetPendingEnemyName(
	namesPath string,
	enemyKey string,
	enemyName string,
) error {
	enemyUpdateMu.Lock()
	defer enemyUpdateMu.Unlock()

	enemyKey = strings.TrimSpace(enemyKey)
	enemyName = strings.TrimSpace(enemyName)

	if enemyKey == "" {
		return errors.New("enemy key must not be empty")
	}

	if enemyName == "" {
		return errors.New("enemy name must not be empty")
	}

	names := make(map[string]string)

	if err := readJSONIfExists(
		namesPath,
		&names,
	); err != nil {
		return fmt.Errorf(
			"read pending enemy names: %w",
			err,
		)
	}

	if names == nil {
		names = make(map[string]string)
	}

	// Keine Datei neu schreiben, wenn sich nichts geändert hat.
	if names[enemyKey] == enemyName {
		return nil
	}

	names[enemyKey] = enemyName

	if err := writeJSONAtomic(
		namesPath,
		names,
	); err != nil {
		return fmt.Errorf(
			"write pending enemy names: %w",
			err,
		)
	}

	return nil
}

func readJSONIfExists(
	path string,
	target any,
) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return err
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf(
			"invalid JSON in %s: %w",
			path,
			err,
		)
	}

	return nil
}

func marshalJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(
		value,
		"",
		"  ",
	)
	if err != nil {
		return nil, err
	}

	return append(data, '\n'), nil
}

func writeJSONAtomic(
	path string,
	value any,
) error {
	data, err := marshalJSON(value)
	if err != nil {
		return err
	}

	return writeBytesAtomic(path, data)
}

func writeBytesAtomic(
	path string,
	data []byte,
) error {
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf(
			"create directory %s: %w",
			dir,
			err,
		)
	}

	tempFile, err := os.CreateTemp(
		dir,
		".json-update-*",
	)
	if err != nil {
		return err
	}

	tempPath := tempFile.Name()
	removeTemp := true

	defer func() {
		_ = tempFile.Close()

		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := tempFile.Write(data); err != nil {
		return err
	}

	if err := tempFile.Sync(); err != nil {
		return err
	}

	if err := tempFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		return err
	}

	removeTemp = false
	return nil
}
