package config

import (
	"io"
	"os"

	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func configureLogger() {
	var writers []io.Writer

	// always log to console
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"}
	writers = append(writers, consoleWriter)

	if config.Logs.LogToFile {
		logFilePath := determineLogFilePath()

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

	setLogLevel(config.Logs.LogLevel)
}

func setLogLevel(level string) {
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		log.Error().Msgf("Invalid log level '%s', defaulting to 'debug'", level)
		logLevel = zerolog.DebugLevel
	}

	zerolog.SetGlobalLevel(logLevel)
}

func determineLogFilePath() string {
	logFilePath := config.Logs.LogFilePath
	if logFilePath == "" && isRunningInDocker() {
		// use a sensible default log file path in Docker
		logFilePath = "/redactedhook/redactedhook.log"
	}
	return logFilePath
}
