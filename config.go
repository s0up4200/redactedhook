package main

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
	UserID      UserIDs   `mapstructure:"userid"`
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

func initConfig(configPath string) {
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

	var configDir string
	if isRunningInDocker() {
		// In Docker, default to the mapped volume directory
		configDir = "/redactedhook"
	} else {
		// For non-Docker, use the user's home directory with .config/redactedhook/
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get user home directory")
		}
		configDir = filepath.Join(homeDir, ".config", "redactedhook")
	}

	configFile := filepath.Join(configDir, defaultConfigFileName)

	// Ensure the config file exists
	if err := createConfigFileIfNotExist(configFile); err != nil {
		log.Fatal().Err(err).Msg("Failed to create or verify config file")
	}

	return configFile
}

func createConfigFileIfNotExist(configFile string) error {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Create the default config file
		defaultConfig := getDefaultConfig() // Ensure this function returns your default config
		if err := os.WriteFile(configFile, defaultConfig, 0644); err != nil {
			return err
		}
		log.Info().Msg("Created default config file")
	}
	return nil
}

func setupViper(configFile string) {
	viper.SetConfigType(defaultConfigType)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetConfigFile(configFile)

	if err := createConfigFileIfNotExist(configFile); err != nil {
		log.Fatal().Err(err).Msg("Failed to create or verify config file")
	}

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
		log.Debug().Msgf("Config file updated: %s", e.Name)
		oldLogLevel := config.Logs.LogLevel
		if err := viper.Unmarshal(&config); err != nil {
			log.Error().Err(err).Msg("Error reading config")
		} else {
			if oldLogLevel != config.Logs.LogLevel {
				configureLogger()
			}
		}
	})
}

func configureLogger() {
	var writers []io.Writer

	// Always log to console
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"}
	writers = append(writers, consoleWriter)

	// If logtofile is true, also log to file
	if config.Logs.LogToFile {
		fileWriter := &lumberjack.Logger{
			Filename:   config.Logs.LogFilePath,
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
	log.Info().Msgf("Log level: %s", level)
}
