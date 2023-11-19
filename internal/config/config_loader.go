package config

import (
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/inhies/go-bytesize"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func InitConfig(configPath string) {
	configFile := determineConfigFile(configPath)
	setupViper(configFile)
	readAndUnmarshalConfig()
	watchConfigChanges()
}

func setupViper(configFile string) {
	viper.SetConfigType(defaultConfigType)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetConfigFile(configFile)

	//if err := createConfigFileIfNotExist(configFile); err != nil {
	//	log.Fatal().Err(err).Msg("Failed to create or verify config file")
	//}

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal().Err(err).Msg("Error reading config file")
	}
}

func readAndUnmarshalConfig() {
	if err := viper.Unmarshal(&config); err != nil {
		log.Error().Err(err).Msg("Unable to unmarshal config")
	} else {
		parseSizeCheck()
		log.Debug().Msgf("Config file read successfully: %s", viper.ConfigFileUsed())
		configureLogger()
	}
}

func parseSizeCheck() {
	if config.SizeCheck.MinSize != "" && !strings.HasPrefix(config.SizeCheck.MinSize, "#") {
		minSize, err := bytesize.Parse(config.SizeCheck.MinSize)
		if err != nil {
			log.Error().Err(err).Msg("Invalid format for minsize")
			return
		}
		config.ParsedSizes.MinSize = minSize
	}

	if config.SizeCheck.MaxSize != "" && !strings.HasPrefix(config.SizeCheck.MaxSize, "#") {
		maxSize, err := bytesize.Parse(config.SizeCheck.MaxSize)
		if err != nil {
			log.Error().Err(err).Msg("Invalid format for maxsize")
			return
		}
		config.ParsedSizes.MaxSize = maxSize
	}
}

func watchConfigChanges() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {

		oldConfig := config

		if err := viper.ReadInConfig(); err != nil {
			log.Error().Err(err).Msg("Error reading config")
			return
		}
		if err := viper.Unmarshal(&config); err != nil {
			log.Error().Err(err).Msg("Error unmarshalling config")
			return
		}

		parseSizeCheck()

		logConfigChanges(oldConfig, config)

		if oldConfig.Logs.LogLevel != config.Logs.LogLevel {
			configureLogger()
		}
		log.Debug().Msgf("Config file updated: %s", e.Name)
	})
}

func logConfigChanges(oldConfig, newConfig Config) {

	if oldConfig.IndexerKeys.REDKey != newConfig.IndexerKeys.REDKey { // IndexerKeys
		log.Debug().Msg("red_apikey changed")
	}
	if oldConfig.IndexerKeys.OPSKey != newConfig.IndexerKeys.OPSKey {
		log.Debug().Msg("ops_apikey changed")
	}

	if oldConfig.UserIDs.REDUserID != newConfig.UserIDs.REDUserID { // UserIDs
		log.Debug().Msgf("REDUserID changed from %d to %d", oldConfig.UserIDs.REDUserID, newConfig.UserIDs.REDUserID)
	}
	if oldConfig.UserIDs.OPSUserID != newConfig.UserIDs.OPSUserID {
		log.Debug().Msgf("OPSUserID changed from %d to %d", oldConfig.UserIDs.OPSUserID, newConfig.UserIDs.OPSUserID)
	}

	if oldConfig.Ratio.MinRatio != newConfig.Ratio.MinRatio { // Ratio
		log.Debug().Msgf("MinRatio changed from %f to %f", oldConfig.Ratio.MinRatio, newConfig.Ratio.MinRatio)
	}

	oldMinSize, _ := bytesize.Parse(oldConfig.SizeCheck.MinSize)
	newMinSize, _ := bytesize.Parse(newConfig.SizeCheck.MinSize)
	if oldMinSize != newMinSize { // SizeCheck
		log.Debug().Msgf("MinSize changed from %s to %s", oldConfig.SizeCheck.MinSize, newConfig.SizeCheck.MinSize)
	}

	oldMaxSize, _ := bytesize.Parse(oldConfig.SizeCheck.MaxSize)
	newMaxSize, _ := bytesize.Parse(newConfig.SizeCheck.MaxSize)
	if oldMaxSize != newMaxSize { // SizeCheck
		log.Debug().Msgf("MaxSize changed from %s to %s", oldConfig.SizeCheck.MaxSize, newConfig.SizeCheck.MaxSize)
	}

	if oldConfig.Uploaders.Uploaders != newConfig.Uploaders.Uploaders { // Uploaders
		log.Debug().Msgf("Uploaders changed from %s to %s", oldConfig.Uploaders.Uploaders, newConfig.Uploaders.Uploaders)
	}
	if oldConfig.Uploaders.Mode != newConfig.Uploaders.Mode { // Uploaders
		log.Debug().Msgf("Uploader mode changed from %s to %s", oldConfig.Uploaders.Mode, newConfig.Uploaders.Mode)
	}

	if oldConfig.Logs.LogLevel != newConfig.Logs.LogLevel { // Logs
		log.Debug().Msgf("Log level changed from %s to %s", oldConfig.Logs.LogLevel, newConfig.Logs.LogLevel)
	}
	if oldConfig.Logs.LogToFile != newConfig.Logs.LogToFile { // Logs
		log.Debug().Msgf("LogToFile changed from %t to %t", oldConfig.Logs.LogToFile, newConfig.Logs.LogToFile)
	}
	if oldConfig.Logs.LogFilePath != newConfig.Logs.LogFilePath { // Logs
		log.Debug().Msgf("LogFilePath changed from %s to %s", oldConfig.Logs.LogFilePath, newConfig.Logs.LogFilePath)
	}
	if oldConfig.Logs.MaxSize != newConfig.Logs.MaxSize { // Logs
		log.Debug().Msgf("Logs MaxSize changed from %d to %d", oldConfig.Logs.MaxSize, newConfig.Logs.MaxSize)
	}
	if oldConfig.Logs.MaxBackups != newConfig.Logs.MaxBackups { // Logs
		log.Debug().Msgf("Logs MaxBackups changed from %d to %d", oldConfig.Logs.MaxBackups, newConfig.Logs.MaxBackups)
	}
	if oldConfig.Logs.MaxAge != newConfig.Logs.MaxAge { // Logs
		log.Debug().Msgf("Logs MaxAge changed from %d to %d", oldConfig.Logs.MaxAge, newConfig.Logs.MaxAge)
	}
	if oldConfig.Logs.Compress != newConfig.Logs.Compress { // Logs
		log.Debug().Msgf("Logs Compress changed from %t to %t", oldConfig.Logs.Compress, newConfig.Logs.Compress)
	}
}
