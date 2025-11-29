package internal

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCacheBasic(t *testing.T) {
	t.Parallel()

	cache := NewCache(10, time.Hour)

	// Test Set and Get
	cache.Set("key1", "value1")
	val := cache.Get("key1")
	if val == nil {
		t.Fatal("Get() returned nil for existing key")
	}
	if val.(string) != "value1" {
		t.Errorf("Get() = %v, want %v", val, "value1")
	}

	// Test non-existent key
	val = cache.Get("nonexistent")
	if val != nil {
		t.Errorf("Get() = %v, want nil for non-existent key", val)
	}
}

func TestCacheEmptyKey(t *testing.T) {
	t.Parallel()

	cache := NewCache(10, time.Hour)

	cache.Set("", "value")
	val := cache.Get("")
	if val != nil {
		t.Error("Empty key should not be stored")
	}
}

func TestCacheNilValue(t *testing.T) {
	t.Parallel()

	cache := NewCache(10, time.Hour)

	cache.Set("key", nil)
	val := cache.Get("key")
	if val != nil {
		t.Error("Nil value should not be stored")
	}
}

func TestCacheTTL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TTL test in short mode")
	}

	cache := NewCache(10, 50*time.Millisecond)

	cache.Set("key", "value")

	// Should exist immediately
	val := cache.Get("key")
	if val == nil {
		t.Fatal("Get() returned nil immediately after Set()")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	val = cache.Get("key")
	if val != nil {
		t.Error("Get() should return nil after TTL expiration")
	}
}

func TestCacheNoTTL(t *testing.T) {
	t.Parallel()

	cache := NewCache(10, 0) // No TTL

	cache.Set("key", "value")

	val := cache.Get("key")
	if val == nil {
		t.Fatal("Get() returned nil for key with no TTL")
	}
}

func TestCacheEviction(t *testing.T) {
	t.Parallel()

	cache := NewCache(3, time.Hour) // Small cache

	// Fill cache
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// All should exist
	if cache.Get("key1") == nil || cache.Get("key2") == nil || cache.Get("key3") == nil {
		t.Fatal("All keys should exist after filling cache")
	}

	// Add one more - should trigger eviction
	cache.Set("key4", "value4")

	// key4 should exist
	if cache.Get("key4") == nil {
		t.Error("Newly added key should exist")
	}

	// At least one old key should be evicted
	count := 0
	if cache.Get("key1") != nil {
		count++
	}
	if cache.Get("key2") != nil {
		count++
	}
	if cache.Get("key3") != nil {
		count++
	}
	if count > 2 {
		t.Error("Eviction should have removed at least one old entry")
	}
}

func TestCacheUpdateExisting(t *testing.T) {
	t.Parallel()

	cache := NewCache(3, time.Hour)

	cache.Set("key", "value1")
	cache.Set("key", "value2")

	val := cache.Get("key")
	if val == nil {
		t.Fatal("Get() returned nil")
	}
	if val.(string) != "value2" {
		t.Errorf("Get() = %v, want %v", val, "value2")
	}
}

func TestCacheClear(t *testing.T) {
	t.Parallel()

	cache := NewCache(10, time.Hour)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	cache.Clear()

	if cache.Get("key1") != nil || cache.Get("key2") != nil {
		t.Error("Clear() should remove all entries")
	}
}

func TestCacheDisabled(t *testing.T) {
	t.Parallel()

	cache := NewCache(0, time.Hour) // Disabled cache

	cache.Set("key", "value")

	val := cache.Get("key")
	if val != nil {
		t.Error("Disabled cache should not store values")
	}
}

func TestCacheConcurrency(t *testing.T) {
	t.Parallel()

	cache := NewCache(100, time.Hour)

	const goroutines = 50
	const operations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)

				cache.Set(key, value)
				val := cache.Get(key)

				if val != nil && val.(string) != value {
					t.Errorf("Concurrent access: got %v, want %v", val, value)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestCacheLRUBehavior(t *testing.T) {
	t.Parallel()

	cache := NewCache(3, time.Hour)

	// Add entries
	cache.Set("key1", "value1")
	time.Sleep(10 * time.Millisecond)
	cache.Set("key2", "value2")
	time.Sleep(10 * time.Millisecond)
	cache.Set("key3", "value3")

	// Access key1 to update its last used time
	cache.Get("key1")

	// Add new entry - should evict key2 (least recently used)
	cache.Set("key4", "value4")

	// key1 should still exist (recently accessed)
	if cache.Get("key1") == nil {
		t.Error("Recently accessed key should not be evicted")
	}

	// key4 should exist (just added)
	if cache.Get("key4") == nil {
		t.Error("Newly added key should exist")
	}
}

func TestCacheExpiredEviction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping expiration test in short mode")
	}

	cache := NewCache(3, 50*time.Millisecond)

	// Add entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Add new entry - should clean up expired entries
	cache.Set("key3", "value3")

	// Expired entries should be gone
	if cache.Get("key1") != nil || cache.Get("key2") != nil {
		t.Error("Expired entries should be removed during eviction")
	}

	// New entry should exist
	if cache.Get("key3") == nil {
		t.Error("New entry should exist")
	}
}

func BenchmarkCacheSet(b *testing.B) {
	cache := NewCache(1000, time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		cache.Set(key, "value")
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := NewCache(1000, time.Hour)

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		cache.Set(fmt.Sprintf("key-%d", i), "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		cache.Get(key)
	}
}

func BenchmarkCacheConcurrent(b *testing.B) {
	cache := NewCache(1000, time.Hour)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%1000)
			if i%2 == 0 {
				cache.Set(key, "value")
			} else {
				cache.Get(key)
			}
			i++
		}
	})
}
