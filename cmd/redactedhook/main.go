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
	healthPath        = "/healthz"
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
	fmt.Println("  health             Perform a health check on the service.")
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
		case "health":
			performHealthCheck()
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

func hasRequiredEnvVars() bool {
	// Check for essential environment variables
	essentialVars := []string{
		"API_TOKEN",
		"RED_APIKEY",
		"OPS_APIKEY",
	}

	for _, v := range essentialVars {
		if _, exists := os.LookupEnv(envPrefix + v); !exists {
			return false
		}
	}
	return true
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
	// Server settings
	config.GetConfig().Server.Host = getEnv("HOST", config.GetConfig().Server.Host)
	if port := os.Getenv(envPrefix + "PORT"); port != "" {
		if val, err := fmt.Sscanf(port, "%d", &config.GetConfig().Server.Port); err != nil || val != 1 {
			log.Warn().Msgf("Invalid PORT value: %s", port)
		}
	}

	// Authorization settings
	config.GetConfig().Authorization.APIToken = getEnv("API_TOKEN", config.GetConfig().Authorization.APIToken)
	config.GetConfig().IndexerKeys.REDKey = getEnv("RED_APIKEY", config.GetConfig().IndexerKeys.REDKey)
	config.GetConfig().IndexerKeys.OPSKey = getEnv("OPS_APIKEY", config.GetConfig().IndexerKeys.OPSKey)

	// Logs settings
	config.GetConfig().Logs.LogLevel = getEnv("LOGS_LOGLEVEL", config.GetConfig().Logs.LogLevel)
	config.GetConfig().Logs.LogToFile = os.Getenv(envPrefix+"LOGS_LOGTOFILE") == "true"
	config.GetConfig().Logs.LogFilePath = getEnv("LOGS_LOGFILEPATH", config.GetConfig().Logs.LogFilePath)

	if maxSize := os.Getenv(envPrefix + "LOGS_MAXSIZE"); maxSize != "" {
		if val, err := fmt.Sscanf(maxSize, "%d", &config.GetConfig().Logs.MaxSize); err != nil || val != 1 {
			log.Warn().Msgf("Invalid LOGS_MAXSIZE value: %s", maxSize)
		}
	}
	if maxBackups := os.Getenv(envPrefix + "LOGS_MAXBACKUPS"); maxBackups != "" {
		if val, err := fmt.Sscanf(maxBackups, "%d", &config.GetConfig().Logs.MaxBackups); err != nil || val != 1 {
			log.Warn().Msgf("Invalid LOGS_MAXBACKUPS value: %s", maxBackups)
		}
	}
	if maxAge := os.Getenv(envPrefix + "LOGS_MAXAGE"); maxAge != "" {
		if val, err := fmt.Sscanf(maxAge, "%d", &config.GetConfig().Logs.MaxAge); err != nil || val != 1 {
			log.Warn().Msgf("Invalid LOGS_MAXAGE value: %s", maxAge)
		}
	}
	config.GetConfig().Logs.Compress = os.Getenv(envPrefix+"LOGS_COMPRESS") == "true"
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().
		Str("method", r.Method).
		Str("remote_addr", r.RemoteAddr).
		Str("user_agent", r.UserAgent()).
		Msg("Health check request received")

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		log.Error().Err(err).Msg("Failed to write health check response")
	}
}

func performHealthCheck() {
	resp, err := http.Get("http://localhost:42135" + healthPath)
	if err != nil {
		fmt.Println("Unhealthy")
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("Healthy")
		os.Exit(0)
	} else {
		fmt.Println("Unhealthy")
		os.Exit(1)
	}
}

func main() {
	initLogger()

	configPath, isCommandExecuted := parseFlags()
	if isCommandExecuted {
		return
	}

	// Initialize with default values
	config.GetConfig().Server.Host = "127.0.0.1"
	config.GetConfig().Server.Port = 42135
	config.GetConfig().Logs.LogLevel = "info"
	config.GetConfig().Logs.MaxSize = 100 // 100MB
	config.GetConfig().Logs.MaxBackups = 3
	config.GetConfig().Logs.MaxAge = 28 // 28 days
	config.GetConfig().Logs.LogFilePath = "redactedhook.log"

	// Try to load config file if it exists
	configFileExists := false
	if _, err := os.Stat(configPath); err == nil {
		config.InitConfig(configPath)
		configFileExists = true
	}

	// If no config file and no environment variables, exit
	if !configFileExists && !hasRequiredEnvVars() {
		log.Fatal().Msg("No config file found and required environment variables are not set. Please provide either a config file or set the required environment variables (REDACTEDHOOK__API_TOKEN, REDACTEDHOOK__RED_APIKEY, REDACTEDHOOK__OPS_APIKEY)")
	}

	// Load environment variables (these will override config file values if present)
	loadEnvironmentConfig()

	// Validate the final configuration
	if err := config.ValidateConfig(); err != nil {
		log.Fatal().Err(err).Msg("Invalid configuration")
	}

	http.HandleFunc(path, api.WebhookHandler)
	http.HandleFunc(healthPath, healthHandler)

	address := fmt.Sprintf("%s:%d", config.GetConfig().Server.Host, config.GetConfig().Server.Port)

	// Create a root context for the application
	ctx := context.Background()

	if err := startHTTPServer(ctx, address); err != nil {
		log.Fatal().Err(err).Msg("Server error")
	}
}
