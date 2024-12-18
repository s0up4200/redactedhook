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
			name:    "Empty request",
			request: RequestData{},
			wantErr: true,
			errMsg:  "no indexer provided",
		},
		{
			name:    "Invalid indexer",
			request: RequestData{Indexer: "invalid"},
			wantErr: true,
			errMsg:  "invalid indexer: invalid",
		},
		{
			name: "Minimum valid torrent ID",
			request: RequestData{
				Indexer:   "ops",
				TorrentID: 1,
				OPSKey:    "validkey123",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Maximum valid torrent ID",
			request: RequestData{
				Indexer:   "ops",
				TorrentID: 999999999,
				OPSKey:    "validkey123",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "REDKey at maximum length",
			request: RequestData{
				Indexer: "redacted",
				REDKey:  "123456789012345678901234567890123456789012",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "OPSKey at maximum length",
			request: RequestData{
				Indexer: "ops",
				OPSKey:  "0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "MinRatio at lower boundary",
			request: RequestData{
				Indexer:  "ops",
				MinRatio: 0,
				OPSKey:   "validkey123",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "MinRatio at upper boundary",
			request: RequestData{
				Indexer:  "ops",
				MinRatio: 999.999,
				OPSKey:   "validkey123",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Valid RecordLabel with special characters",
			request: RequestData{
				Indexer:     "ops",
				RecordLabel: "label1 & label2 - label3",
				OPSKey:      "validkey123",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Empty RecordLabel field",
			request: RequestData{
				Indexer:     "ops",
				RecordLabel: "",
				OPSKey:      "validkey123",
			},
			wantErr: false,
			errMsg:  "",
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
