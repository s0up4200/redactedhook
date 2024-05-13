package api

import (
	"time"

	"github.com/rs/zerolog/log"
)

const cacheExpiryDuration = 5 * time.Minute

type CacheItem struct {
	Data        *ResponseData
	LastFetched time.Time
}

var cache = make(map[string]CacheItem)

func cacheResponseData(cacheKey string, responseData *ResponseData) {
	cache[cacheKey] = CacheItem{
		Data:        responseData,
		LastFetched: time.Now(),
	}
}

func checkCache(cacheKey, indexer string) (*ResponseData, bool) {
	if cached, ok := cache[cacheKey]; ok && time.Since(cached.LastFetched) < cacheExpiryDuration {
		log.Trace().Msgf("[%s] Using cached data for %s", indexer, cacheKey)
		return cached.Data, true
	}
	return nil, false
}
