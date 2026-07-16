package logreader

import (
	"path/filepath"

	"ravendex/config"
	"ravendex/filesystem"
)

// Resolve returns the first existing Wizard101 log path
func ResolveLogPath() (string, bool) {
	if config.AppConfig.InstallDir != "" {
		candidate := filepath.Join(
			config.AppConfig.InstallDir,
			"Bin",
			"WizardClient.log",
		)

		if filesystem.FileExists(candidate) {
			return candidate, true
		}
	}

	for _, path := range config.LogSearchPaths {
		if filesystem.FileExists(path) {
			return path, true
		}
	}

	return "", false
}
