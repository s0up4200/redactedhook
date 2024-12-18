package config

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func setupTestEnv() {
	viper.Reset()
	os.Clearenv()
	viper.SetConfigType("toml")
	viper.SetConfigFile("testconfig.toml")
	if err := viper.ReadConfig(bytes.NewReader([]byte(`
	[authorization]
	api_token = "test_token"
	
	[indexer_keys]
	red_apikey = "red_key"
	ops_apikey = "ops_key"
	
	[userid]
	red_user_id = 1
	ops_user_id = 2
	
	[ratio]
	minratio = 0.5
	
	[record_labels]
	record_labels = "test_label"
	
	[logs]
	loglevel = "debug"
	logtofile = false
	logfilepath = "test.log"
	maxsize = 5
	maxbackups = 2
	maxage = 7
	compress = false
	
	[server]
	host = "127.0.0.1"
	port = 42135
	`))); err != nil {
		panic("Failed to read test config: " + err.Error())
	}
}

func TestValidateConfig(t *testing.T) {
	setupTestEnv()
	viper.Set("authorization.api_token", "")

	err := ValidateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Authorization API Token is required.")

	viper.Set("authorization.api_token", "valid_token")
	err = ValidateConfig()
	assert.NoError(t, err)
}

func TestWatchConfigChanges(t *testing.T) {
	setupTestEnv()

	// simulate a config file change
	viper.Set("server.port", 8080)
	err := viper.WriteConfigAs("testconfig_updated.toml")
	assert.NoError(t, err)

	InitConfig("testconfig_updated.toml")
	assert.Equal(t, 8080, config.Server.Port)

	os.Remove("testconfig_updated.toml")
}

func TestValidateConfigWithPartialIndexers(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func()
		wantErr     bool
		errMsg      string
	}{
		{
			name: "only RED configured",
			setupConfig: func() {
				setupTestEnv()
				viper.Set("indexer_keys.red_apikey", "valid_red_key")
				viper.Set("indexer_keys.ops_apikey", "")
			},
			wantErr: false,
		},
		{
			name: "only OPS configured",
			setupConfig: func() {
				setupTestEnv()
				viper.Set("indexer_keys.red_apikey", "")
				viper.Set("indexer_keys.ops_apikey", "valid_ops_key")
			},
			wantErr: false,
		},
		{
			name: "no indexers configured",
			setupConfig: func() {
				setupTestEnv()
				viper.Set("indexer_keys.red_apikey", "")
				viper.Set("indexer_keys.ops_apikey", "")
			},
			wantErr: true,
			errMsg:  "At least one indexer API key (RED or OPS) must be configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupConfig()
			err := ValidateConfig()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
