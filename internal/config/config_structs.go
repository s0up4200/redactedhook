package config

import "github.com/inhies/go-bytesize"

var config Config

type Config struct {
	IndexerKeys   IndexerKeys   `mapstructure:"indexer_keys"`
	Authorization Authorization `mapstructure:"authorization"`
	UserIDs       UserIDs       `mapstructure:"userid"`
	Ratio         Ratio         `mapstructure:"ratio"`
	SizeCheck     SizeCheck     `mapstructure:"sizecheck"`
	ParsedSizes   ParsedSizeCheck
	Uploaders     Uploaders    `mapstructure:"uploaders"`
	RecordLabels  RecordLabels `mapstructure:"record_labels"`
	Logs          Logs         `mapstructure:"logs"`
}

type Authorization struct {
	APIToken string `mapstructure:"api_token"`
}

type IndexerKeys struct {
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

type RecordLabels struct {
	RecordLabels string `mapstructure:"record_labels"`
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
