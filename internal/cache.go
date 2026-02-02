package internal

import (
	"sync"
	"sync/atomic"
	"time"
)

type cacheEntry struct {
	lastUsed  int64
	expiresAt int64
	value     any
}

func (e *cacheEntry) isExpired(now int64) bool {
	return e.expiresAt > 0 && now > e.expiresAt
}

type Cache struct {
	mu         sync.RWMutex
	entries    map[string]*cacheEntry
	maxEntries int
	ttl        time.Duration
}

func NewCache(maxEntries int, ttl time.Duration) *Cache {
	if maxEntries < 0 {
		maxEntries = 0
	}
	return &Cache{
		entries:    make(map[string]*cacheEntry, maxEntries),
		maxEntries: maxEntries,
		ttl:        ttl,
	}
}

func (c *Cache) Get(key string) any {
	if key == "" {
		return nil
	}
	now := time.Now().UnixNano()

	c.mu.RLock()
	entry := c.entries[key]
	if entry == nil {
		c.mu.RUnlock()
		return nil
	}

	if entry.isExpired(now) {
		c.mu.RUnlock()
		c.mu.Lock()
		// Re-check entry after acquiring write lock
		entry = c.entries[key]
		if entry != nil && entry.isExpired(now) {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return nil
	}

	value := entry.value
	atomic.StoreInt64(&entry.lastUsed, now)
	c.mu.RUnlock()

	return value
}

func (c *Cache) Set(key string, value any) {
	if value == nil || key == "" || c.maxEntries == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.entries[key]
	if !exists && len(c.entries) >= c.maxEntries {
		c.evictOne()
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
	var oldestKey string
	var oldestTime int64 = 1<<63 - 1

	// First pass: collect and remove all expired entries
	for k, e := range c.entries {
		if e.isExpired(nowNano) {
			delete(c.entries, k)
			return
		}
		lastUsed := atomic.LoadInt64(&e.lastUsed)
		if lastUsed < oldestTime {
			oldestKey = k
			oldestTime = lastUsed
		}
	}

	// Second pass: evict oldest entry if no expired entries found
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry, c.maxEntries)
}
