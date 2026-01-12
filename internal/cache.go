package internal

import (
	"sync"
	"sync/atomic"
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

type Cache struct {
	mu         sync.RWMutex
	entries    map[string]*cacheEntry
	maxEntries int
	ttl        time.Duration
}

func NewCache(maxEntries int, ttl time.Duration) *Cache {
	capacity := max(maxEntries, 0)
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
	now := time.Now().UnixNano()

	c.mu.RLock()
	entry := c.entries[key]
	if entry == nil {
		c.mu.RUnlock()
		return nil
	}

	// Check expiration under read lock
	if entry.isExpired(now) {
		c.mu.RUnlock()
		// Upgrade to write lock to delete expired entry
		c.mu.Lock()
		// Double-check after acquiring write lock
		if entry := c.entries[key]; entry != nil && entry.isExpired(now) {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return nil
	}

	// Entry exists and not expired - get value under read lock
	value := entry.value
	// Update lastUsed with atomic operation (safe even after RUnlock)
	atomic.StoreInt64(&entry.lastUsed, now)
	c.mu.RUnlock()

	return value
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
	var expiredKeys []string

	// First pass: collect expired entries and find oldest (all under write lock)
	// We're already holding the lock from Set(), so we can safely iterate
	for k, e := range c.entries {
		if e.isExpired(nowNano) {
			expiredKeys = append(expiredKeys, k)
		} else {
			lastUsed := atomic.LoadInt64(&e.lastUsed)
			if lastUsed < oldestTime {
				oldestKey = k
				oldestTime = lastUsed
			}
		}
	}

	// Delete expired entries first
	for _, k := range expiredKeys {
		delete(c.entries, k)
		if len(expiredKeys) > 1 {
			// If we deleted multiple expired entries, we're done
			return
		}
	}

	// If no expired entries found, remove the oldest one
	if len(expiredKeys) == 0 && oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	capacity := max(c.maxEntries, 0)
	c.entries = make(map[string]*cacheEntry, capacity)
}
