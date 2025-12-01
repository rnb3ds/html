package internal

import (
	"sync"
	"time"
)

type cacheEntry struct {
	value     any
	expiresAt int64
	lastUsed  int64
}

func (e *cacheEntry) isExpired(now int64) bool {
	return e.expiresAt > 0 && now > e.expiresAt
}

// Cache provides thread-safe caching with TTL and LRU eviction.
type Cache struct {
	mu         sync.RWMutex
	entries    map[string]*cacheEntry
	maxEntries int
	ttl        time.Duration
}

// NewCache creates a cache with the specified maximum entries and TTL.
func NewCache(maxEntries int, ttl time.Duration) *Cache {
	capacity := max(maxEntries, 0)
	return &Cache{
		entries:    make(map[string]*cacheEntry, capacity),
		maxEntries: maxEntries,
		ttl:        ttl,
	}
}

// Get retrieves a value from the cache.
func (c *Cache) Get(key string) any {
	if key == "" {
		return nil
	}
	now := time.Now().UnixNano()
	c.mu.Lock()
	defer c.mu.Unlock()
	entry := c.entries[key]
	if entry == nil {
		return nil
	}
	if entry.isExpired(now) {
		delete(c.entries, key)
		return nil
	}
	entry.lastUsed = now
	return entry.value
}

// Set adds or updates a value in the cache.
func (c *Cache) Set(key string, value any) {
	if value == nil || key == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.maxEntries == 0 {
		return
	}
	if len(c.entries) >= c.maxEntries {
		if _, exists := c.entries[key]; !exists {
			c.evictOne()
		}
	}
	now := time.Now()
	nowNano := now.UnixNano()
	entry := &cacheEntry{
		value:    value,
		lastUsed: nowNano,
	}
	if c.ttl > 0 {
		entry.expiresAt = now.Add(c.ttl).UnixNano()
	}
	c.entries[key] = entry
}

func (c *Cache) evictOne() {
	nowNano := time.Now().UnixNano()
	for k, e := range c.entries {
		if e.isExpired(nowNano) {
			delete(c.entries, k)
			return
		}
	}
	var oldestKey string
	var oldestTime int64 = 1<<63 - 1
	for k, e := range c.entries {
		if e.lastUsed < oldestTime {
			oldestKey = k
			oldestTime = e.lastUsed
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// Clear removes all entries from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	capacity := max(c.maxEntries, 0)
	c.entries = make(map[string]*cacheEntry, capacity)
}
