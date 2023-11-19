package config

import (
	"os"

	"github.com/rs/zerolog/log"
)

const (
	defaultConfigFileName = "config.toml"
	defaultConfigType     = "toml"
	defaultConfigDir      = ".config/redactedhook"
	defaultLogLevel       = "trace"
)

func GetConfig() *Config {
	return &config
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

func getDefaultConfig() []byte {
	return []byte(`[authorization]
api_token = ""    # generate with ./redactedhook generate-apitoken

[indexer_keys]
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

[record_labels]
#record_labels = "" # comma separated list of record labels to filter for

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
