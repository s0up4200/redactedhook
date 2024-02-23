package api

import (
	"testing"
)

func TestValidateRequestData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request RequestData
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid request",
			request: RequestData{Indexer: "ops", TorrentID: 123, REDKey: "123456789012345678901234567890123456789012", OPSKey: "123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012", MinRatio: 1.0, MinSize: 0, MaxSize: 10, Uploaders: "uploader1", RecordLabel: "label1", Mode: "blacklist"},
			wantErr: false,
			errMsg:  "",
		},
		{
			name:    "Invalid indexer",
			request: RequestData{Indexer: "invalid"},
			wantErr: true,
			errMsg:  "invalid indexer: invalid",
		},
		{
			name:    "Invalid torrent ID",
			request: RequestData{Indexer: "ops", TorrentID: 1000000000},
			wantErr: true,
			errMsg:  "invalid torrent ID: 1000000000",
		},
		{
			name:    "REDKey too long",
			request: RequestData{Indexer: "redacted", REDKey: "12345678901234567890212345678901234567890123"},
			wantErr: true,
			errMsg:  "REDKey is too long",
		},
		{
			name:    "OPSKey too long",
			request: RequestData{Indexer: "ops", OPSKey: "123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012109213091823098123091283"},
			wantErr: true,
			errMsg:  "OPSKey is too long",
		},
		{
			name:    "MinRatio out of range",
			request: RequestData{Indexer: "ops", MinRatio: 1000},
			wantErr: true,
			errMsg:  "minRatio must be between 0 and 999.999",
		},
		{
			name:    "MinSize greater than MaxSize",
			request: RequestData{Indexer: "ops", MinSize: 11, MaxSize: 10},
			wantErr: true,
			errMsg:  "minsize cannot be greater than maxsize",
		},
		{
			name:    "Invalid Uploaders",
			request: RequestData{Indexer: "ops", Uploaders: "uploader#1"},
			wantErr: true,
			errMsg:  "uploaders field should only contain alphanumeric characters and underscores",
		},
		{
			name:    "Valid Uploaders",
			request: RequestData{Indexer: "ops", Uploaders: "uploader_1", Mode: "whitelist"},
			wantErr: false,
			errMsg:  "",
		},
		{
			name:    "Invalid RecordLabel",
			request: RequestData{Indexer: "ops", RecordLabel: "label#1"},
			wantErr: true,
			errMsg:  "recordLabels field should only contain alphanumeric characters, spaces, and safe special characters",
		},
		{
			name:    "Invalid Mode with Uploaders",
			request: RequestData{Indexer: "ops", Uploaders: "uploader1", Mode: "invalid_mode"},
			wantErr: true,
			errMsg:  "mode must be either 'whitelist' or 'blacklist', got 'invalid_mode'",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateRequestData(&tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequestData() error = %v, wantErr %v", err, tt.wantErr)
			} else if err != nil && err.Error() != tt.errMsg {
				t.Errorf("validateRequestData() error = %v, want %v", err, tt.errMsg)
			}
		})
	}
}
