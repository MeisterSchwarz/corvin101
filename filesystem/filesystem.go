package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
)

func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func AppDataDir() string {
	dir := filepath.Join(
		os.Getenv("APPDATA"),
		"wizlink",
	)

	_ = os.MkdirAll(dir, 0755)

	return dir
}

func PendingEnemyNamesPath(
	language string,
	world string,
) string {
	filename := fmt.Sprintf(
		"enemy_names__%s__%s.json",
		world,
		language,
	)

	return filepath.Join(
		AppDataDir(),
		"pending",
		filename,
	)
}

func PendingEnemyZonesPath(
	language string,
	world string,
) string {
	filename := fmt.Sprintf(
		"enemy_zones__%s__%s.json",
		world,
		language,
	)

	return filepath.Join(
		AppDataDir(),
		"pending",
		filename,
	)
}
