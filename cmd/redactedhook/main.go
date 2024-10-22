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
	"syscall"
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
	path              = "/hook"
	tokenLength       = 16
	shutdownTimeout   = 10 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 120 * time.Second
	readHeaderTimeout = 5 * time.Second
	defaultConfigPath = "config.toml"
	envPrefix         = "REDACTEDHOOK__"
)

func generateAPIToken() (string, error) {
	b := make([]byte, tokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}
	apiKey := hex.EncodeToString(b)
	fmt.Fprintf(os.Stdout, "API Token: %v, copy and paste into your config.toml\n", apiKey)
	return apiKey, nil
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
	flag.StringVar(&configPath, "config", defaultConfigPath, "Path to the configuration file")
	flag.Parse()

	if len(flag.Args()) > 0 {
		switch flag.Arg(0) {
		case "generate-apitoken":
			apiToken, err := generateAPIToken()
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to generate API token")
			}
			_ = apiToken // Used in output
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
	if value, exists := os.LookupEnv(envPrefix + key); exists {
		return value
	}
	return defaultValue
}

func initLogger() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"})
}

func createServer(address string) *http.Server {
	return &http.Server{
		Addr:              address,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
	}
}

func startHTTPServer(ctx context.Context, address string) error {
	server := createServer(address)

	// Create error channel to capture server errors
	serverError := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverError <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	log.Info().Msgf("Starting server on %s", address)
	log.Info().Msgf("Version: %s, Commit: %s, Build Date: %s", version, commit, buildDate)

	// Handle shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverError:
		return err
	case <-shutdown:
		log.Info().Msg("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
		log.Info().Msg("Server shutdown completed")
	}

	return nil
}

func loadEnvironmentConfig() {
	config.GetConfig().Authorization.APIToken = getEnv("API_TOKEN", config.GetConfig().Authorization.APIToken)
	config.GetConfig().IndexerKeys.REDKey = getEnv("RED_APIKEY", config.GetConfig().IndexerKeys.REDKey)
	config.GetConfig().IndexerKeys.OPSKey = getEnv("OPS_APIKEY", config.GetConfig().IndexerKeys.OPSKey)
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

	loadEnvironmentConfig()

	http.HandleFunc(path, api.WebhookHandler)

	host := getEnv("HOST", config.GetConfig().Server.Host)
	port := getEnv("PORT", fmt.Sprintf("%d", config.GetConfig().Server.Port))
	address := fmt.Sprintf("%s:%s", host, port)

	// Create a root context for the application
	ctx := context.Background()

	if err := startHTTPServer(ctx, address); err != nil {
		log.Fatal().Err(err).Msg("Server error")
	}
}
