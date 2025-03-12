package cache

import (
	"sync"
	"time"
)

type CacheKey string

type CacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
}

type QueryCache struct {
	mu    sync.RWMutex
	cache map[string]CacheEntry
}

func NewQueryCache() *QueryCache {
	return &QueryCache{
		cache: make(map[string]CacheEntry),
	}
}

func (c *QueryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if entry, exists := c.cache[key]; exists {
		if time.Now().Before(entry.ExpiresAt) {
			return entry.Data, true
		}
		// Clean up expired entry
		delete(c.cache, key)
	}
	return nil, false
}

func (c *QueryCache) Set(key string, data interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = CacheEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(ttl),
	}
}
