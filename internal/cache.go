package internal

import (
	"sync"
	"sync/atomic"
	"time"
)

type cacheEntry struct {
	value     any
	expiresAt int64 // Unix nano timestamp, 0 means no expiration
	lastUsed  atomic.Int64
}

func (e *cacheEntry) isExpired(now int64) bool {
	exp := e.expiresAt
	return exp > 0 && now > exp
}

// Cache provides thread-safe caching with TTL and LRU eviction.
type Cache struct {
	mu         sync.RWMutex
	entries    map[string]*cacheEntry
	maxEntries int
	ttl        time.Duration
}

// NewCache creates a new cache with the specified maximum entries and TTL.
func NewCache(maxEntries int, ttl time.Duration) *Cache {
	capacity := maxEntries
	if capacity <= 0 {
		capacity = 0
	}
	return &Cache{
		entries:    make(map[string]*cacheEntry, capacity),
		maxEntries: maxEntries,
		ttl:        ttl,
	}
}

func (c *Cache) Get(key string) any {
	if key == "" {
		return nil
	}

	c.mu.RLock()
	entry := c.entries[key]
	c.mu.RUnlock()

	if entry == nil {
		return nil
	}

	now := time.Now().UnixNano()

	// Check expiration before updating lastUsed to avoid unnecessary atomic operation
	if entry.isExpired(now) {
		// Need to recheck under write lock to avoid race condition
		c.mu.Lock()
		// Recheck if entry still exists and is expired (it might have been deleted by another goroutine)
		if currentEntry, exists := c.entries[key]; exists && currentEntry.isExpired(now) {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return nil
	}

	// Update last used time atomically (safe without lock)
	entry.lastUsed.Store(now)
	return entry.value
}

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
		value: value,
	}
	entry.lastUsed.Store(nowNano)

	if c.ttl > 0 {
		entry.expiresAt = now.Add(c.ttl).UnixNano()
	}

	c.entries[key] = entry
}

func (c *Cache) evictOne() {
	nowNano := time.Now().UnixNano()
	var oldestKey string
	oldestTime := int64(1<<63 - 1) // max int64
	expiredCount := 0

	for k, e := range c.entries {
		if e.isExpired(nowNano) {
			delete(c.entries, k)
			expiredCount++
			if expiredCount >= 10 {
				return
			}
			continue
		}
		if lastUsed := e.lastUsed.Load(); lastUsed < oldestTime {
			oldestKey = k
			oldestTime = lastUsed
		}
	}

	if expiredCount == 0 && oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	capacity := c.maxEntries
	if capacity <= 0 {
		capacity = 0
	}
	c.entries = make(map[string]*cacheEntry, capacity)
}
