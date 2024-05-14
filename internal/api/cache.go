package api

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	cacheExpiryDuration  = 5 * time.Minute
	cacheCleanupInterval = 10 * time.Minute
)

type CacheItem struct {
	Data        *ResponseData
	LastFetched time.Time
}

var (
	cache     = make(map[string]CacheItem)
	cacheLock sync.RWMutex
)

func init() {
	// Start a background goroutine to periodically clean up expired cache entries.
	go startCacheCleanup()
}

func cacheResponseData(cacheKey string, responseData *ResponseData) {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	cache[cacheKey] = CacheItem{
		Data:        responseData,
		LastFetched: time.Now(),
	}
}

func checkCache(cacheKey, indexer string) (*ResponseData, bool) {
	cacheLock.RLock()
	defer cacheLock.RUnlock()

	if cached, ok := cache[cacheKey]; ok {
		if time.Since(cached.LastFetched) < cacheExpiryDuration {
			log.Trace().Msgf("[%s] Using cached data for %s", indexer, cacheKey)
			return cached.Data, true
		}
	}
	return nil, false
}

func startCacheCleanup() {
	for {
		time.Sleep(cacheCleanupInterval)
		removeExpiredCacheEntries()
	}
}

func removeExpiredCacheEntries() {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	now := time.Now()
	for key, item := range cache {
		if now.Sub(item.LastFetched) >= cacheExpiryDuration {
			delete(cache, key)
			//log.Trace().Msgf("Removed expired cache entry for %s", key)
		}
	}
}
