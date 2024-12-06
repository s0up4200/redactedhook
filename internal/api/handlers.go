package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/s0up4200/redactedhook/internal/config"
)

const (
	StatusUploaderNotAllowed = http.StatusIMUsed + 1
	StatusLabelNotAllowed    = http.StatusIMUsed + 2
	StatusSizeNotAllowed     = http.StatusIMUsed + 3
	StatusRatioNotAllowed    = http.StatusIMUsed
)

const (
	ErrInvalidJSONResponse   = "invalid JSON response"
	ErrRecordLabelNotFound   = "record label not found"
	ErrRecordLabelNotAllowed = "record label not allowed"
	ErrUploaderNotAllowed    = "uploader is not allowed"
	ErrSizeNotAllowed        = "torrent size is outside the requested size range"
	ErrRatioBelowMinimum     = "returned ratio is below minimum requirement"
)

type validationError struct {
	err    error
	status int
}

func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	cfg := config.GetConfig()
	var requestData RequestData

	if err := validateRequest(r, cfg, &requestData); err != nil {
		writeHTTPError(w, err.err, err.status)
		return
	}

	log.Info().Msgf("Received data request from %s", r.RemoteAddr)

	if err := processRequest(&requestData); err != nil {
		handleErrors(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Info().Msgf("[%s] Conditions met, responding with status 200", requestData.Indexer)

	if err := sendDiscordNotification("Request responded with HTTP 200"); err != nil {
		log.Error().Err(err).Msg("Failed to send Discord notification")
	}
}

func validateRequest(r *http.Request, cfg *config.Config, requestData *RequestData) *validationError {
	fallbackToConfig(requestData)

	if err := verifyAPIKey(r.Header.Get("X-API-Token"), cfg.Authorization.APIToken); err != nil {
		return &validationError{err, http.StatusUnauthorized}
	}

	if err := validateRequestMethod(r.Method); err != nil {
		return &validationError{err, http.StatusBadRequest}
	}

	if err := decodeJSONPayload(r, requestData); err != nil {
		return &validationError{err, http.StatusBadRequest}
	}
	defer r.Body.Close()

	if err := validateIndexer(requestData.Indexer); err != nil {
		return &validationError{err, http.StatusBadRequest}
	}

	if err := validateRequestData(requestData); err != nil {
		return &validationError{err, http.StatusBadRequest}
	}

	return nil
}

func processRequest(requestData *RequestData) error {
	apiBase, err := determineAPIBase(requestData.Indexer)
	if err != nil {
		return err
	}

	reqHeader := make(http.Header)
	if err := setAuthorizationHeader(&reqHeader, requestData); err != nil {
		return err
	}

	return runHooks(requestData, apiBase)
}

func runHooks(requestData *RequestData, apiBase string) error {
	if requestData.TorrentID != 0 && (requestData.MinSize != 0 || requestData.MaxSize != 0) {
		if err := hookSize(requestData, apiBase); err != nil {
			return errors.New(ErrSizeNotAllowed)
		}
	}

	if requestData.TorrentID != 0 && requestData.Uploaders != "" {
		if err := hookUploader(requestData, apiBase); err != nil {
			return errors.New(ErrUploaderNotAllowed)
		}
	}

	if requestData.TorrentID != 0 && requestData.RecordLabel != "" {
		if err := hookRecordLabel(requestData, apiBase); err != nil {
			return errors.New(ErrRecordLabelNotAllowed)
		}
	}

	if requestData.MinRatio != 0 {
		if err := hookRatio(requestData, apiBase); err != nil {
			return errors.New(ErrRatioBelowMinimum)
		}
	}

	return nil
}

func writeHTTPError(w http.ResponseWriter, err error, statusCode int) {
	http.Error(w, err.Error(), statusCode)
}

func handleErrors(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	switch err.Error() {
	case ErrInvalidJSONResponse:
		http.Error(w, ErrInvalidJSONResponse, http.StatusInternalServerError)

	case ErrRecordLabelNotFound:
		http.Error(w, ErrRecordLabelNotFound, http.StatusBadRequest)

	case ErrRecordLabelNotAllowed:
		http.Error(w, ErrRecordLabelNotAllowed, http.StatusForbidden)

	case ErrUploaderNotAllowed:
		http.Error(w, ErrUploaderNotAllowed, http.StatusForbidden)

	case ErrSizeNotAllowed:
		http.Error(w, ErrSizeNotAllowed, http.StatusBadRequest)

	case ErrRatioBelowMinimum:
		http.Error(w, ErrRatioBelowMinimum, http.StatusForbidden)

	default:
		log.Error().Err(err).Msg("Unhandled error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func sendDiscordNotification(message string) error {
	webhookURL := config.GetConfig().Notifications.DiscordWebhookURL
	if webhookURL == "" {
		return nil
	}

	payload := map[string]string{"content": message}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to send Discord notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code from Discord: %d", resp.StatusCode)
	}

	return nil
}
