package filesystem

import (
	"os"
	"path/filepath"
)

const appName = "corvin101"

func FileExists(path string) bool {
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

func DirectoryExists(path string) bool {
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func AppDataDir() string {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		// Windows fallback.
		baseDir = os.Getenv("APPDATA")
	}

	dir := filepath.Join(
		baseDir,
		appName,
	)

	_ = os.MkdirAll(
		dir,
		0755,
	)

	return dir
}
