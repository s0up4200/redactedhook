package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/s0up4200/redactedhook/internal/config"
)

// handles webhooks: auth, decode payload, validate, respond 200.
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	var requestData RequestData

	cfg := config.GetConfig()
	fallbackToConfig(&requestData)

	if err := verifyAPIKey(r.Header.Get("X-API-Token"), cfg.Authorization.APIToken); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err := validateRequestMethod(r.Method); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := decodeJSONPayload(r, &requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := validateIndexer(requestData.Indexer); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := validateRequestData(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info().Msgf("Received data request from %s", r.RemoteAddr)

	apiBase, err := determineAPIBase(requestData.Indexer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	reqHeader := make(http.Header)
	setAuthorizationHeader(&reqHeader, &requestData)

	// Call hooks

	if requestData.TorrentID != 0 && (requestData.MinSize != 0 || requestData.MaxSize != 0) {
		if err := hookSize(&requestData, apiBase); err != nil {
			handleErrors(w, err, StatusSizeNotAllowed)
			return

		}
	}

	if requestData.TorrentID != 0 && requestData.Uploaders != "" {
		if err := hookUploader(&requestData, apiBase); err != nil {
			handleErrors(w, err, StatusUploaderNotAllowed)
			return

		}
	}

	if requestData.TorrentID != 0 && requestData.RecordLabel != "" {
		if err := hookRecordLabel(&requestData, apiBase); err != nil {
			handleErrors(w, err, StatusLabelNotAllowed)
			return
		}
	}

	if requestData.MinRatio != 0 {
		if err := hookRatio(&requestData, apiBase); err != nil {
			handleErrors(w, err, StatusRatioNotAllowed)
			return

		}
	}

	w.WriteHeader(http.StatusOK) // HTTP status code 200
	log.Info().Msgf("[%s] Conditions met, responding with status 200", requestData.Indexer)
}

func handleErrors(w http.ResponseWriter, err error, defaultStatusCode int) {
	if strings.Contains(err.Error(), "invalid JSON response") {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return // We're done here, no need to continue.
	}

	if strings.HasPrefix(err.Error(), "HTTP error:") {
		// Extract the status code from the error message
		var statusCode int
		_, scanErr := fmt.Sscanf(err.Error(), "HTTP error: %d", &statusCode)
		if scanErr == nil && statusCode != 0 {
			http.Error(w, err.Error(), statusCode)
			return // We're done here, too.
		}
		// Fallback to internal server error if status code extraction fails
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return // Still done.
	}

	http.Error(w, err.Error(), defaultStatusCode)
}
