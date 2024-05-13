package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

func setAuthorizationHeader(reqHeader *http.Header, requestData *RequestData) error {
	var apiKey string
	switch requestData.Indexer {
	case "redacted":
		apiKey = requestData.REDKey
	case "ops":
		apiKey = requestData.OPSKey
	default:
		err := fmt.Errorf("invalid indexer: %s", requestData.Indexer)
		log.Error().Err(err).Msg("Failed to set authorization header")
		return err
	}
	reqHeader.Set("Authorization", apiKey)
	return nil
}

func decodeJSONPayload(r *http.Request, requestData *RequestData) error {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(requestData); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	return nil
}
