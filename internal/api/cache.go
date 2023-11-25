package api

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

type CacheItem struct {
	Data        *ResponseData
	LastFetched time.Time
}

var cache = make(map[string]CacheItem) // Keyed by indexer

func cacheResponseData(cacheKey string, responseData *ResponseData) {
	cache[cacheKey] = CacheItem{
		Data:        responseData,
		LastFetched: time.Now(),
	}
}

func checkCache(cacheKey string, indexer string) (*ResponseData, bool) {
	if cached, ok := cache[cacheKey]; ok && time.Since(cached.LastFetched) < 5*time.Minute {
		log.Trace().Msgf("[%s] Using cached data for key: %s", indexer, cacheKey)
		return cached.Data, true
	}
	return nil, false
}

func getAPIKey(requestData *RequestData) (string, error) {
	switch requestData.Indexer {
	case "redacted":
		return requestData.REDKey, nil
	case "ops":
		return requestData.OPSKey, nil
	default:
		return "", fmt.Errorf("invalid indexer: %s", requestData.Indexer)
	}
}
