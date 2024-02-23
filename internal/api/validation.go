package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

// verifyAPIKey checks if the provided API key matches the expected one.
func verifyAPIKey(headerAPIKey string, expectedAPIKey string) error {
	if expectedAPIKey == "" || headerAPIKey != expectedAPIKey {
		return fmt.Errorf("invalid or missing API key")
	}
	return nil
}

// validateRequestMethod ensures the request uses the POST method.
func validateRequestMethod(method string) error {
	if method != http.MethodPost {
		return fmt.Errorf("only POST method is supported")
	}
	return nil
}

// checks if the given `RequestData` object contains valid data and returns an error if any of the validations fail.
func validateRequestData(requestData *RequestData) error {
	uploadersRegex := regexp.MustCompile(`^[a-zA-Z0-9_. ]+$`)
	safeCharacterRegex := regexp.MustCompile(`^[\p{L}\p{N}\s&,-]+$`)

	if requestData.Indexer != "ops" && requestData.Indexer != "redacted" {
		errMsg := fmt.Sprintf("invalid indexer: %s", requestData.Indexer)
		log.Debug().Msg(errMsg)
		return fmt.Errorf(errMsg)
	}

	if requestData.TorrentID > 999999999 {
		errMsg := fmt.Sprintf("invalid torrent ID: %d", requestData.TorrentID)
		log.Debug().Msg(errMsg)
		return fmt.Errorf(errMsg)
	}

	if requestData.REDKey != "" && len(requestData.REDKey) > 42 {
		errMsg := "REDKey is too long"
		log.Debug().Msg(errMsg)
		return fmt.Errorf(errMsg)
	}

	if requestData.OPSKey != "" && len(requestData.OPSKey) > 120 {
		errMsg := "OPSKey is too long"
		log.Debug().Msg(errMsg)
		return fmt.Errorf(errMsg)
	}

	if requestData.MinRatio < 0 || requestData.MinRatio > 999.999 {
		errMsg := "minRatio must be between 0 and 999.999"
		log.Debug().Msg(errMsg)
		return fmt.Errorf(errMsg)
	}

	if requestData.MaxSize > 0 && requestData.MinSize > requestData.MaxSize {
		errMsg := "minsize cannot be greater than maxsize"
		log.Debug().Msg(errMsg)
		return fmt.Errorf(errMsg)
	}

	if requestData.Uploaders != "" {
		if !uploadersRegex.MatchString(requestData.Uploaders) {
			errMsg := "uploaders field should only contain alphanumeric characters"
			log.Debug().Msg(errMsg)
			return fmt.Errorf(errMsg)
		}

		if requestData.Mode != "whitelist" && requestData.Mode != "blacklist" {
			errMsg := fmt.Sprintf("mode must be either 'whitelist' or 'blacklist', got '%s'", requestData.Mode)
			log.Debug().Msg(errMsg)
			return fmt.Errorf(errMsg)
		}
	}

	if requestData.RecordLabel != "" {
		labels := strings.Split(requestData.RecordLabel, ",")
		for _, label := range labels {
			trimmedLabel := strings.TrimSpace(label)
			if !safeCharacterRegex.MatchString(trimmedLabel) {
				errMsg := "recordLabels field should only contain alphanumeric characters, spaces, and safe special characters"
				log.Debug().Msg(errMsg)
				return fmt.Errorf(errMsg)
			}
		}
	}

	return nil
}

// checks if a given indexer string is valid or not.
func validateIndexer(indexer string) error {
	if indexer != "ops" && indexer != "redacted" {
		if indexer == "" {
			return fmt.Errorf("no indexer provided")
		}
		return fmt.Errorf("invalid indexer: %s", indexer)
	}
	return nil
}
