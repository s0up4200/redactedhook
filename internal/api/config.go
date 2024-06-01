package api

import (
	"github.com/inhies/go-bytesize"
	"github.com/s0up4200/redactedhook/internal/config"
)

// fallbackToConfig prioritizes webhook data over config data.
// If webhook data is present, it overwrites the existing config data.
func fallbackToConfig(requestData *RequestData) {
	cfg := config.GetConfig()

	// Helper functions to set fields, prioritizing webhook data if present
	setInt := func(webhookField *int, configValue int) {
		if *webhookField == 0 {
			*webhookField = configValue
		}
	}

	setFloat64 := func(webhookField *float64, configValue float64) {
		if *webhookField == 0 {
			*webhookField = configValue
		}
	}

	setByteSize := func(webhookField *bytesize.ByteSize, configValue bytesize.ByteSize) {
		if *webhookField == 0 {
			*webhookField = configValue
		}
	}

	setString := func(webhookField *string, configValue string) {
		if *webhookField == "" {
			*webhookField = configValue
		}
	}

	// Check and set the fields, ensuring webhook data takes priority if present
	setInt(&requestData.REDUserID, cfg.UserIDs.REDUserID)
	setInt(&requestData.OPSUserID, cfg.UserIDs.OPSUserID)
	setString(&requestData.REDKey, cfg.IndexerKeys.REDKey)
	setString(&requestData.OPSKey, cfg.IndexerKeys.OPSKey)
	setFloat64(&requestData.MinRatio, cfg.Ratio.MinRatio)
	setByteSize(&requestData.MinSize, cfg.ParsedSizes.MinSize)
	setByteSize(&requestData.MaxSize, cfg.ParsedSizes.MaxSize)
	setString(&requestData.Uploaders, cfg.Uploaders.Uploaders)
	setString(&requestData.Mode, cfg.Uploaders.Mode)
	setString(&requestData.RecordLabel, cfg.RecordLabels.RecordLabels)
}
