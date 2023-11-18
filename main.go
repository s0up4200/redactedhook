package main

import (
	"flag"
	"net/http"
	"os"

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

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05", NoColor: false})
	var configPath string

	flag.StringVar(&configPath, "config", "", "Path to the configuration file")
	flag.Parse()

	config.InitConfig(configPath)

	http.HandleFunc(api.Pathhook, api.HookData)

	address := os.Getenv("SERVER_ADDRESS")
	if address == "" {
		address = "127.0.0.1"
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "42135"
	}

	serverAddr := address + ":" + port
	log.Info().Msgf("Starting server on %s", serverAddr)
	log.Info().Msgf("Version: %s, Commit: %s, Build Date: %s", version, commit, buildDate)
	err := http.ListenAndServe(serverAddr, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
