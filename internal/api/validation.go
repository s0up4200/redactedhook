package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

func verifyAPIKey(headerAPIKey, expectedAPIKey string) error {
	if expectedAPIKey == "" || headerAPIKey != expectedAPIKey {
		return fmt.Errorf("invalid or missing API key")
	}
	return nil
}

func validateRequestMethod(method string) error {
	if method != http.MethodPost {
		return fmt.Errorf("only POST method is supported")
	}
	return nil
}

func validateRequestData(requestData *RequestData) error {
	safeCharacterRegex := regexp.MustCompile(`^[\p{L}\p{N}\s&,-]+$`)

	if err := validateIndexer(requestData.Indexer); err != nil {
		log.Debug().Err(err).Msg("Validation error")
		return err
	}

	if requestData.TorrentID > 999_999_999 {
		errMsg := fmt.Sprintf("invalid torrent ID: %d", requestData.TorrentID)
		log.Debug().Msg(errMsg)
		return fmt.Errorf(errMsg)
	}

	if len(requestData.REDKey) > 42 {
		errMsg := "REDKey is too long"
		log.Debug().Msg(errMsg)
		return fmt.Errorf(errMsg)
	}

	if len(requestData.OPSKey) > 120 {
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
		errMsg := "minSize cannot be greater than maxSize"
		log.Debug().Msg(errMsg)
		return fmt.Errorf(errMsg)
	}

	if requestData.Uploaders != "" {
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

func validateIndexer(indexer string) error {
	if indexer != "ops" && indexer != "redacted" {
		if indexer == "" {
			return fmt.Errorf("no indexer provided")
		}
		return fmt.Errorf("invalid indexer: %s", indexer)
	}
	return nil
}
