package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/s0up4200/redactedhook/internal/config"
)

func TestGenerateAPIToken(t *testing.T) {
	token, err := generateAPIToken()
	if err != nil {
		t.Errorf("generateAPIToken() error = %v", err)
	}
	if len(token) != tokenLength*2 { // *2 because hex encoding doubles length
		t.Errorf("generateAPIToken() token length = %v, want %v", len(token), tokenLength*2)
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultVal   string
		envVal       string
		expected     string
		shouldSetEnv bool
	}{
		{
			name:         "returns default when env not set",
			key:          "TEST_KEY",
			defaultVal:   "default",
			envVal:       "",
			expected:     "default",
			shouldSetEnv: false,
		},
		{
			name:         "returns env value when set",
			key:          "TEST_KEY",
			defaultVal:   "default",
			envVal:       "custom",
			expected:     "custom",
			shouldSetEnv: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSetEnv {
				os.Setenv(envPrefix+tt.key, tt.envVal)
				defer os.Unsetenv(envPrefix + tt.key)
			}

			if got := getEnv(tt.key, tt.defaultVal); got != tt.expected {
				t.Errorf("getEnv() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHasRequiredEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		{
			envVars: map[string]string{
				envPrefix + "API_TOKEN":  "token",
				envPrefix + "RED_APIKEY": "red",
				envPrefix + "OPS_APIKEY": "ops",
			},
			expected: true,
		},
		{
			name: "missing one required var",
			envVars: map[string]string{
				envPrefix + "API_TOKEN":  "token",
				envPrefix + "RED_APIKEY": "red",
			},
			expected: false,
		},
		{
			name:     "no vars present",
			envVars:  map[string]string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment before each test
			os.Clearenv()

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			if got := hasRequiredEnvVars(); got != tt.expected {
				t.Errorf("hasRequiredEnvVars() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoadEnvironmentConfig(t *testing.T) {
	// Save original config
	originalConfig := config.GetConfig()
	defer func() {
		// Restore original config after test
		config.GetConfig().Server = originalConfig.Server
		config.GetConfig().Authorization = originalConfig.Authorization
		config.GetConfig().IndexerKeys = originalConfig.IndexerKeys
		config.GetConfig().Logs = originalConfig.Logs
	}()

	tests := []struct {
		name  string
		env   map[string]string
		check func(t *testing.T)
	}{
		{
			name: "server settings",
			env: map[string]string{
				envPrefix + "HOST": "0.0.0.0",
				envPrefix + "PORT": "8080",
			},
			check: func(t *testing.T) {
				if config.GetConfig().Server.Host != "0.0.0.0" {
					t.Errorf("Host = %v, want %v", config.GetConfig().Server.Host, "0.0.0.0")
				}
				if config.GetConfig().Server.Port != 8080 {
					t.Errorf("Port = %v, want %v", config.GetConfig().Server.Port, 8080)
				}
			},
		},
		{
			name: "authorization settings",
			env: map[string]string{
				envPrefix + "API_TOKEN":  "test-token",
				envPrefix + "RED_APIKEY": "red-key",
				envPrefix + "OPS_APIKEY": "ops-key",
			},
			check: func(t *testing.T) {
				if config.GetConfig().Authorization.APIToken != "test-token" {
					t.Errorf("APIToken = %v, want %v", config.GetConfig().Authorization.APIToken, "test-token")
				}
				if config.GetConfig().IndexerKeys.REDKey != "red-key" {
					t.Errorf("REDKey = %v, want %v", config.GetConfig().IndexerKeys.REDKey, "red-key")
				}
				if config.GetConfig().IndexerKeys.OPSKey != "ops-key" {
					t.Errorf("OPSKey = %v, want %v", config.GetConfig().IndexerKeys.OPSKey, "ops-key")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment before each test
			os.Clearenv()

			// Set test environment variables
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			loadEnvironmentConfig()
			tt.check(t)
		})
	}
}

func TestCreateServer(t *testing.T) {
	address := "localhost:8080"
	server := createServer(address)

	if server.Addr != address {
		t.Errorf("server.Addr = %v, want %v", server.Addr, address)
	}

	if server.ReadTimeout != readTimeout {
		t.Errorf("server.ReadTimeout = %v, want %v", server.ReadTimeout, readTimeout)
	}

	if server.WriteTimeout != writeTimeout {
		t.Errorf("server.WriteTimeout = %v, want %v", server.WriteTimeout, writeTimeout)
	}

	if server.IdleTimeout != idleTimeout {
		t.Errorf("server.IdleTimeout = %v, want %v", server.IdleTimeout, idleTimeout)
	}

	if server.ReadHeaderTimeout != readHeaderTimeout {
		t.Errorf("server.ReadHeaderTimeout = %v, want %v", server.ReadHeaderTimeout, readHeaderTimeout)
	}
}

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "OK"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}
