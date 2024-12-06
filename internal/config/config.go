package config

import (
	"fmt"
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

func CreateConfigFile() string {
	config := `[server]
host = "127.0.0.1" # Server host
port = 42135       # Server port

[authorization]
api_token = "ch4ng3this" # generate with "redactedhook generate-apitoken"
# the api_token needs to be set as a header for the webhook to work
# eg. Header: X-API-Token=aaa129cd1d66ed6fa567da2d07a5dd0e

[indexer_keys]
#red_apikey = "" # generate in user settings, needs torrent and user privileges
#ops_apikey = "" # generate in user settings, needs torrent and user privileges

[userid]
#red_user_id = 0 # from /user.php?id=xxx
#ops_user_id = 0 # from /user.php?id=xxx

[ratio]
#minratio = 0.6 # reject releases if you are below this ratio

[sizecheck]
#minsize = "100MB" # minimum size for checking, e.g., "10MB"
#maxsize = "500MB" # maximum size for checking, e.g., "1GB"

[uploaders]
#uploaders = "greatest-uploader" # comma separated list of uploaders to allow
#mode = "whitelist" # whitelist or blacklist

[record_labels]
#record_labels = "" # comma separated list of record labels to filter for

[logs]
loglevel = "trace"               # trace, debug, info
logtofile = false                # Set to true to enable logging to a file
logfilepath = "redactedhook.log" # Path to the log file
maxsize = 10                     # Max file size in MB
maxbackups = 3                   # Max number of old log files to keep
maxage = 28                      # Max age in days to keep a log file
compress = false                 # Whether to compress old log files

[notifications]
discord_webhook_url = "" # URL for Discord webhook notifications
`

	err := os.WriteFile(defaultConfigFileName, []byte(config), 0644)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to write default configuration file")
	}
	fmt.Println("Configuration file 'config.toml' generated.")
	return defaultConfigFileName
}
