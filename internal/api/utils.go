package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// takes a slice of strings and returns a new slice with all the labels
// converted to lowercase and trimmed of any leading or trailing whitespace.
func normalizeLabels(labels []string) []string {
	normalized := make([]string, len(labels))
	for i, label := range labels {
		normalized[i] = strings.ToLower(strings.TrimSpace(label))
	}
	return normalized
}
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// returns the appropriate API key based on the indexer specified in the `requestData` parameter.
func getAPIKey(requestData *RequestData) (string, error) {
	switch requestData.Indexer {
	case "redacted":
		return requestData.REDKey, nil
	case "ops":
		return requestData.OPSKey, nil
	default:
		return "", fmt.Errorf("invalid indexer: %s", requestData.Indexer)
	}
}

// sets the Authorization header in an HTTP request header based on the indexer specified
func setAuthorizationHeader(reqHeader *http.Header, requestData *RequestData) {
	var apiKey string
	if requestData.Indexer == "redacted" {
		apiKey = requestData.REDKey
	} else if requestData.Indexer == "ops" {
		apiKey = requestData.OPSKey
	}
	reqHeader.Set("Authorization", apiKey)
}

// decodes a JSON payload from an HTTP request and stores it in a struct.
func decodeJSONPayload(r *http.Request, requestData *RequestData) error {
	if err := json.NewDecoder(r.Body).Decode(requestData); err != nil {
		return fmt.Errorf("invalid JSON payload")
	}
	return nil
}
