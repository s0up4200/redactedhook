package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/inhies/go-bytesize"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

//var (
//	version = "dev"
//	commit  = "none"
//)

const (
	APIEndpointBaseRedacted = "https://redacted.ch/ajax.php"
	APIEndpointBaseOrpheus  = "https://orpheus.network/ajax.php"
	Pathhook                = "/hook"
)

var (
	redactedLimiter = rate.NewLimiter(rate.Every(10*time.Second), 10)
	orpheusLimiter  = rate.NewLimiter(rate.Every(10*time.Second), 5)
)

type RequestData struct {
	REDUserID   int               `json:"red_user_id,omitempty"`
	OPSUserID   int               `json:"ops_user_id,omitempty"`
	TorrentID   int               `json:"torrent_id,omitempty"`
	REDKey      string            `json:"red_apikey,omitempty"`
	OPSKey      string            `json:"ops_apikey,omitempty"`
	MinRatio    float64           `json:"minratio,omitempty"`
	MinSize     bytesize.ByteSize `json:"minsize,omitempty"`
	MaxSize     bytesize.ByteSize `json:"maxsize,omitempty"`
	Uploaders   string            `json:"uploaders,omitempty"`
	RecordLabel string            `json:"record_labels,omitempty"`
	Mode        string            `json:"mode,omitempty"`
	Indexer     string            `json:"indexer"`
	TorrentName string            `json:"torrentname,omitempty"`
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
		Torrent *struct {
			Username        string `json:"username"`
			Size            int64  `json:"size"`
			RecordLabel     string `json:"remasterRecordLabel"`
			ReleaseName     string `json:"filePath"`
			CatalogueNumber string `json:"remasterCatalogueNumber"`
		} `json:"torrent"`
	} `json:"response"`
}

type APIClient struct {
	RedactedLimiter *rate.Limiter
	OrpheusLimiter  *rate.Limiter
	Client          *http.Client
	Config          *Config
}

func NewAPIClient(config *Config) *APIClient {
	return &APIClient{
		RedactedLimiter: redactedLimiter,
		OrpheusLimiter:  orpheusLimiter,
		Client: &http.Client{
			Timeout: time.Second * 10,
		},
		Config: config,
	}
}

func (api *APIClient) fetchAPIData(action string, id int, apiKey string, indexer string) (*ResponseData, error) {
	limiter, apiBase, sourceName := api.getLimiterAndBase(indexer)

	if !limiter.Allow() {
		log.Warn().Msgf("%s: Too many requests (%s)", indexer, action)
		return nil, fmt.Errorf("too many requests")
	}

	endpoint := fmt.Sprintf("%s?action=%s&id=%d", apiBase, action, id)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", apiKey)

	resp, err := api.Client.Do(req)
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
		log.Warn().Msgf("Received API response from %s with status '%s' and error message: '%s'", sourceName, responseData.Status, responseData.Error)
		return nil, fmt.Errorf(responseData.Error)
	}

	return &responseData, nil
}

func (api *APIClient) getLimiterAndBase(indexer string) (*rate.Limiter, string, string) {
	switch indexer {
	case "redacted":
		return api.RedactedLimiter, APIEndpointBaseRedacted, "RED"
	case "ops":
		return api.OrpheusLimiter, APIEndpointBaseOrpheus, "OPS"
	default:
		return nil, "", ""
	}
}

func (api *APIClient) hookData(w http.ResponseWriter, r *http.Request, config *Config) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is supported", http.StatusBadRequest)
		return
	}

	var data *ResponseData
	var requestData RequestData

	// Read JSON payload from the request body
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &requestData)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to unmarshal JSON payload")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Log request received
	logMsg := fmt.Sprintf("Received data request from %s", r.RemoteAddr)
	if requestData.TorrentName != "" {
		logMsg += fmt.Sprintf(" - TorrentName: %s", requestData.TorrentName)
	}
	log.Info().Msg(logMsg)

	reqHeader := make(http.Header)
	var apiKey string
	if requestData.Indexer == "redacted" {
		apiKey = requestData.REDKey
	} else if requestData.Indexer == "ops" {
		apiKey = requestData.OPSKey
	}
	reqHeader.Set("Authorization", apiKey)

	// Fallback to config values if not present in the webhook
	if apiKey == "" {
		if requestData.Indexer == "redacted" {
			apiKey = config.APIKeys.REDKey
		} else if requestData.Indexer == "ops" {
			apiKey = config.APIKeys.OPSKey
		}
	}

	if requestData.MinRatio == 0 {
		requestData.MinRatio = config.MinRatio
	}

	if requestData.REDUserID == 0 {
		requestData.REDUserID = config.UserID.REDUserID
	}

	if requestData.OPSUserID == 0 {
		requestData.OPSUserID = config.UserID.OPSUserID
	}

	if requestData.TorrentID != 0 {
		var action string = "torrent"
		if data == nil || data.Response.Torrent == nil {
			data, err = api.fetchAPIData(action, requestData.TorrentID, apiKey, requestData.Indexer)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		recordLabel := data.Response.Torrent.RecordLabel
		catalogueNumber := data.Response.Torrent.CatalogueNumber
		name := data.Response.Group.Name

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

		log.Debug().Msgf("Checking release: %s%s (TorrentID: %d)", name, labelAndCatalogue, requestData.TorrentID)
	}

	// hook ratio
	if requestData.MinRatio != 0 {
		var userID int
		var action string

		if requestData.Indexer == "redacted" {
			userID = requestData.REDUserID
			action = "user"
		} else if requestData.Indexer == "ops" {
			userID = requestData.OPSUserID
			action = "user"
		}

		if userID != 0 {
			data, err = api.fetchAPIData(action, userID, apiKey, requestData.Indexer)
			if err != nil {
				http.Error(w, fmt.Sprintf("Internal Server Error: %s", err.Error()), http.StatusInternalServerError)
				return
			}

			ratio := data.Response.Stats.Ratio
			minRatio := requestData.MinRatio
			username := data.Response.Username

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
		var action string = "torrent"
		if data == nil || data.Response.Torrent == nil {
			data, err = api.fetchAPIData(action, requestData.TorrentID, apiKey, requestData.Indexer)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		username := data.Response.Torrent.Username
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
		var action string = "torrent"
		if data == nil || data.Response.Torrent == nil {
			data, err = api.fetchAPIData(action, requestData.TorrentID, apiKey, requestData.Indexer)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		recordLabel := data.Response.Torrent.RecordLabel
		name := data.Response.Group.Name
		requestedRecordLabels := strings.Split(requestData.RecordLabel, ",")

		if recordLabel == "" {
			log.Debug().Msgf("No record label found for release: %s. Responding with status code 228.", name)
			w.WriteHeader(http.StatusIMUsed + 2) // HTTP status code 228
			return
		}

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
		// Check if MinSize is greater than MaxSize
		if requestData.MinSize != 0 && requestData.MaxSize != 0 && requestData.MinSize > requestData.MaxSize {
			errMessage := fmt.Sprintf("Invalid size range: MinSize %s is greater than MaxSize %s", requestData.MinSize.String(), requestData.MaxSize.String())
			http.Error(w, errMessage, http.StatusBadRequest) // HTTP status code 400
			log.Debug().Msg(errMessage)
			return
		}
		var action string = "torrent"
		if data == nil || data.Response.Torrent == nil {
			data, err = api.fetchAPIData(action, requestData.TorrentID, apiKey, requestData.Indexer)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		torrentSize := bytesize.ByteSize(data.Response.Torrent.Size)

		// Compare sizes
		if (requestData.MinSize != 0 && torrentSize < requestData.MinSize) ||
			(requestData.MaxSize != 0 && torrentSize > requestData.MaxSize) {
			w.WriteHeader(http.StatusIMUsed + 3) // HTTP status code 229
			log.Debug().Msgf("Torrent size: %s", torrentSize.String())
			if requestData.MinSize != 0 {
				log.Debug().Msgf("Requested min size: %s", requestData.MinSize.String())
			}
			if requestData.MaxSize != 0 {
				log.Debug().Msgf("Requested max size: %s", requestData.MaxSize.String())
			}
			log.Debug().Msgf("Torrent size %s is outside the requested size range, responding with status 229", torrentSize.String())
			return
		}

	}
	w.WriteHeader(http.StatusOK) // HTTP status code 200
	log.Debug().Msg("Conditions met, responding with status 200")
}
