package api

import (
	"github.com/s0up4200/redactedhook/internal/config"
)

// checks if certain fields in the requestData struct are empty or zero,
// and if so, it populates them with values from the cfg struct.
func fallbackToConfig(requestData *RequestData) {
	config := config.GetConfig()

	// Directly assign values from the global config if they are not set in requestData
	if requestData.REDUserID == 0 {
		requestData.REDUserID = config.UserIDs.REDUserID
	}
	if requestData.OPSUserID == 0 {
		requestData.OPSUserID = config.UserIDs.OPSUserID
	}
	if requestData.REDKey == "" {
		requestData.REDKey = config.IndexerKeys.REDKey
	}
	if requestData.OPSKey == "" {
		requestData.OPSKey = config.IndexerKeys.OPSKey
	}
	if requestData.MinRatio == 0 {
		requestData.MinRatio = config.Ratio.MinRatio
	}
	if requestData.MinSize == 0 {
		requestData.MinSize = config.ParsedSizes.MinSize
	}
	if requestData.MaxSize == 0 {
		requestData.MaxSize = config.ParsedSizes.MaxSize
	}
	if requestData.Uploaders == "" {
		requestData.Uploaders = config.Uploaders.Uploaders
	}
	if requestData.Mode == "" {
		requestData.Mode = config.Uploaders.Mode
	}
	if requestData.RecordLabel == "" {
		requestData.RecordLabel = config.RecordLabels.RecordLabels
	}
}
