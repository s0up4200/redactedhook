package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/inhies/go-bytesize"
	"github.com/rs/zerolog/log"
	"github.com/s0up4200/redactedhook/internal/config"
)

const (
	StatusUploaderNotAllowed = http.StatusIMUsed + 1
	StatusLabelNotAllowed    = http.StatusIMUsed + 2
	StatusSizeNotAllowed     = http.StatusIMUsed + 3
	StatusRatioNotAllowed    = http.StatusIMUsed
)

func WebhookHandler(w http.ResponseWriter, r *http.Request) {

	var torrentData *ResponseData
	var userData *ResponseData
	var requestData RequestData

	cfg := config.GetConfig()

	// Check for API key in the request header
	apiKeyHeader := r.Header.Get("X-API-Token")
	if cfg.Authorization.APIToken == "" || apiKeyHeader != cfg.Authorization.APIToken {
		log.Error().Msg("Invalid or missing API key")
		http.Error(w, "Invalid or missing API key", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is supported", http.StatusBadRequest)
		log.Warn().Msgf("Invalid method: %s", r.Method)
		return
	}

	// Read and validate JSON payload
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		log.Warn().Err(err).Msg("Invalid JSON payload")
		return
	}

	defer r.Body.Close()

	if requestData.Indexer != "ops" && requestData.Indexer != "redacted" {
		if requestData.Indexer == "" {
			log.Error().Msg("No indexer provided")
			http.Error(w, "no indexer provided", http.StatusBadRequest)
		} else {
			log.Error().Msgf("Invalid indexer: %s", requestData.Indexer)
			http.Error(w, fmt.Sprintf("Invalid indexer: %s", requestData.Indexer), http.StatusBadRequest)
		}
		return
	}

	// Check each field in requestData and fallback to config if empty
	fallbackToConfig(&requestData, cfg)

	// Validate requestData fields
	if err := validateRequestData(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Log request received
	logMsg := fmt.Sprintf("Received data request from %s", r.RemoteAddr)
	log.Info().Msg(logMsg)

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

	// hook uploader
	if requestData.TorrentID != 0 && requestData.Uploaders != "" {
		if err := fetchResponseData(&requestData, &torrentData, requestData.TorrentID, "torrent", apiBase); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		username := torrentData.Response.Torrent.Username
		usernames := strings.Split(requestData.Uploaders, ",")

		for i, username := range usernames { // Trim whitespace from each username
			usernames[i] = strings.TrimSpace(username)
		}
		usernamesStr := strings.Join(usernames, ", ") // Join the usernames with a comma and a single space
		log.Trace().Msgf("[%s] Requested uploaders [%s]: %s", requestData.Indexer, requestData.Mode, usernamesStr)

		isListed := false
		for _, uname := range usernames {
			if uname == username {
				isListed = true
				break
			}
		}

		if (requestData.Mode == "blacklist" && isListed) || (requestData.Mode == "whitelist" && !isListed) {
			http.Error(w, "Uploader is not allowed", StatusUploaderNotAllowed)
			log.Debug().Msgf("[%s] Uploader (%s) is not allowed", requestData.Indexer, username)
			return
		}
	}

	// hook record label
	if requestData.TorrentID != 0 && requestData.RecordLabel != "" {
		if err := fetchResponseData(&requestData, &torrentData, requestData.TorrentID, "torrent", apiBase); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		recordLabel := strings.ToLower(strings.TrimSpace(torrentData.Response.Torrent.RecordLabel))
		name := torrentData.Response.Group.Name

		requestedRecordLabels := normalizeLabels(strings.Split(requestData.RecordLabel, ","))

		if recordLabel == "" {
			log.Debug().Msgf("[%s] No record label found for release: %s", requestData.Indexer, name)
			http.Error(w, "Record label not allowed", StatusLabelNotAllowed)
			return
		}

		recordLabelsStr := strings.Join(requestedRecordLabels, ", ")
		log.Trace().Msgf("[%s] Requested record labels: [%s]", requestData.Indexer, recordLabelsStr)

		isRecordLabelPresent := contains(requestedRecordLabels, recordLabel)

		if !isRecordLabelPresent {
			log.Debug().Msgf("[%s] The record label '%s' is not included in the requested record labels: [%s]", requestData.Indexer, recordLabel, recordLabelsStr)
			http.Error(w, "Record label not allowed", StatusLabelNotAllowed)
			return
		}
	}

	// hook size
	if requestData.TorrentID != 0 && (requestData.MinSize != 0 || requestData.MaxSize != 0) {
		if err := fetchResponseData(&requestData, &torrentData, requestData.TorrentID, "torrent", apiBase); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		torrentSize := bytesize.ByteSize(torrentData.Response.Torrent.Size)

		minSize := bytesize.ByteSize(requestData.MinSize)
		maxSize := bytesize.ByteSize(requestData.MaxSize)

		log.Trace().Msgf("[%s] Torrent size: %s, Requested size range: %s - %s", requestData.Indexer, torrentSize, requestData.MinSize, requestData.MaxSize)

		if (requestData.MinSize != 0 && torrentSize < minSize) ||
			(requestData.MaxSize != 0 && torrentSize > maxSize) {
			log.Debug().Msgf("[%s] Torrent size %s is outside the requested size range: %s to %s", requestData.Indexer, torrentSize, minSize, maxSize)
			http.Error(w, "Torrent size is outside the requested size range", StatusSizeNotAllowed)
			return
		}
	}

	// hook ratio
	if requestData.MinRatio != 0 {
		userID := requestData.REDUserID
		if requestData.Indexer == "ops" {
			userID = requestData.OPSUserID
		}
		if err := fetchResponseData(&requestData, &userData, userID, "user", apiBase); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ratio := userData.Response.Stats.Ratio
		minRatio := requestData.MinRatio
		username := userData.Response.Username

		log.Trace().Msgf("[%s] MinRatio set to %.2f for %s", requestData.Indexer, minRatio, username)

		if ratio < minRatio {
			http.Error(w, "Returned ratio is below minimum requirement", StatusRatioNotAllowed)
			log.Debug().Msgf("[%s] Returned ratio %.2f is below minratio %.2f for %s", requestData.Indexer, ratio, minRatio, username)
			return

		}
	}

	w.WriteHeader(http.StatusOK) // HTTP status code 200
	log.Info().Msgf("[%s] Conditions met, responding with status 200", requestData.Indexer)
}
