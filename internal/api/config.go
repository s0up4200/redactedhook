package api

import (
	"github.com/inhies/go-bytesize"
	"github.com/s0up4200/redactedhook/internal/config"
)

// fallbackToConfig checks if certain fields in the requestData struct are empty or zero,
// and if so, populates them with values from the cfg struct.
func fallbackToConfig(requestData *RequestData) {
	cfg := config.GetConfig()

	setIfEmptyInt := func(field *int, value int) {
		if *field == 0 {
			*field = value
		}
	}

	setIfEmptyFloat64 := func(field *float64, value float64) {
		if *field == 0 {
			*field = value
		}
	}

	setIfEmptyByteSize := func(field *bytesize.ByteSize, value bytesize.ByteSize) {
		if *field == 0 {
			*field = value
		}
	}

	setIfEmptyString := func(field *string, value string) {
		if *field == "" {
			*field = value
		}
	}

	setIfEmptyInt(&requestData.REDUserID, cfg.UserIDs.REDUserID)
	setIfEmptyInt(&requestData.OPSUserID, cfg.UserIDs.OPSUserID)
	setIfEmptyString(&requestData.REDKey, cfg.IndexerKeys.REDKey)
	setIfEmptyString(&requestData.OPSKey, cfg.IndexerKeys.OPSKey)
	setIfEmptyFloat64(&requestData.MinRatio, cfg.Ratio.MinRatio)
	setIfEmptyByteSize(&requestData.MinSize, cfg.ParsedSizes.MinSize)
	setIfEmptyByteSize(&requestData.MaxSize, cfg.ParsedSizes.MaxSize)
	setIfEmptyString(&requestData.Uploaders, cfg.Uploaders.Uploaders)
	setIfEmptyString(&requestData.Mode, cfg.Uploaders.Mode)
	setIfEmptyString(&requestData.RecordLabel, cfg.RecordLabels.RecordLabels)
}
