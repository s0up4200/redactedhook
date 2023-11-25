package api

import (
	"time"

	"github.com/rs/zerolog/log"
)

var cache = make(map[string]CacheItem) // keyed by indexer

// stores the responseData in cache with the specified cacheKey and updates the LastFetched timestamp.
func cacheResponseData(cacheKey string, responseData *ResponseData) {
	cache[cacheKey] = CacheItem{
		Data:        responseData,
		LastFetched: time.Now(),
	}
}

// checks if there is cached data for a given cache key and indexer,
// and returns the cached data if it exists and is not expired.
func checkCache(cacheKey string, indexer string) (*ResponseData, bool) {
	if cached, ok := cache[cacheKey]; ok && time.Since(cached.LastFetched) < 5*time.Minute {
		log.Trace().Msgf("[%s] Using cached data for %s", indexer, cacheKey)
		return cached.Data, true
	}
	return nil, false
}
