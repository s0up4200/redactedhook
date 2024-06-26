package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

const (
	APIEndpointBaseRedacted = "https://redacted.ch/ajax.php"
	APIEndpointBaseOrpheus  = "https://orpheus.network/ajax.php"
)

func makeRequest(endpoint, apiKey string, limiter *rate.Limiter, indexer string, target interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := limiter.Wait(ctx); err != nil {
		log.Warn().Msgf("%s: Rate limit exceeded", indexer)
		return fmt.Errorf("rate limit exceeded for %s: %w", indexer, err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		log.Error().Err(err).Msg("Error creating HTTP request")
		return err
	}
	req.Header.Set("Authorization", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Error executing HTTP request")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("HTTP error: %d from %s", resp.StatusCode, endpoint)
		log.Error().Msg(errMsg)
		return errors.New(errMsg)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Error reading response body")
		return err
	}

	if err := json.Unmarshal(respBody, target); err != nil {
		log.Error().Err(err).Msg("Invalid JSON response")
		return fmt.Errorf("invalid JSON response: %w", err)
	}

	responseData, ok := target.(*ResponseData)
	if !ok {
		log.Error().Msg("Invalid target type for JSON unmarshalling")
		return fmt.Errorf("invalid target type")
	}

	if responseData.Status != "success" {
		return fmt.Errorf("API error from %s: %s", indexer, responseData.Error)
	}

	return nil
}

func initiateAPIRequest(id int, action, apiKey, apiBase, indexer string) (*ResponseData, error) {
	limiter, err := getLimiter(indexer)
	if err != nil {
		return nil, fmt.Errorf("could not get rate limiter for indexer: %s, %w", indexer, err)
	}

	endpoint := fmt.Sprintf("%s?action=%s&id=%d", apiBase, action, id)
	responseData := &ResponseData{}
	if err := makeRequest(endpoint, apiKey, limiter, indexer, responseData); err != nil {
		return nil, err
	}

	if action == "torrent" && responseData.Response.Torrent != nil {
		releaseName := html.UnescapeString(responseData.Response.Torrent.ReleaseName)
		uploader := responseData.Response.Torrent.Username
		log.Debug().Msgf("[%s] Checking release: %s - (Uploader: %s) (TorrentID: %d)", indexer, releaseName, uploader, id)
	}

	return responseData, nil
}

// fetchResponseData fetches response data from an API, checks the cache first, and caches the response data for future use.
func fetchResponseData(requestData *RequestData, id int, action, apiBase string) (*ResponseData, error) {
	cacheKey := fmt.Sprintf("%sID %d", action, id)
	if cachedData, found := checkCache(cacheKey, requestData.Indexer); found {
		return cachedData, nil
	}

	apiKey, err := getAPIKey(requestData)
	if err != nil {
		return nil, err
	}

	responseData, err := initiateAPIRequest(id, action, apiKey, apiBase, requestData.Indexer)
	if err != nil {
		wrappedErr := fmt.Errorf("error fetching %s data for ID %d: %w", action, id, err)
		log.Error().Err(wrappedErr).Msg("Data fetching")
		return nil, wrappedErr
	}

	cacheResponseData(cacheKey, responseData)
	return responseData, nil
}

func determineAPIBase(indexer string) (string, error) {
	switch indexer {
	case "redacted":
		return APIEndpointBaseRedacted, nil
	case "ops":
		return APIEndpointBaseOrpheus, nil
	default:
		return "", fmt.Errorf("invalid indexer: %s", indexer)
	}
}

func getAPIKey(requestData *RequestData) (string, error) {
	switch requestData.Indexer {
	case "redacted":
		return requestData.REDKey, nil
	case "ops":
		return requestData.OPSKey, nil
	default:
		return "", errors.New("invalid indexer")
	}
}
