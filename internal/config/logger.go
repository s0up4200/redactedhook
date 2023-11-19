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
}
