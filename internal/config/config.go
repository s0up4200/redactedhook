package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/inhies/go-bytesize"
	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	defaultConfigFileName = "config.toml"
	defaultConfigType     = "toml"
	defaultConfigDir      = ".config/redactedhook"
	defaultLogLevel       = "trace"
)

var config Config

type Config struct {
	APIKeys     APIKeys   `mapstructure:"apikeys"`
	UserIDs     UserIDs   `mapstructure:"userid"`
	Ratio       Ratio     `mapstructure:"ratio"`
	SizeCheck   SizeCheck `mapstructure:"sizecheck"`
	ParsedSizes ParsedSizeCheck
	Uploaders   Uploaders `mapstructure:"uploaders"`
	Logs        Logs      `mapstructure:"logs"`
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

type SizeCheck struct {
	MinSize string `mapstructure:"minsize"`
	MaxSize string `mapstructure:"maxsize"`
}

type ParsedSizeCheck struct {
	MinSize bytesize.ByteSize
	MaxSize bytesize.ByteSize
}

type Uploaders struct {
	Uploaders string `mapstructure:"uploaders"`
	Mode      string `mapstructure:"mode"`
}

type Logs struct {
	LogLevel    string `mapstructure:"loglevel"`
	LogToFile   bool   `mapstructure:"logtofile"`
	LogFilePath string `mapstructure:"logfilepath"`
	MaxSize     int    `mapstructure:"maxsize"`    // Max file size in MB
	MaxBackups  int    `mapstructure:"maxbackups"` // Max number of old log files to keep
	MaxAge      int    `mapstructure:"maxage"`     // Max age in days to keep a log file
	Compress    bool   `mapstructure:"compress"`   // Whether to compress old log files
}

func GetConfig() *Config {
	return &config
}

func InitConfig(configPath string) {
	configFile := determineConfigFile(configPath)
	setupViper(configFile)
	readAndUnmarshalConfig()
	watchConfigChanges()
}

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

	configDir := defaultConfigDir
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

//func createConfigFileIfNotExist(configFile string) error {
//	if _, err := os.Stat(configFile); os.IsNotExist(err) {
//		// Create the default config file
//		defaultConfig := getDefaultConfig() // Ensure this function returns your default config
//		if err := os.WriteFile(configFile, defaultConfig, 0644); err != nil {
//			return err
//		}
//		log.Info().Msg("Created default config file")
//	}
//	return nil
//}

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

func getDefaultConfig() []byte {
	return []byte(`[apikeys]
#red_apikey = ""  # generate in user settings, needs torrent and user privileges
#ops_apikey = ""  # generate in user settings, needs torrent and user privileges

[userid]
#red_user_id = 0  # from /user.php?id=xxx
#ops_user_id = 0  # from /user.php?id=xxx

[ratio]
#minratio = 0.6   # reject releases if you are below this ratio

[sizecheck]
#minsize = ""     # minimum size for checking, e.g., "10MB"
#maxsize = ""     # maximum size for checking, e.g., "1GB"

[uploaders]
#uploaders = ""   # comma separated list of uploaders to allow
#mode = ""        # whitelist or blacklist

[logs]
loglevel = "trace" # trace, debug, info
logtofile = false  # Set to true to enable logging to a file
logfilepath = ""   # Path to the log file
maxsize = 0        # Max file size in MB
maxbackups = 0     # Max number of old log files to keep
maxage = 0         # Max age in days to keep a log file
compress = false   # Whether to compress old log files
`)
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

	if oldConfig.APIKeys.REDKey != newConfig.APIKeys.REDKey { // APIKeys
		log.Debug().Msg("red_apikey changed")
	}
	if oldConfig.APIKeys.OPSKey != newConfig.APIKeys.OPSKey {
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

func configureLogger() {
	var writers []io.Writer

	// Always log to console
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"}
	writers = append(writers, consoleWriter)

	// If logtofile is true, also log to file
	if config.Logs.LogToFile {
		logFilePath := config.Logs.LogFilePath
		if logFilePath == "" && isRunningInDocker() {
			logFilePath = "/redactedhook/redactedhook.log" // Use a sensible default in Docker
		}
		fileWriter := &lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    config.Logs.MaxSize,    // megabytes
			MaxBackups: config.Logs.MaxBackups, // number of backups
			MaxAge:     config.Logs.MaxAge,     // days
			Compress:   config.Logs.Compress,   // compress rolling files
		}
		writers = append(writers, fileWriter)
	}

	// Combine all writers
	multiWriter := zerolog.MultiLevelWriter(writers...)
	log.Logger = zerolog.New(multiWriter).With().Timestamp().Logger()

	// Set the log level
	setLogLevel(config.Logs.LogLevel)
}

func setLogLevel(level string) {
	loglevel, err := zerolog.ParseLevel(level)
	if err != nil {
		// If the provided log level is invalid, log an error and default to debug level.
		log.Error().Msgf("Invalid log level '%s', defaulting to 'debug'", level)
		loglevel = zerolog.DebugLevel
	}

	// Apply the determined log level.
	zerolog.SetGlobalLevel(loglevel)
	//log.Info().Msgf("Set log level to '%s'", loglevel.String())
}
