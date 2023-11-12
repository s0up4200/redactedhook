package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	APIKeys struct {
		REDKey string `mapstructure:"red_apikey"`
		OPSKey string `mapstructure:"ops_apikey"`
	} `mapstructure:"apikeys"`
	UserID struct {
		REDUserID int `mapstructure:"red_user_id"`
		OPSUserID int `mapstructure:"ops_user_id"`
	} `mapstructure:"userid"`
	MinRatio float64 `mapstructure:"minratio"`
}

func ensureConfigFileExists(configFile string) error {
	configDir := filepath.Dir(configFile)

	// Ensure the directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
		}
	}

	// Check if the file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Create the file with default content in TOML format
		defaultConfig := []byte(`
[apikeys]
red_apikey = "your-red-api-key"
ops_apikey = "your-ops-api-key"

[userid]
red_user_id = 0
ops_user_id = 0

[ratio]
minratio = 0.6
		`)
		return os.WriteFile(configFile, defaultConfig, 0644)
	}
	return nil
}

func LoadConfig(configFile string) (*Config, error) {
	if err := ensureConfigFileExists(configFile); err != nil {
		return nil, err
	}

	viper.SetConfigFile(configFile)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
