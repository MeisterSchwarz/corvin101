package filesystem

import (
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
		"wizard101rpc",
	)
	_ = os.MkdirAll(dir, 0755)
	return dir
}
