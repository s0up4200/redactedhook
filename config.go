package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	APIKeys     APIKeys `mapstructure:"apikeys"`
	UserID      UserIDs `mapstructure:"userid"`
	Ratio       Ratio   `mapstructure:"ratio"`
	MinSize     int64   `mapstructure:"minsize"`
	MaxSize     int64   `mapstructure:"maxsize"`
	Uploaders   string  `mapstructure:"uploaders"`
	RecordLabel string  `mapstructure:"record_labels"`
	Mode        string  `mapstructure:"mode"`
	Logs        Logs    `mapstructure:"logs"`
}

type APIKeys struct {
	REDKey string `mapstructure:"red_apikey"`
	OPSKey string `mapstructure:"ops_apikey"`
}

type UserIDs struct {
	REDUserID int `mapstructure:"red_user_id"`
	OPSUserID int `mapstructure:"ops_user_id"`
}

type Ratio struct {
	MinRatio float64 `mapstructure:"minratio"`
}

type Logs struct {
	LogLevel string `mapstructure:"loglevel"`
}

var config Config

func initConfig(configPath string) {
	var configFile string

	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get user home directory, using current directory instead")
			configFile = filepath.Join(".", "config.toml")
		} else {
			configDir := filepath.Join(home, ".config", "redactedhook")
			configFile = filepath.Join(configDir, "config.toml")

			// Try creating the directory
			if _, err := os.Stat(configDir); os.IsNotExist(err) {
				if mkdirErr := os.MkdirAll(configDir, os.ModePerm); mkdirErr != nil {
					log.Error().Err(mkdirErr).Msg("Failed to create config directory in home, using current directory instead")
					configFile = filepath.Join(".", "config.toml")
				}
			}
		}
	} else {
		configFile = configPath
	}

	viper.SetConfigType("toml")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetConfigFile(configFile)

	// Check if the config file exists, and if not, create it with default values
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		content := []byte(`[apikeys]
#red_apikey = ""  # generate in user settings, needs torrent and user privileges
#ops_apikey = ""  # generate in user settings, needs torrent and user privileges
		
[userid]
#red_user_id = 0 # from /user.php?id=xxx
#ops_user_id = 0 # from /user.php?id=xxx
		
[ratio]
#minratio = 0.6 # reject releases if you are below this ratio

[logs]
loglevel = "trace" # trace, debug, info`)
		err := os.WriteFile(configFile, content, 0644)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create default config file")
		}
		log.Info().Msg("Created default config file")
	} else if err != nil {
		log.Fatal().Err(err).Msg("Failed to check if config file exists")
	}

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal().Err(err).Msg("Error reading config file")
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.Error().Err(err).Msg("Unable to unmarshal config")
	} else {
		log.Debug().Msgf("Config file read successfully: %s", viper.ConfigFileUsed())
		if config.Logs.LogLevel != "" {
			setLogLevel(config.Logs.LogLevel)
		} else {
			log.Warn().Msg("Log level not specified in config, using default")
			setLogLevel("debug")
		}
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Debug().Msgf("Config file updated: %s", e.Name)
		if err := viper.Unmarshal(&config); err != nil {
			log.Error().Err(err).Msg("Error reading config")
		} else {
			setLogLevel(config.Logs.LogLevel)
		}
	})
}

func setLogLevel(level string) {
	var loglevel zerolog.Level
	switch level {
	case "trace":
		loglevel = zerolog.TraceLevel
	case "debug":
		loglevel = zerolog.DebugLevel
	case "info":
		loglevel = zerolog.InfoLevel
	default:
		loglevel = zerolog.DebugLevel // default to DebugLevel if log level is empty
		level = "debug"
	}
	zerolog.SetGlobalLevel(loglevel)
	log.Debug().Msgf("Log level: %s", level)
}
