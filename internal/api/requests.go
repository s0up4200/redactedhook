package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

// sends an HTTP GET request to an endpoint with an API key, applies a rate limiter, and unmarshals the response JSON into a target object.
func makeRequest(endpoint, apiKey string, limiter *rate.Limiter, indexer string, target interface{}) error {

	if !limiter.Allow() {
		log.Warn().Msgf("%s: Too many requests", indexer)
		return fmt.Errorf("rate limit exceeded for %s", indexer)
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		log.Error().Err(err).Msg("Error creating HTTP request")
		return err
	}
	req.Header.Set("Authorization", apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Error executing HTTP request")
		return err
	}
	defer resp.Body.Close()

	//dump, err := httputil.DumpResponse(resp, true)
	//if err != nil {
	//	log.Error().Err(err).Msg("Error dumping the response")
	//	return err
	//}
	//
	//fmt.Printf("HTTP Response:\n%s\n", dump)

	if resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("HTTP error: %d from %s", resp.StatusCode, endpoint)
		log.Error().Msg(errMsg)
		return errors.New(errMsg)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("fetchAPI error")
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
		//	log.Warn().Msgf("API error from %s: %s", indexer, responseData.Error)
		return fmt.Errorf("API error from %s: %s", indexer, responseData.Error)
	}

	return nil
}

// initiates an API request with the given parameters and returns the response data or an error.
func initiateAPIRequest(id int, action string, apiKey, apiBase, indexer string) (*ResponseData, error) {
	limiter := getLimiter(indexer)
	if limiter == nil {
		return nil, fmt.Errorf("could not get rate limiter for indexer: %s", indexer)
	}

	endpoint := fmt.Sprintf("%s?action=%s&id=%d", apiBase, action, id)
	responseData := &ResponseData{}
	err := makeRequest(endpoint, apiKey, limiter, indexer, responseData)
	if err != nil {
		return nil, err
	}

	if action == "torrent" && responseData.Response.Torrent != nil {
		releaseName := html.UnescapeString(responseData.Response.Torrent.ReleaseName)
		uploader := responseData.Response.Torrent.Username
		log.Debug().Msgf("[%s] Checking release: %s - (Uploader: %s) (TorrentID: %d)", indexer, releaseName, uploader, id)
	}

	return responseData, nil
}

// fetches response data from an API, checks the cache first, and caches the response data for future use.
func fetchResponseData(requestData *RequestData, id int, action string, apiBase string) (*ResponseData, error) {

	// Check cache first
	cacheKey := fmt.Sprintf("%sID %d", action, id)
	cachedData, found := checkCache(cacheKey, requestData.Indexer)
	if found {
		return cachedData, nil
	}

	apiKey, err := getAPIKey(requestData)
	if err != nil {
		return nil, err
	}

	responseData, err := initiateAPIRequest(id, action, apiKey, apiBase, requestData.Indexer)
	if err != nil {
		if strings.Contains(err.Error(), "rate limit exceeded") {
			return nil, err
		}
		wrappedErr := fmt.Errorf("error fetching %s data for ID %d: %w", action, id, err)
		log.Error().Err(wrappedErr).Msg("Data fetching")
		return nil, wrappedErr
	}

	// Cache the response data
	cacheResponseData(cacheKey, responseData)

	return responseData, nil
}

// determines the API base endpoint based on the provided indexer.
func determineAPIBase(indexer string) (string, error) {
	switch indexer {
	case "redacted":
		return APIEndpointBaseRedacted, nil
	case "ops":
		return APIEndpointBaseOrpheus, nil
	default:
		return "", fmt.Errorf("invalid path")
	}
}
