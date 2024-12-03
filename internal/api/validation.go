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

	if requestData.Indexer == "redacted" && requestData.REDKey == "" {
		log.Debug().Msg("Missing RED API key")
		return fmt.Errorf("RED API key is required for Redacted indexer")
	}

	if requestData.Indexer == "ops" && requestData.OPSKey == "" {
		log.Debug().Msg("Missing OPS API key")
		return fmt.Errorf("OPS API key is required for Orpheus indexer")
	}

	if requestData.TorrentID > 999_999_999 {
		log.Debug().Int("torrentID", requestData.TorrentID).Msg("Invalid torrent ID")
		return fmt.Errorf("invalid torrent ID: %d", requestData.TorrentID)
	}

	if len(requestData.REDKey) > 42 {
		log.Debug().Msg("REDKey is too long")
		return fmt.Errorf("REDKey is too long")
	}

	if len(requestData.OPSKey) > 120 {
		log.Debug().Msg("OPSKey is too long")
		return fmt.Errorf("OPSKey is too long")
	}

	if requestData.MinRatio < 0 || requestData.MinRatio > 999.999 {
		log.Debug().Msg("minRatio must be between 0 and 999.999")
		return fmt.Errorf("minRatio must be between 0 and 999.999")
	}

	if requestData.MaxSize > 0 && requestData.MinSize > requestData.MaxSize {
		log.Debug().Msg("minSize cannot be greater than maxSize")
		return fmt.Errorf("minSize cannot be greater than maxSize")
	}

	if requestData.Uploaders != "" {
		if requestData.Mode != "whitelist" && requestData.Mode != "blacklist" {
			log.Debug().Str("mode", requestData.Mode).Msg("Invalid mode")
			return fmt.Errorf("mode must be either 'whitelist' or 'blacklist', got '%s'", requestData.Mode)
		}
	}

	if requestData.RecordLabel != "" {
		labels := strings.Split(requestData.RecordLabel, ",")
		for _, label := range labels {
			trimmedLabel := strings.TrimSpace(label)
			if !safeCharacterRegex.MatchString(trimmedLabel) {
				log.Debug().Msg("Invalid record label format")
				return fmt.Errorf("recordLabels field should only contain alphanumeric characters, spaces, and safe special characters")
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
