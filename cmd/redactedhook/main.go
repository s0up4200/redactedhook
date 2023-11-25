package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/s0up4200/redactedhook/internal/api"
	"github.com/s0up4200/redactedhook/internal/config"
)

var (
	version   string
	commit    string
	buildDate string
)

const (
	path             = "/hook"
	EnvServerAddress = "SERVER_ADDRESS"
	EnvServerPort    = "SERVER_PORT"
)

func generateAPIToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		log.Fatal().Err(err).Msg("Failed to generate API key")
		return ""
	}
	apiKey := hex.EncodeToString(b)
	// codeql-ignore-next-line: go/clear-text-logging-of-sensitive-information
	fmt.Fprintf(os.Stdout, "API Token: %v, copy and paste into your config.toml\n", apiKey)
	return apiKey
}

func flagCommands() (string, bool) {
	var configPath string
	flag.StringVar(&configPath, "config", "config.toml", "Path to the configuration file")
	flag.Parse()

	if len(flag.Args()) > 0 {
		switch flag.Arg(0) {
		case "generate-apitoken":
			return generateAPIToken(16), true
		case "create-config":
			return config.CreateConfigFile(), true
		default:
			log.Fatal().Msgf("Unknown command: %s", flag.Arg(0))
		}
	}
	return configPath, false
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func startHTTPServer(address, port string) {
	server := &http.Server{Addr: address + ":" + port}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	log.Info().Msgf("Starting server on %s", address+":"+port)
	log.Info().Msgf("Version: %s, Commit: %s, Build Date: %s", version, commit, buildDate)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server shutdown failed")
	} else {
		log.Info().Msg("Server gracefully stopped")
	}
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05", NoColor: false})

	configPath, isCommandExecuted := flagCommands()
	if isCommandExecuted {
		return
	}

	config.InitConfig(configPath)

	err := config.ValidateConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid configuration")
	} else {
		log.Debug().Msg("Configuration is valid.")
	}

	http.HandleFunc(path, api.WebhookHandler)

	address := getEnv(EnvServerAddress, "127.0.0.1")
	port := getEnv(EnvServerPort, "42135")

	startHTTPServer(address, port)
}
