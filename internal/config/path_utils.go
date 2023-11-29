package config

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

func isRunningInDocker() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

func determineConfigFile(configPath string) string {
	if configPath != "" {
		return configPath
	}

	var configDir string
	if isRunningInDocker() {
		// In Docker, default to the mapped volume directory
		configDir = os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = "/redactedhook"
		}
	} else {
		// For non-Docker, use the user's home directory with .config/redactedhook/
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get user home directory")
		}
		configDir = filepath.Join(homeDir, defaultConfigDir)
	}

	configFile := filepath.Join(configDir, defaultConfigFileName)

	//// Ensure the config file exists
	//if err := createConfigFileIfNotExist(configFile); err != nil {
	//	log.Fatal().Err(err).Msg("Failed to create or verify config file")
	//}

	return configFile
}
