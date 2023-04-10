package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type RequestData struct {
	ID        int     `json:"id"`
	APIKey    string  `json:"apikey"`
	MinRatio  float64 `json:"minratio,omitempty"`
	TorrentID int     `json:"torrent_id,omitempty"`
	Usernames string  `json:"uploaders,omitempty"`
}

type ResponseData struct {
	Status          string `json:"status"`
	Error           string `json:"error"`
	TorrentUploader struct {
		Username string `json:"username"`
	} `json:"torrentUploader"`
	Stats struct {
		Ratio float64 `json:"ratio"`
	} `json:"stats"`
}

var httpClient = &http.Client{}

func main() {
	// Configure zerolog to use colored output
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05", NoColor: false})

	http.HandleFunc("/redacted/ratio", checkRatio)
	http.HandleFunc("/redacted/uploader", checkUploader)
	log.Info().Msg("Starting server on 127.0.0.1:42135")
	err := http.ListenAndServe("127.0.0.1:42135", nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}

func checkRatio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is supported", http.StatusBadRequest)
		return
	}

	// Log request received
	log.Debug().Msgf("Received request from %s", r.RemoteAddr)

	// Read JSON payload from the request body
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var requestData RequestData
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		log.Debug().Msgf("Failed to unmarshal JSON payload: %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	endpoint := fmt.Sprintf("https://redacted.ch/ajax.php?action=user&id=%d", requestData.ID)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", requestData.APIKey)

	if requestData.APIKey == "" {
		log.Error().Msg("API key is empty")
		http.Error(w, "API key is empty", http.StatusBadRequest)
		return
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var responseData ResponseData
	err = json.Unmarshal(respBody, &responseData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check for a "failure" status in the JSON response
	if responseData.Status == "failure" {
		log.Error().Msgf("JSON response indicates a failure: %s\n", responseData.Error)
		http.Error(w, responseData.Error, http.StatusBadRequest)
		return
	}

	ratio := responseData.Stats.Ratio
	minRatio := requestData.MinRatio

	if ratio < minRatio {
		w.WriteHeader(http.StatusIMUsed) // HTTP status code 226
		log.Debug().Msgf("Returned ratio (%f) is below minratio (%f), responding with status 226", ratio, minRatio)
	} else {
		w.WriteHeader(http.StatusOK) // HTTP status code 200
		log.Debug().Msgf("Returned ratio (%f) is equal to or above minratio (%f), responding with status 200", ratio, minRatio)
	}
}

func checkUploader(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Debug().Msg("Non-POST method received")
		http.Error(w, "Only POST method is supported", http.StatusBadRequest)
		return
	}

	// Log request received
	log.Debug().Msgf("Received request from %s", r.RemoteAddr)

	// Read JSON payload from the request body
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var requestData RequestData
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		log.Debug().Msgf("Failed to unmarshal JSON payload: %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Make the request to the API
	endpoint := fmt.Sprintf("https://redacted.ch/ajax.php?action=torrent&id=%d", requestData.ID)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", requestData.APIKey)

	if requestData.APIKey == "" {
		log.Error().Msg("API key is empty")
		http.Error(w, "API key is empty", http.StatusBadRequest)
		return
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Unmarshal the response JSON into the appropriate struct
	var responseData ResponseData
	err = json.Unmarshal(respBody, &responseData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check for a "failure" status in the JSON response
	if responseData.Status == "failure" {
		log.Error().Msgf("JSON response indicates a failure: %s\n", responseData.Error)
		http.Error(w, responseData.Error, http.StatusBadRequest)
		return
	}

	// Check if the uploader is in the blacklist
	username := responseData.TorrentUploader.Username
	usernames := strings.Split(requestData.Usernames, ",")

	log.Debug().Msgf("Found uploader: %s", username) // Print the uploader's username

	for _, uname := range usernames {
		if uname == username {
			w.WriteHeader(http.StatusIMUsed + 1) // HTTP status code 226
			log.Debug().Msgf("Uploader (%s) is blacklisted, responding with status 226", username)
			return
		}
	}

	// If uploader is not in the blacklist, respond with HTTP status 200
	w.WriteHeader(http.StatusOK) // HTTP status code 200
	log.Debug().Msg("Uploader not in blacklist, responding with status 200")
}
