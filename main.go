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
	APIEndpointBaseRedacted = "https://redacted.ch/ajax.php"
	APIEndpointBaseOrpheus  = "https://orpheus.network/ajax.php"
	Pathhook                = "/hook"
)

var redactedLimiter = rate.NewLimiter(rate.Every(1*time.Second), 1)
var orpheusLimiter = rate.NewLimiter(rate.Every(10*time.Second), 5)

//var (
//	version = "dev"
//	commit  = "none"
//)

type RequestData struct {
	REDUserID   int     `json:"red_user_id,omitempty"`
	OPSUserID   int     `json:"ops_user_id,omitempty"`
	TorrentID   int     `json:"torrent_id,omitempty"`
	REDKey      string  `json:"red_apikey,omitempty"`
	OPSKey      string  `json:"ops_apikey,omitempty"`
	MinRatio    float64 `json:"minratio,omitempty"`
	MinSize     int64   `json:"minsize,omitempty"`
	MaxSize     int64   `json:"maxsize,omitempty"`
	Uploaders   string  `json:"uploaders,omitempty"`
	RecordLabel string  `json:"record_labels,omitempty"`
	Mode        string  `json:"mode,omitempty"`
	Indexer     string  `json:"indexer"`
}

type ResponseData struct {
	Status   string `json:"status"`
	Error    string `json:"error"`
	Response struct {
		Username string `json:"username"`
		Stats    struct {
			Ratio float64 `json:"ratio"`
		} `json:"stats"`
		Group struct {
			Name string `json:"name"`
		} `json:"group"`
		Torrent struct {
			Username        string `json:"username"`
			Size            int64  `json:"size"`
			RecordLabel     string `json:"remasterRecordLabel"`
			ReleaseName     string `json:"filePath"`
			CatalogueNumber string `json:"remasterCatalogueNumber"`
		} `json:"torrent"`
	} `json:"response"`
}

func fetchTorrentData(torrentID int, apiKey string, apiBase string, indexer string) (*ResponseData, error) {

	// Determine the correct limiter based on the indexer
	var limiter *rate.Limiter
	switch indexer {
	case "redacted":
		limiter = redactedLimiter
	case "ops":
		limiter = orpheusLimiter
	default:
		// Return an error instead of using http.Error
		return nil, fmt.Errorf("invalid indexer")
	}

	// Use the limiter
	if !limiter.Allow() {
		log.Warn().Msgf(("%s: Too many requests (fetchTorrentData)"), indexer)
		return nil, fmt.Errorf("too many requests")
	}

	endpoint := fmt.Sprintf("%s?action=torrent&id=%d", apiBase, torrentID)
	req, err := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("Authorization", apiKey)

	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responseData ResponseData
	err = json.Unmarshal(respBody, &responseData)
	if err != nil {
		return nil, err
	}

	if responseData.Status != "success" {
		var sourceName string
		if indexer == "redacted" {
			sourceName = "RED"
		} else if indexer == "ops" {
			sourceName = "OPS"
		}
		log.Warn().Msgf("Received API response from %s with status '%s' and error message: '%s'", sourceName, responseData.Status, responseData.Error)
		return nil, fmt.Errorf("API error from %s: %s", sourceName, responseData.Error)
	}

	return &responseData, nil
}

func fetchUserData(userID int, apiKey string, indexer string, apiBase string) (*ResponseData, error) {
	// Determine the correct limiter based on the indexer
	var limiter *rate.Limiter
	switch indexer {
	case "redacted":
		limiter = redactedLimiter
	case "ops":
		limiter = orpheusLimiter
	default:
		return nil, fmt.Errorf("invalid indexer")
	}

	// Use the limiter
	if !limiter.Allow() {
		log.Warn().Msgf("%s: Too many requests (fetchUserData)", indexer)
		return nil, fmt.Errorf("too many requests")
	}

	endpoint := fmt.Sprintf("%s?action=user&id=%d", apiBase, userID)
	req, err := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("Authorization", apiKey)

	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responseData ResponseData
	err = json.Unmarshal(respBody, &responseData)
	if err != nil {
		return nil, err
	}

	if responseData.Status != "success" {
		var sourceName string
		if indexer == "redacted" {
			sourceName = "RED"
		} else if indexer == "ops" {
			sourceName = "OPS"
		}
		log.Warn().Msgf("Received API response from %s with status '%s' and error message: '%s'", sourceName, responseData.Status, responseData.Error)
		return nil, fmt.Errorf("API error from %s: %s", sourceName, responseData.Error)
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
	// Determine the appropriate API base based on the requested hook path
	var apiBase string
	switch requestData.Indexer {
	case "redacted":
		apiBase = APIEndpointBaseRedacted
	case "ops":
		apiBase = APIEndpointBaseOrpheus
	default:
		http.Error(w, "Invalid path", http.StatusNotFound)
		return
	}

	reqHeader := make(http.Header)
	var apiKey string
	if requestData.Indexer == "redacted" {
		apiKey = requestData.REDKey
	} else if requestData.Indexer == "ops" {
		apiKey = requestData.OPSKey
	}
	reqHeader.Set("Authorization", apiKey)

	// hook ratio
	if requestData.MinRatio != 0 {
		var userID int
		var apiKey string
		if requestData.Indexer == "redacted" {
			if requestData.REDUserID == 0 {
				log.Debug().Msg("red_user_id is missing but required when minratio is set for 'redacted'")
				http.Error(w, "red_user_id is required for 'redacted' when minratio is set", http.StatusBadRequest)
				return
			}
			userID = requestData.REDUserID
			apiKey = requestData.REDKey
			log.Debug().Msgf("MinRatio check for Redacted with user ID: %d", userID)
		} else if requestData.Indexer == "ops" {
			if requestData.OPSUserID == 0 {
				log.Debug().Msg("ops_user_id is missing but required when minratio is set for 'ops'")
				http.Error(w, "ops_user_id is required for 'ops' when minratio is set", http.StatusBadRequest)
				return
			}
			userID = requestData.OPSUserID
			apiKey = requestData.OPSKey
			log.Debug().Msgf("MinRatio check for OPS with user ID: %d", userID)
		}

		if userID != 0 {
			if userData == nil {
				userData, err = fetchUserData(userID, apiKey, requestData.Indexer, apiBase)
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
	}

	// hook uploader
	if requestData.TorrentID != 0 && requestData.Uploaders != "" {
		if torrentData == nil {
			var apiKey string
			if requestData.Indexer == "redacted" {
				apiKey = requestData.REDKey
			} else if requestData.Indexer == "ops" {
				apiKey = requestData.OPSKey
			}
			torrentData, err = fetchTorrentData(requestData.TorrentID, apiKey, apiBase, requestData.Indexer)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		//name := torrentData.Response.Group.Name
		//releaseName := torrentData.Response.Torrent.ReleaseName
		//TorrentID := requestData.TorrentID
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
			var apiKey string
			if requestData.Indexer == "redacted" {
				apiKey = requestData.REDKey
			} else if requestData.Indexer == "ops" {
				apiKey = requestData.OPSKey
			}
			torrentData, err = fetchTorrentData(requestData.TorrentID, apiKey, apiBase, requestData.Indexer)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		recordLabel := torrentData.Response.Torrent.RecordLabel
		catalogueNumber := torrentData.Response.Torrent.CatalogueNumber
		name := torrentData.Response.Group.Name
		//releaseName := torrentData.Response.Torrent.ReleaseName
		TorrentID := requestData.TorrentID
		requestedRecordLabels := strings.Split(requestData.RecordLabel, ",")

		var labelAndCatalogue string

		if recordLabel == "" && catalogueNumber == "" {
			labelAndCatalogue = ""
		} else if recordLabel == "" {
			labelAndCatalogue = fmt.Sprintf(" (Cat#: %s)", catalogueNumber)
		} else if catalogueNumber == "" {
			labelAndCatalogue = fmt.Sprintf(" (Label: %s)", recordLabel)
		} else {
			labelAndCatalogue = fmt.Sprintf(" (Label: %s - Cat#: %s)", recordLabel, catalogueNumber)
		}

		log.Debug().Msgf("Checking release: %s%s (TorrentID: %d)", name, labelAndCatalogue, TorrentID)

		if recordLabel == "" {
			log.Debug().Msgf("No record label found for release: %s. Responding with status code 228.", name)
			w.WriteHeader(http.StatusIMUsed + 2) // HTTP status code 228
			return
		}

		//log.Debug().Msgf("Requested record labels: %v", requestedRecordLabels)

		isRecordLabelPresent := false
		for _, rLabel := range requestedRecordLabels {
			if rLabel == recordLabel {
				isRecordLabelPresent = true
				break
			}
		}

		if !isRecordLabelPresent {
			w.WriteHeader(http.StatusIMUsed + 2) // HTTP status code 228
			log.Debug().Msgf("The record label '%s' is not included in the requested record labels: %v. Responding with status code 228.", recordLabel, requestedRecordLabels)
			return
		}
	}

	// hook size
	if requestData.TorrentID != 0 && (requestData.MinSize != 0 || requestData.MaxSize != 0) {
		if torrentData == nil {
			var apiKey string
			if requestData.Indexer == "redacted" {
				apiKey = requestData.REDKey
			} else if requestData.Indexer == "ops" {
				apiKey = requestData.OPSKey
			}
			torrentData, err = fetchTorrentData(requestData.TorrentID, apiKey, apiBase, requestData.Indexer)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		torrentSize := torrentData.Response.Torrent.Size

		log.Debug().Msgf("Torrent size: %d", torrentSize)
		log.Debug().Msgf("Requested min size: %d", requestData.MinSize)
		log.Debug().Msgf("Requested max size: %d", requestData.MaxSize)

		if (requestData.MinSize != 0 && torrentSize < requestData.MinSize) ||
			(requestData.MaxSize != 0 && torrentSize > requestData.MaxSize) {
			w.WriteHeader(http.StatusIMUsed + 3) // HTTP status code 229
			log.Debug().Msgf("Torrent size %d is outside the requested size range (%d to %d), responding with status 229", torrentSize, requestData.MinSize, requestData.MaxSize)
			return
		}
	}

	w.WriteHeader(http.StatusOK) // HTTP status code 200
	log.Debug().Msg("Conditions met, responding with status 200")
}

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05", NoColor: false})

	//log.Info().Msgf("RedactedHook version %s, commit %s", version, commit[:7])

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
