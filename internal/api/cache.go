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
	done      = make(chan struct{}) // Channel to signal cleanup goroutine to stop
)

func init() {
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
	ticker := time.NewTicker(cacheCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			removeExpiredCacheEntries()
		case <-done:
			return
		}
	}
}

func removeExpiredCacheEntries() {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	now := time.Now()
	for key, item := range cache {
		if now.Sub(item.LastFetched) >= cacheExpiryDuration {
			delete(cache, key)
			// log.Trace().Msgf("Removed expired cache entry for %s", key)
		}
	}
}

// StopCache stops the cleanup goroutine gracefully
func StopCache() {
	close(done)
}
