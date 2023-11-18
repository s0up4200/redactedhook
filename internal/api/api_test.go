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
		{"Valid request", RequestData{Indexer: "ops", TorrentID: 123, REDKey: "12345678901234567890123456789012", OPSKey: "12345678901234567890123456789012", MinRatio: 1.0, MinSize: 0, MaxSize: 10, Uploaders: "uploader1", RecordLabel: "label1", Mode: "blacklist"}, false, ""},

		// Invalid indexer
		{"Invalid indexer", RequestData{Indexer: "invalid"}, true, "invalid indexer: invalid"},

		// Invalid torrent ID
		{"Invalid torrent ID", RequestData{Indexer: "ops", TorrentID: -1}, true, "invalid torrent ID: -1"},

		// REDKey too short
		{"REDKey too short", RequestData{Indexer: "ops", REDKey: "short"}, true, "REDKey is too short"},

		// OPSKey too short
		{"OPSKey too short", RequestData{Indexer: "ops", OPSKey: "short"}, true, "OPSKey is too short"},

		// MinRatio negative
		{"MinRatio negative", RequestData{Indexer: "ops", MinRatio: -1}, true, "minratio cannot be negative"},

		// MinSize greater than MaxSize
		{"MinSize greater than MaxSize", RequestData{Indexer: "ops", MinSize: 11, MaxSize: 10}, true, "minsize cannot be greater than maxsize"},

		// Invalid Uploaders
		{"Invalid Uploaders", RequestData{Indexer: "ops", Uploaders: "uploader#1"}, true, "uploaders field should only contain alphanumeric characters"},

		// Invalid RecordLabel
		{"Invalid RecordLabel", RequestData{Indexer: "ops", RecordLabel: "label#1"}, true, "record_labels field should only contain alphanumeric characters"},

		// Invalid Mode
		{"Invalid Mode", RequestData{Indexer: "ops", Mode: "invalid_mode"}, true, "invalid mode: invalid_mode"},
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
