package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/s0up4200/redactedhook/internal/config"
)

func TestSendDiscordNotification(t *testing.T) {
	// Mock the Discord webhook URL
	webhookURL := "http://example.com/webhook"
	config.GetConfig().Notifications.DiscordWebhookURL = webhookURL

	// Create a test server to mock the Discord webhook
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.String() != "/webhook" {
			t.Errorf("Expected request to /webhook, got %s", r.URL.String())
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		if buf.String() != `{"content":"Test message"}` {
			t.Errorf("Expected body to be %s, got %s", `{"content":"Test message"}`, buf.String())
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	// Override the webhook URL to point to the test server
	config.GetConfig().Notifications.DiscordWebhookURL = ts.URL + "/webhook"

	// Call the function to test
	err := sendDiscordNotification("Test message")
	if err != nil {
		t.Errorf("sendDiscordNotification() error = %v", err)
	}
}
