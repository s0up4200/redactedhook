package api

import (
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/s0up4200/redactedhook/internal/config"
)

const (
	StatusLabelNotAllowed = http.StatusIMUsed + 1
	StatusRatioNotAllowed = http.StatusIMUsed
)

const (
	ErrInvalidJSONResponse   = "invalid JSON response"
	ErrRecordLabelNotFound   = "record label not found"
	ErrRecordLabelNotAllowed = "record label not allowed"
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

	case ErrRatioBelowMinimum:
		http.Error(w, ErrRatioBelowMinimum, http.StatusForbidden)

	default:
		log.Error().Err(err).Msg("Unhandled error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
