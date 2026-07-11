package filesystem

import (
	"embed"
	"os"
	"path"
	"path/filepath"
)

//go:embed json/**
var FS embed.FS

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

func PendingEnemyNamesPath() string {
	return filepath.Join(
		AppDataDir(),
		"pending",
		"enemy_names.json",
	)
}

func PendingEnemyZonesPath() string {
	return filepath.Join(
		AppDataDir(),
		"pending",
		"enemy_zones.json",
	)
}

func ReadJSON(language, category, file string) ([]byte, error) {
	return FS.ReadFile(path.Join(
		"json",
		language,
		category,
		file,
	))
}
