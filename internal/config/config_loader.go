package config

import (
	"errors"
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
	// Set default values before reading the config file
	viper.SetDefault("userid.red_user_id", 0)
	viper.SetDefault("userid.ops_user_id", 0)
	viper.SetDefault("ratio.minratio", 0)
	viper.SetDefault("sizecheck.minsize", "")
	viper.SetDefault("sizecheck.maxsize", "")
	viper.SetDefault("uploaders.uploaders", "")
	viper.SetDefault("uploaders.mode", "")
	viper.SetDefault("record_labels.record_labels", "")

	viper.SetConfigType(defaultConfigType)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetConfigFile(configFile)

	// Uncomment this if you want to ensure the config file exists
	// and create it if it does not.
	// if err := createConfigFileIfNotExist(configFile); err != nil {
	//	log.Fatal().Err(err).Msg("Failed to create or verify config file")
	// }

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal().Err(err).Msg("Error reading config file")
	}
}

func readAndUnmarshalConfig() {
	if err := viper.Unmarshal(&config); err != nil {
		log.Error().Err(err).Msg("Unable to unmarshal config")
	} else {
		parseSizeCheck()
		log.Debug().Msgf("Config file read: %s", viper.ConfigFileUsed())
		configureLogger()
	}
}

func parseSizeCheck() {
	// Parse MinSize
	minSizeStr := viper.GetString("sizecheck.minsize")
	if minSizeStr == "" {
		config.ParsedSizes.MinSize = 0 // Reset to default when empty string is provided
	} else {
		minSize, err := bytesize.Parse(minSizeStr)
		if err != nil {
			log.Error().Err(err).Msg("Invalid format for MinSize; unable to parse")
		} else {
			config.ParsedSizes.MinSize = minSize
		}
	}

	// Parse MaxSize
	maxSizeStr := viper.GetString("sizecheck.maxsize")
	if maxSizeStr == "" {
		config.ParsedSizes.MaxSize = 0 // Reset to default when empty string is provided
	} else {
		maxSize, err := bytesize.Parse(maxSizeStr)
		if err != nil {
			log.Error().Err(err).Msg("Invalid format for MaxSize; unable to parse")
		} else {
			config.ParsedSizes.MaxSize = maxSize
		}
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

func ValidateConfig() error {
	var validationErrors []string

	if !viper.IsSet("authorization.api_token") || viper.GetString("authorization.api_token") == "" {
		validationErrors = append(validationErrors, "Authorization API Token is required")
	}

	if viper.IsSet("indexer_keys.red_apikey") && viper.GetString("indexer_keys.red_apikey") == "" {
		validationErrors = append(validationErrors, "Indexer REDKey should not be empty")
	}

	if viper.IsSet("indexer_keys.ops_apikey") && viper.GetString("indexer_keys.ops_apikey") == "" {
		validationErrors = append(validationErrors, "Indexer OPSKey should not be empty")
	}

	//if viper.IsSet("userid.red_user_id") && viper.GetInt("userid.red_user_id") <= 0 {
	//	validationErrors = append(validationErrors, "Invalid RED User ID")
	//}

	//if viper.IsSet("userid.ops_user_id") && viper.GetInt("userid.ops_user_id") <= 0 {
	//	validationErrors = append(validationErrors, "Invalid OPS User ID")
	//}

	//if viper.IsSet("ratio.minratio") && viper.GetFloat64("ratio.minratio") <= 0 {
	//	validationErrors = append(validationErrors, "Minimum ratio should be positive")
	//}

	//if viper.IsSet("sizecheck.minsize") && viper.GetString("sizecheck.minsize") == "" {
	//	validationErrors = append(validationErrors, "Invalid minimum size")
	//}

	//if viper.IsSet("sizecheck.maxsize") && viper.GetString("sizecheck.maxsize") == "" {
	//	validationErrors = append(validationErrors, "Invalid maximum size")
	//}

	//if viper.IsSet("uploaders.uploaders") && viper.GetString("uploaders.uploaders") == "" {
	//	validationErrors = append(validationErrors, "Invalid uploader list")
	//}

	//if viper.IsSet("uploaders.mode") && viper.GetString("uploaders.mode") == "" {
	//	validationErrors = append(validationErrors, "Invalid uploader mode set")
	//}

	//if viper.IsSet("record_labels.record_labels") && viper.GetString("record_labels.record_labels") == "" {
	//	validationErrors = append(validationErrors, "Invalid record_labels set")
	//}

	if !viper.IsSet("logs.loglevel") || viper.GetString("logs.loglevel") == "" {
		validationErrors = append(validationErrors, "Log level is required")
	}

	if !viper.IsSet("logs.logtofile") {
		validationErrors = append(validationErrors, "Log to file flag is required")
	}

	if viper.GetBool("logs.logtofile") && (!viper.IsSet("logs.logfilepath") || viper.GetString("logs.logfilepath") == "") {
		validationErrors = append(validationErrors, "Log file path is required when logging to a file")
	}

	if !viper.IsSet("logs.maxsize") || viper.GetInt("logs.maxsize") <= 0 {
		validationErrors = append(validationErrors, "Max log file size should be a positive integer")
	}

	if !viper.IsSet("logs.maxbackups") || viper.GetInt("logs.maxbackups") < 0 {
		validationErrors = append(validationErrors, "Max backups should be a non-negative integer")
	}

	if !viper.IsSet("logs.maxage") || viper.GetInt("logs.maxage") <= 0 {
		validationErrors = append(validationErrors, "Max age should be a positive integer")
	}

	if !viper.IsSet("logs.compress") {
		validationErrors = append(validationErrors, "Compress flag is required")
	}

	if len(validationErrors) > 0 {
		return errors.New(strings.Join(validationErrors, "; "))
	}

	return nil
}
