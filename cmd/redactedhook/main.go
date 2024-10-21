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
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

const (
	path        = "/hook"
	tokenLength = 16
)

func generateAPIToken() string {
	b := make([]byte, tokenLength)
	if _, err := rand.Read(b); err != nil {
		log.Fatal().Err(err).Msg("Failed to generate API key")
	}
	apiKey := hex.EncodeToString(b)
	fmt.Fprintf(os.Stdout, "API Token: %v, copy and paste into your config.toml\n", apiKey)
	return apiKey
}

func printHelp() {
	fmt.Println("Usage: redactedhook [options] [command]")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  generate-apitoken  Generate a new API token and print it.")
	fmt.Println("  create-config      Create a default configuration file.")
	fmt.Println("  help               Display this help message.")
}

func parseFlags() (string, bool) {
	var configPath string
	flag.StringVar(&configPath, "config", "config.toml", "Path to the configuration file")
	flag.Parse()

	if len(flag.Args()) > 0 {
		switch flag.Arg(0) {
		case "generate-apitoken":
			generateAPIToken()
			return "", true
		case "create-config":
			config.CreateConfigFile()
			return "", true
		case "help":
			printHelp()
			return "", true
		default:
			log.Fatal().Msgf("Unknown command: %s. Use 'redactedhook help' to see available commands.", flag.Arg(0))
		}
	}
	return configPath, false
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func initLogger() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"})
}

func startHTTPServer(address string) {
	server := &http.Server{Addr: address}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server crashed")
		}
	}()

	log.Info().Msgf("Starting server on %s", address)
	log.Info().Msgf("Version: %s, Commit: %s, Build Date: %s", version, commit, buildDate)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server shutdown failed")
	}
}

func main() {
	initLogger()

	configPath, isCommandExecuted := parseFlags()
	if isCommandExecuted {
		return
	}

	config.InitConfig(configPath)

	if err := config.ValidateConfig(); err != nil {
		log.Fatal().Err(err).Msg("Invalid configuration")
	}

	http.HandleFunc(path, api.WebhookHandler)

	host := getEnv("REDACTEDHOOK__HOST", config.GetConfig().Server.Host)
	port := getEnv("REDACTEDHOOK__PORT", fmt.Sprintf("%d", config.GetConfig().Server.Port))
	apiToken := getEnv("REDACTEDHOOK__API_TOKEN", config.GetConfig().Authorization.APIToken)
	redApiKey := getEnv("REDACTEDHOOK__RED_APIKEY", config.GetConfig().IndexerKeys.REDKey)
	opsApiKey := getEnv("REDACTEDHOOK__OPS_APIKEY", config.GetConfig().IndexerKeys.OPSKey)

	config.GetConfig().Authorization.APIToken = apiToken
	config.GetConfig().IndexerKeys.REDKey = redApiKey
	config.GetConfig().IndexerKeys.OPSKey = opsApiKey

	address := fmt.Sprintf("%s:%s", host, port)

	startHTTPServer(address)
}
