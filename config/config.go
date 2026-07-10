package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"wizlink/filesystem"
)

const (
	DiscordClientID = "1451506790910136413"
)

type Config struct {
	InstallDir string `json:"install_dir"`
}

var AppConfig Config

// config file location (per-user)
var configPath = filepath.Join(
	filesystem.AppDataDir(),
	"config.json",
)

// loads the config from disk into AppConfig
func Load() error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &AppConfig)
}

// persists AppConfig to disk
func Save() error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(AppConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// known Wizard101 log locations
var LogSearchPaths = []string{
	// Standalone (DE)
	`C:\ProgramData\KingsIsle Entertainment\Wizard101(DE)\Bin\WizardClient.log`,

	// Standalone (EN)
	`C:\ProgramData\KingsIsle Entertainment\Wizard101\Bin\WizardClient.log`,

	// Steam (default library)
	`C:\Program Files (x86)\Steam\steamapps\common\Wizard101\Bin\WizardClient.log`,
}
