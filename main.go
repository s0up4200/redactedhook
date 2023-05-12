package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

const (
	APIEndpointBase = "https://redacted.ch/ajax.php"
	Pathhook        = "/redacted/hook"
)

// Rate limit requests to max 7 requests per 10 seconds
var limiter = rate.NewLimiter(rate.Every(1*time.Second), 10)

type RequestData struct {
	UserID      int     `json:"user_id,omitempty"`
	TorrentID   int     `json:"torrent_id,omitempty"`
	APIKey      string  `json:"apikey"`
	MinRatio    float64 `json:"minratio,omitempty"`
	Uploaders   string  `json:"uploaders,omitempty"`
	RecordLabel string  `json:"record_labels,omitempty"`
	Mode        string  `json:"mode,omitempty"`
}

type ResponseData struct {
	Status   string `json:"status"`
	Error    string `json:"error"`
	Response struct {
		Username string `json:"username"`
		Stats    struct {
			Ratio float64 `json:"ratio"`
		} `json:"stats"`
		Torrent struct {
			Username    string `json:"username"`
			RecordLabel string `json:"remasterRecordLabel"`
			//CatalogueNumber string `json:"remasterCatalogueNumber"`
		} `json:"torrent"`
	} `json:"response"`
}

func hookAPIResponse(resp *http.Response) error {
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		log.Warn().Msgf("Unexpected content type: %s, likely wrong API key", contentType)
		return fmt.Errorf("unexpected content type: %s", contentType)
	}
	return nil
}

func fetchTorrentData(torrentID int, apiKey string) (*ResponseData, error) {

	if !limiter.Allow() {
		log.Warn().Msg("Too many requests (fetchTorrentData)")
		return nil, fmt.Errorf("too many requests")
	}

	endpoint := fmt.Sprintf("%s?action=torrent&id=%d", APIEndpointBase, torrentID)
	req, err := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("Authorization", apiKey)

	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = hookAPIResponse(resp)
	if err != nil {
		return nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responseData ResponseData
	err = json.Unmarshal(respBody, &responseData)
	if err != nil {
		return nil, err
	}

	return &responseData, nil
}

func fetchUserData(userID int, apiKey string) (*ResponseData, error) {

	if !limiter.Allow() {
		log.Warn().Msg("Too many requests (fetchUserData)")
		return nil, fmt.Errorf("too many requests")
	}

	endpoint := fmt.Sprintf("%s?action=user&id=%d", APIEndpointBase, userID)
	req, err := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("Authorization", apiKey)

	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = hookAPIResponse(resp)
	if err != nil {
		return nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responseData ResponseData
	err = json.Unmarshal(respBody, &responseData)
	if err != nil {
		return nil, err
	}

	return &responseData, nil
}

func hookData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is supported", http.StatusBadRequest)
		return
	}

	var torrentData *ResponseData
	var userData *ResponseData

	// Log request received
	log.Info().Msgf("Received data request from %s", r.RemoteAddr)

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

	reqHeader := make(http.Header)
	reqHeader.Set("Authorization", requestData.APIKey)

	// hook ratio
	if requestData.UserID != 0 && requestData.MinRatio != 0 {

		if userData == nil {
			userData, err = fetchUserData(requestData.UserID, requestData.APIKey)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		ratio := userData.Response.Stats.Ratio
		minRatio := requestData.MinRatio
		username := userData.Response.Username

		log.Debug().Msgf("MinRatio set to %.2f for %s", minRatio, username)

		if ratio < minRatio {
			w.WriteHeader(http.StatusIMUsed) // HTTP status code 226
			log.Debug().Msgf("Returned ratio %.2f is below minratio %.2f for %s, responding with status 226", ratio, minRatio, username)
			return
		}
	}

	// hook uploader
	if requestData.TorrentID != 0 && requestData.Uploaders != "" {
		if torrentData == nil {
			torrentData, err = fetchTorrentData(requestData.TorrentID, requestData.APIKey)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		username := torrentData.Response.Torrent.Username
		usernames := strings.Split(requestData.Uploaders, ",")

		log.Debug().Msgf("Requested uploaders [%s]: %s", requestData.Mode, usernames)

		isListed := false
		for _, uname := range usernames {
			if uname == username {
				isListed = true
				break
			}
		}

		if (requestData.Mode == "blacklist" && isListed) || (requestData.Mode == "whitelist" && !isListed) {
			w.WriteHeader(http.StatusIMUsed + 1) // HTTP status code 227
			log.Debug().Msgf("Uploader (%s) is not allowed, responding with status 227", username)
			return
		}
	}

	// hook record label
	if requestData.TorrentID != 0 && requestData.RecordLabel != "" {
		if torrentData == nil {
			torrentData, err = fetchTorrentData(requestData.TorrentID, requestData.APIKey)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		recordLabel := torrentData.Response.Torrent.RecordLabel
		requestedRecordLabels := strings.Split(requestData.RecordLabel, ",")
		log.Debug().Msgf("Requested record labels: %v", requestedRecordLabels)

		isRecordLabelPresent := false
		for _, rLabel := range requestedRecordLabels {
			if rLabel == recordLabel {
				isRecordLabelPresent = true
				break
			}
		}

		if !isRecordLabelPresent {
			w.WriteHeader(http.StatusIMUsed + 2) // HTTP status code 228
			log.Debug().Msgf("Record label (%s) is not in the requested record labels (%v), responding with status 228", recordLabel, requestedRecordLabels)
			return
		}
	}

	w.WriteHeader(http.StatusOK) // HTTP status code 200
	log.Debug().Msg("Conditions met, responding with status 200")
}

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05", NoColor: false})

	http.HandleFunc(Pathhook, hookData)

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
