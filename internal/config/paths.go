package config

import (
	"path/filepath"

	"corvin101/internal/filesystem"
)

func configPath() string {
	return filepath.Join(
		filesystem.AppDataDir(),
		"config.json",
	)
}
