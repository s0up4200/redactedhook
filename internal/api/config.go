package api

import (
	"github.com/inhies/go-bytesize"
	"github.com/rs/zerolog/log"
	"github.com/s0up4200/redactedhook/internal/config"
	"github.com/spf13/viper"
)

// checks if certain fields in the requestData struct are empty or zero,
// and if so, it populates them with values from the cfg struct.
func fallbackToConfig(requestData *RequestData) {

	cfg := config.Config{}

	needsConfig := requestData.REDUserID == 0 ||
		requestData.OPSUserID == 0 ||
		requestData.REDKey == "" ||
		requestData.OPSKey == "" ||
		requestData.MinRatio == 0 ||
		requestData.MinSize == 0 ||
		requestData.MaxSize == 0 ||
		requestData.Uploaders == "" ||
		requestData.Mode == "" ||
		requestData.RecordLabel == ""

	if needsConfig {
		if err := viper.Unmarshal(&cfg); err != nil {
			log.Error().Err(err).Msg("Unable to decode into struct")
			return
		}
	}

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
