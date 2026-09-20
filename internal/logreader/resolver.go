package logreader

import (
	"path/filepath"

	"corvin101/internal/config"
	"corvin101/internal/filesystem"
)

const logFileName = "WizardClient.log"

var defaultSearchPaths = []string{
	// Standalone (DE)
	`C:\ProgramData\KingsIsle Entertainment\Wizard101(DE)\Bin\WizardClient.log`,

	// Standalone (EN)
	`C:\ProgramData\KingsIsle Entertainment\Wizard101\Bin\WizardClient.log`,

	// Steam default library
	`C:\Program Files (x86)\Steam\steamapps\common\Wizard101\Bin\WizardClient.log`,
}

func ResolveLogPath(
	cfg config.Config,
) (string, bool) {
	if cfg.InstallDir != "" {
		candidate := filepath.Join(
			cfg.InstallDir,
			"Bin",
			logFileName,
		)

		if filesystem.FileExists(candidate) {
			return candidate, true
		}
	}

	for _, path := range defaultSearchPaths {
		if filesystem.FileExists(path) {
			return path, true
		}
	}

	return "", false
}
