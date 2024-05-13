package api

import (
	"fmt"
	"net/http"
	"strings"

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
	cfg := config.GetConfig()
	var requestData RequestData
	fallbackToConfig(&requestData)

	if err := verifyAPIKey(r.Header.Get("X-API-Token"), cfg.Authorization.APIToken); err != nil {
		writeHTTPError(w, err, http.StatusUnauthorized)
		return
	}

	if err := validateRequestMethod(r.Method); err != nil {
		writeHTTPError(w, err, http.StatusBadRequest)
		return
	}

	if err := decodeJSONPayload(r, &requestData); err != nil {
		writeHTTPError(w, err, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := validateIndexer(requestData.Indexer); err != nil {
		writeHTTPError(w, err, http.StatusBadRequest)
		return
	}

	if err := validateRequestData(&requestData); err != nil {
		writeHTTPError(w, err, http.StatusBadRequest)
		return
	}

	log.Info().Msgf("Received data request from %s", r.RemoteAddr)

	apiBase, err := determineAPIBase(requestData.Indexer)
	if err != nil {
		writeHTTPError(w, err, http.StatusNotFound)
		return
	}

	reqHeader := make(http.Header)
	setAuthorizationHeader(&reqHeader, &requestData)

	if hookError := runHooks(&requestData, apiBase); hookError != nil {
		handleErrors(w, hookError)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Info().Msgf("[%s] Conditions met, responding with status 200", requestData.Indexer)
}

func runHooks(requestData *RequestData, apiBase string) error {
	if requestData.TorrentID != 0 && (requestData.MinSize != 0 || requestData.MaxSize != 0) {
		if err := hookSize(requestData, apiBase); err != nil {
			return errWithStatus(err, StatusSizeNotAllowed)
		}
	}

	if requestData.TorrentID != 0 && requestData.Uploaders != "" {
		if err := hookUploader(requestData, apiBase); err != nil {
			return errWithStatus(err, StatusUploaderNotAllowed)
		}
	}

	if requestData.TorrentID != 0 && requestData.RecordLabel != "" {
		if err := hookRecordLabel(requestData, apiBase); err != nil {
			return errWithStatus(err, StatusLabelNotAllowed)
		}
	}

	if requestData.MinRatio != 0 {
		if err := hookRatio(requestData, apiBase); err != nil {
			return errWithStatus(err, StatusRatioNotAllowed)
		}
	}

	return nil
}

type statusError struct {
	err        error
	statusCode int
}

func (se *statusError) Error() string {
	return se.err.Error()
}

func errWithStatus(err error, statusCode int) error {
	return &statusError{err: err, statusCode: statusCode}
}

func writeHTTPError(w http.ResponseWriter, err error, statusCode int) {
	http.Error(w, err.Error(), statusCode)
}

func handleErrors(w http.ResponseWriter, err error) {
	if serr, ok := err.(*statusError); ok {
		if strings.Contains(serr.Error(), "invalid JSON response") {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		} else if handleHTTPErrorWithStatus(w, serr.Error()) {
			return
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func handleHTTPErrorWithStatus(w http.ResponseWriter, errMsg string) bool {
	if strings.HasPrefix(errMsg, "HTTP error:") {
		var statusCode int
		if _, scanErr := fmt.Sscanf(errMsg, "HTTP error: %d", &statusCode); scanErr == nil && statusCode != 0 {
			http.Error(w, errMsg, statusCode)
			return true
		}
	}
	return false
}
