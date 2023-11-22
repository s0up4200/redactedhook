package api

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/inhies/go-bytesize"
	"github.com/rs/zerolog/log"
	"github.com/s0up4200/redactedhook/internal/config"
)

func validateRequestData(requestData *RequestData) error {
	alphanumericRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	safeCharacterRegex := regexp.MustCompile(`^[\w\s&,-]+$`)

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
		if !alphanumericRegex.MatchString(requestData.Uploaders) {
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

func fallbackToConfig(requestData *RequestData, cfg *config.Config) {
	if requestData.REDUserID == 0 {
		requestData.REDUserID = cfg.UserIDs.REDUserID
	}
	if requestData.OPSUserID == 0 {
		requestData.OPSUserID = cfg.UserIDs.OPSUserID
	}
	if requestData.REDKey == "" {
		requestData.REDKey = cfg.IndexerKeys.REDKey
	}
	if requestData.OPSKey == "" {
		requestData.OPSKey = cfg.IndexerKeys.OPSKey
	}
	if requestData.MinRatio == 0 {
		requestData.MinRatio = cfg.Ratio.MinRatio
	}
	if requestData.MinSize == 0 {
		requestData.MinSize = bytesize.ByteSize(cfg.ParsedSizes.MinSize)
	}
	if requestData.MaxSize == 0 {
		requestData.MaxSize = bytesize.ByteSize(cfg.ParsedSizes.MaxSize)
	}
	if requestData.Uploaders == "" {
		requestData.Uploaders = cfg.Uploaders.Uploaders
	}
	if requestData.Mode == "" {
		requestData.Mode = cfg.Uploaders.Mode
	}
	if requestData.RecordLabel == "" {
		requestData.RecordLabel = cfg.RecordLabels.RecordLabels
	}
}
