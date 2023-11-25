package api

import (
	"testing"
)

func TestValidateRequestData(t *testing.T) {
	tests := []struct {
		name    string
		request RequestData
		wantErr bool
		errMsg  string
	}{
		// Valid request
		{"Valid request", RequestData{Indexer: "ops", TorrentID: 123, REDKey: "123456789012345678901234567890123456789012", OPSKey: "123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012", MinRatio: 1.0, MinSize: 0, MaxSize: 10, Uploaders: "uploader1", RecordLabel: "label1", Mode: "blacklist"}, false, ""},

		// Invalid indexer
		{"Invalid indexer", RequestData{Indexer: "invalid"}, true, "invalid indexer: invalid"},

		// Invalid torrent ID
		{"Invalid torrent ID", RequestData{Indexer: "ops", TorrentID: 1000000000}, true, "invalid torrent ID: 1000000000"},

		// REDKey too long
		{"REDKey too long", RequestData{Indexer: "redacted", REDKey: "12345678901234567890212345678901234567890123"}, true, "REDKey is too long"},

		// OPSKey too long
		{"OPSKey too long", RequestData{Indexer: "ops", OPSKey: "123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012109213091823098123091283"}, true, "OPSKey is too long"},

		// MinRatio out of range
		{"MinRatio out of range", RequestData{Indexer: "ops", MinRatio: 1000}, true, "minRatio must be between 0 and 999.999"},

		// MinSize greater than MaxSize
		{"MinSize greater than MaxSize", RequestData{Indexer: "ops", MinSize: 11, MaxSize: 10}, true, "minsize cannot be greater than maxsize"},

		// Invalid Uploaders
		{"Invalid Uploaders", RequestData{Indexer: "ops", Uploaders: "uploader#1"}, true, "uploaders field should only contain alphanumeric characters"},

		// Invalid RecordLabel
		{"Invalid RecordLabel", RequestData{Indexer: "ops", RecordLabel: "label#1"}, true, "recordLabels field should only contain alphanumeric characters, spaces, and safe special characters"},

		// Invalid Mode with Uploaders
		{"Invalid Mode with Uploaders", RequestData{Indexer: "ops", Uploaders: "uploader1", Mode: "invalid_mode"}, true, "mode must be either 'whitelist' or 'blacklist', got 'invalid_mode'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequestData(&tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequestData() error = %v, wantErr %v", err, tt.wantErr)
			} else if err != nil && err.Error() != tt.errMsg {
				t.Errorf("validateRequestData() error = %v, want %v", err, tt.errMsg)
			}
		})
	}
}
