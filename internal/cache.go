// Package internal provides caching functionality for content extraction results.
// It implements a thread-safe LRU cache with TTL support to improve performance
// for repeated extractions of the same content.
package internal

import (
	"sync"
	"time"
)

type cacheEntry struct {
	prev, next *cacheEntry
	lastUsed   int64
	expiresAt  int64
	value      any
	key        string
}

func (e *cacheEntry) isExpired(now int64) bool {
	return e.expiresAt > 0 && now > e.expiresAt
}

type Cache struct {
	mu         sync.RWMutex
	entries    map[string]*cacheEntry
	maxEntries int
	ttl        time.Duration
	head, tail *cacheEntry // Sentinel nodes for doubly-linked list
}

func NewCache(maxEntries int, ttl time.Duration) *Cache {
	if maxEntries < 0 {
		maxEntries = 0
	}
	c := &Cache{
		entries:    make(map[string]*cacheEntry, maxEntries),
		maxEntries: maxEntries,
		ttl:        ttl,
	}
	// Initialize sentinel nodes
	c.head = &cacheEntry{}
	c.tail = &cacheEntry{}
	c.head.next = c.tail
	c.tail.prev = c.head
	return c
}

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
		c.removeNode(entry)
		delete(c.entries, key)
		return nil
	}

	// Update last used time and move to front
	entry.lastUsed = now
	c.moveToFront(entry)

	return entry.value
}

func (c *Cache) Set(key string, value any) {
	if value == nil || key == "" || c.maxEntries == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixNano()

	// Check if key already exists (update case)
	if entry, exists := c.entries[key]; exists {
		entry.value = value
		entry.lastUsed = now
		if c.ttl > 0 {
			entry.expiresAt = now + c.ttl.Nanoseconds()
		}
		c.moveToFront(entry)
		return
	}

	// Need to add new entry - check capacity
	if len(c.entries) >= c.maxEntries {
		c.evictOne()
	}

	// Create new entry
	entry := &cacheEntry{
		value:     value,
		lastUsed:  now,
		key:       key,
		expiresAt: 0,
	}
	if c.ttl > 0 {
		entry.expiresAt = now + c.ttl.Nanoseconds()
	}

	// Add to front of list and map
	c.entries[key] = entry
	c.addToFront(entry)
}

// moveToFront moves an entry to the front (most recently used position)
func (c *Cache) moveToFront(entry *cacheEntry) {
	if entry == nil || entry == c.head || entry == c.tail {
		return
	}
	// Remove from current position
	entry.prev.next = entry.next
	entry.next.prev = entry.prev
	// Add to front
	c.addToFront(entry)
}

// addToFront adds an entry right after head (most recently used position)
func (c *Cache) addToFront(entry *cacheEntry) {
	if entry == nil {
		return
	}
	entry.prev = c.head
	entry.next = c.head.next
	c.head.next.prev = entry
	c.head.next = entry
}

// removeNode removes an entry from the doubly-linked list
func (c *Cache) removeNode(entry *cacheEntry) {
	if entry == nil || entry == c.head || entry == c.tail {
		return
	}
	entry.prev.next = entry.next
	entry.next.prev = entry.prev
	entry.prev = nil
	entry.next = nil
}

func (c *Cache) evictOne() {
	nowNano := time.Now().UnixNano()

	// First, try to remove an expired entry
	for key, entry := range c.entries {
		if entry.isExpired(nowNano) {
			c.removeNode(entry)
			delete(c.entries, key)
			return
		}
	}

	// If no expired entries, evict the least recently used (tail.prev)
	// The tail.prev node is the LRU entry (right before tail sentinel)
	if c.tail.prev != c.head {
		lruEntry := c.tail.prev
		c.removeNode(lruEntry)
		delete(c.entries, lruEntry.key)
	}
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Clear all entries
	for key := range c.entries {
		delete(c.entries, key)
	}
	// Reset linked list
	c.head.next = c.tail
	c.tail.prev = c.head
}
