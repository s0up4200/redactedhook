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

const (
	APIEndpointBase = "https://redacted.ch/ajax.php"
	PathRatio       = "/redacted/ratio"
	PathUploader    = "/redacted/uploader"
)

type RequestData struct {
	UserID    int     `json:"user_id"`
	TorrentID int     `json:"torrent_id"`
	APIKey    string  `json:"apikey"`
	MinRatio  float64 `json:"minratio"`
	Uploaders string  `json:"uploaders"`
}

type ResponseData struct {
	Status   string `json:"status"`
	Error    string `json:"error"`
	Response struct {
		Stats struct {
			Ratio float64 `json:"ratio"`
		} `json:"stats"`
		Torrent struct {
			Username string `json:"username"`
		} `json:"torrent"`
	} `json:"response"`
}

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05", NoColor: false})

	http.HandleFunc(PathRatio, checkRatio)
	http.HandleFunc(PathUploader, checkUploader)

	address := os.Getenv("SERVER_ADDRESS")
	if address == "" {
		address = "127.0.0.1"
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "42135"
	}

	// Start the server
	serverAddr := address + ":" + port
	log.Info().Msg("Starting server on " + serverAddr)
	err := http.ListenAndServe(serverAddr, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}

func checkAPIResponse(resp *http.Response) error {
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return fmt.Errorf("unexpected content type: %s", contentType)
	}
	return nil
}

func checkRatio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is supported", http.StatusBadRequest)
		return
	}

	// Log request received
	log.Info().Msgf("Received ratio request from %s", r.RemoteAddr)

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

	endpoint := fmt.Sprintf("%s?action=user&id=%d", APIEndpointBase, requestData.UserID)

	client := &http.Client{}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", requestData.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	err = checkAPIResponse(resp)
	if err != nil {
		log.Error().Msgf("API response indicates maintenance or unexpected content: %s", err.Error())
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

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
		log.Error().Msgf("JSON response indicates a failure: %s", responseData.Error)
		http.Error(w, responseData.Error, http.StatusBadRequest)
		return
	}

	ratio := responseData.Response.Stats.Ratio
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
	log.Info().Msgf("Received uploader request from %s", r.RemoteAddr)

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

	if requestData.Uploaders == "" {
		log.Error().Msg("Uploaders list is empty")
		http.Error(w, "Uploaders list is empty", http.StatusBadRequest)
		return
	}

	endpoint := fmt.Sprintf("%s?action=torrent&id=%d", APIEndpointBase, requestData.TorrentID)

	client := &http.Client{}
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

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	err = checkAPIResponse(resp)
	if err != nil {
		log.Error().Msgf("API response indicates maintenance or unexpected content: %s", err.Error())
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

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
		log.Error().Msgf("JSON response indicates a failure: %s", responseData.Error)
		http.Error(w, responseData.Error, http.StatusBadRequest)
		return
	}

	username := responseData.Response.Torrent.Username
	usernames := strings.Split(requestData.Uploaders, ",")

	log.Debug().Msgf("Found uploader: %s", username) // Print the uploader's username

	for _, uname := range usernames {
		if uname == username {
			w.WriteHeader(http.StatusIMUsed + 1) // HTTP status code 227
			log.Debug().Msgf("Uploader (%s) is blacklisted, responding with status 227", username)
			return
		}
	}

	w.WriteHeader(http.StatusOK) // HTTP status code 200
	log.Debug().Msg("Uploader not in blacklist, responding with status 200")
}
