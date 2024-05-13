package config

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

func isRunningInDocker() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func determineConfigFile(configPath string) string {
	if configPath != "" {
		return configPath
	}

	var configDir string
	if isRunningInDocker() {
		configDir = os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = "/redactedhook"
		}
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get user home directory")
		}
		configDir = filepath.Join(homeDir, defaultConfigDir)
	}

	configFile := filepath.Join(configDir, defaultConfigFileName)
	return configFile
}
