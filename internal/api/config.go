package api

import (
	"github.com/inhies/go-bytesize"
	"github.com/s0up4200/redactedhook/internal/config"
)

type configField struct {
	webhookField interface{}
	configValue  interface{}
}

func fallbackToConfig(requestData *RequestData) {
	cfg := config.GetConfig()

	fields := []configField{
		{&requestData.REDUserID, cfg.UserIDs.REDUserID},
		{&requestData.OPSUserID, cfg.UserIDs.OPSUserID},
		{&requestData.REDKey, cfg.IndexerKeys.REDKey},
		{&requestData.OPSKey, cfg.IndexerKeys.OPSKey},
		{&requestData.MinRatio, cfg.Ratio.MinRatio},
		{&requestData.MinSize, cfg.ParsedSizes.MinSize},
		{&requestData.MaxSize, cfg.ParsedSizes.MaxSize},
		{&requestData.Uploaders, cfg.Uploaders.Uploaders},
		{&requestData.Mode, cfg.Uploaders.Mode},
		{&requestData.RecordLabel, cfg.RecordLabels.RecordLabels},
	}

	for _, field := range fields {
		switch v := field.webhookField.(type) {
		case *int:
			if *v == 0 {
				*v = field.configValue.(int)
			}
		case *float64:
			if *v == 0 {
				*v = field.configValue.(float64)
			}
		case *bytesize.ByteSize:
			if *v == 0 {
				*v = field.configValue.(bytesize.ByteSize)
			}
		case *string:
			if *v == "" {
				*v = field.configValue.(string)
			}
		}
	}
}
