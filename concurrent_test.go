// Package html provides concurrent safety tests for the HTML processor.
// These tests verify thread-safety and detect race conditions.
package html

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cybergodev/html/internal"
)

// TestConcurrentProcessorExtraction tests concurrent HTML extraction.
func TestConcurrentProcessorExtraction(t *testing.T) {
	config := DefaultConfig()
	config.MaxCacheEntries = 100
	config.CacheTTL = time.Minute
	processor, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Close()

	html := []byte(`<html><body><p>Test content</p></body></html>`)
	numGoroutines := 100
	numOperations := 50

	var wg sync.WaitGroup
	var errorCount atomic.Int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_, err := processor.Extract(html)
				if err != nil {
					errorCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	if errorCount.Load() > 0 {
		t.Errorf("Concurrent extraction had %d errors", errorCount.Load())
	}

	stats := processor.GetStatistics()
	expectedOps := int64(numGoroutines * numOperations)
	if stats.TotalProcessed != expectedOps {
		t.Errorf("Expected %d total processed, got %d", expectedOps, stats.TotalProcessed)
	}
}

// TestConcurrentCacheOperations tests concurrent cache access.
func TestConcurrentCacheOperations(t *testing.T) {
	cache := internal.NewCache(1000, time.Minute)

	numGoroutines := 50
	numOperations := 100

	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cache.Set(key, fmt.Sprintf("value-%d-%d", id, j))
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cache.Get(key)
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrentCacheSetGet tests concurrent Set and Get on same keys.
func TestConcurrentCacheSetGet(t *testing.T) {
	cache := internal.NewCache(100, time.Minute)

	numGoroutines := 20
	numOperations := 1000
	keys := []string{"shared-1", "shared-2", "shared-3"}

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(2)

		// Writer goroutine
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				for _, key := range keys {
					cache.Set(key, fmt.Sprintf("writer-%d-value-%d", id, j))
				}
			}
		}(i)

		// Reader goroutine
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				for _, key := range keys {
					_ = cache.Get(key)
				}
			}
		}()
	}

	wg.Wait()
}

// TestConcurrentCacheClear tests concurrent Clear with Set/Get.
func TestConcurrentCacheClear(t *testing.T) {
	cache := internal.NewCache(100, time.Minute)

	numGoroutines := 30
	numOperations := 200

	var wg sync.WaitGroup

	// Writers
	for i := 0; i < numGoroutines/3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cache.Set(fmt.Sprintf("key-%d", id), fmt.Sprintf("value-%d", j))
			}
		}(i)
	}

	// Readers
	for i := 0; i < numGoroutines/3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cache.Get(fmt.Sprintf("key-%d", id))
			}
		}(i)
	}

	// Clearers
	for i := 0; i < numGoroutines/3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations/10; j++ {
				cache.Clear()
			}
		}()
	}

	wg.Wait()
}

// TestConcurrentAuditCollector tests concurrent audit logging.
func TestConcurrentAuditCollector(t *testing.T) {
	config := AuditConfig{
		Enabled:           true,
		LogBlockedTags:    true,
		LogBlockedAttrs:   true,
		LogBlockedURLs:    true,
		IncludeRawValues:  true,
		MaxRawValueLength: 100,
	}
	collector := NewAuditCollector(config)
	defer collector.Close()

	numGoroutines := 50
	numOperations := 100

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				collector.RecordBlockedTag(fmt.Sprintf("script-%d-%d", id, j))
				collector.RecordBlockedAttr("onclick", fmt.Sprintf("alert(%d)", id))
				collector.RecordBlockedURL(fmt.Sprintf("javascript:alert(%d)", id), "xss")
			}
		}(i)
	}

	// Concurrent reads during writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = collector.GetEntries()
			}
		}()
	}

	wg.Wait()

	entries := collector.GetEntries()
	expectedEntries := numGoroutines * numOperations * 3 // 3 types of records
	if len(entries) != expectedEntries {
		t.Errorf("Expected %d entries, got %d", expectedEntries, len(entries))
	}
}

// TestConcurrentAuditCollectorClear tests concurrent Clear with Record.
func TestConcurrentAuditCollectorClear(t *testing.T) {
	config := AuditConfig{Enabled: true}
	collector := NewAuditCollector(config)
	defer collector.Close()

	numGoroutines := 20
	numOperations := 200

	var wg sync.WaitGroup

	// Writers
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				collector.RecordBlockedTag(fmt.Sprintf("tag-%d-%d", id, j))
			}
		}(i)
	}

	// Clearers
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations/10; j++ {
				collector.Clear()
			}
		}()
	}

	wg.Wait()
}

// TestConcurrentProcessorStatistics tests concurrent statistics access.
func TestConcurrentProcessorStatistics(t *testing.T) {
	config := DefaultConfig()
	config.MaxCacheEntries = 100
	config.CacheTTL = time.Minute
	processor, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Close()

	html := []byte(`<html><body><p>Test</p></body></html>`)
	numGoroutines := 50
	numOperations := 100

	var wg sync.WaitGroup

	// Processors
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_, _ = processor.Extract(html)
			}
		}()
	}

	// Statistics readers
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				stats := processor.GetStatistics()
				_ = stats.TotalProcessed
				_ = stats.CacheHits
				_ = stats.CacheMisses
			}
		}()
	}

	wg.Wait()

	// Reset statistics concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			processor.ResetStatistics()
		}()
	}

	wg.Wait()
}

// TestConcurrentChannelAuditSink tests concurrent writes to ChannelAuditSink.
func TestConcurrentChannelAuditSink(t *testing.T) {
	sink := NewChannelAuditSink(1000)
	defer sink.Close()

	numGoroutines := 50
	numOperations := 100

	var wg sync.WaitGroup

	// Start consumer
	go func() {
		for range sink.Channel() {
			// Drain channel
		}
	}()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				sink.Write(AuditEntry{
					EventType: AuditEventBlockedTag,
					Message:   fmt.Sprintf("test-%d-%d", id, j),
				})
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrentWriterAuditSink tests concurrent writes to WriterAuditSink.
func TestConcurrentWriterAuditSink(t *testing.T) {
	// Use a discard writer for testing
	sink := NewWriterAuditSink(discardWriter{})
	defer sink.Close()

	numGoroutines := 50
	numOperations := 100

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				sink.Write(AuditEntry{
					EventType: AuditEventBlockedTag,
					Message:   fmt.Sprintf("test-%d-%d", id, j),
				})
			}
		}(i)
	}

	wg.Wait()
}

// discardWriter is a no-op writer for testing.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// TestConcurrentProcessorClose tests concurrent Close calls.
func TestConcurrentProcessorClose(t *testing.T) {
	processor, err := New()
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	numGoroutines := 100
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = processor.Close() // Multiple closes should be safe
		}()
	}

	wg.Wait()
}

// TestConcurrentPoolAccess tests concurrent sync.Pool usage.
func TestConcurrentPoolAccess(t *testing.T) {
	numGoroutines := 100
	numOperations := 500

	var wg sync.WaitGroup

	// Test BuilderPool
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				sb := internal.GetBuilder()
				sb.WriteString("test string")
				_ = sb.String()
				internal.PutBuilder(sb)
			}
		}()
	}

	// Test BufferPool
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				buf := internal.GetBuffer()
				buf.Write([]byte("test bytes"))
				_ = buf.Bytes()
				internal.PutBuffer(buf)
			}
		}()
	}

	// Test Hash128Pool
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				h := internal.GetHash128()
				h.Write([]byte("test data"))
				var sum [16]byte
				h.Sum(sum[:0])
				internal.PutHash128(h)
			}
		}()
	}

	wg.Wait()
}

// TestConcurrentLinkExtraction tests concurrent link extraction.
func TestConcurrentLinkExtraction(t *testing.T) {
	processor, err := New()
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Close()

	html := []byte(`<html><body>
		<a href="https://example.com/1">Link 1</a>
		<a href="https://example.com/2">Link 2</a>
		<img src="image.jpg">
		<script src="script.js"></script>
	</body></html>`)

	numGoroutines := 50
	numOperations := 100

	var wg sync.WaitGroup
	var errorCount atomic.Int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_, err := processor.ExtractAllLinks(html)
				if err != nil {
					errorCount.Add(1)
				}
			}
		}()
	}

	wg.Wait()

	if errorCount.Load() > 0 {
		t.Errorf("Concurrent link extraction had %d errors", errorCount.Load())
	}
}

// TestConcurrentCacheWithTTL tests cache with TTL under concurrent access.
func TestConcurrentCacheWithTTL(t *testing.T) {
	// Short TTL to test expiration
	cache := internal.NewCache(100, 50*time.Millisecond)

	numGoroutines := 30
	numOperations := 100

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(2)

		// Writer
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cache.Set(fmt.Sprintf("key-%d", id), fmt.Sprintf("value-%d-%d", id, j))
			}
		}(i)

		// Reader (may encounter expired entries)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				val := cache.Get(fmt.Sprintf("key-%d", id))
				_ = val // Value may be nil due to expiration
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrentMultiSink tests concurrent writes to MultiSink.
func TestConcurrentMultiSink(t *testing.T) {
	sink1 := NewChannelAuditSink(100)
	sink2 := NewChannelAuditSink(100)
	multiSink := NewMultiSink(sink1, sink2)
	defer multiSink.Close()

	// Start consumers
	go func() {
		for range sink1.Channel() {
		}
	}()
	go func() {
		for range sink2.Channel() {
		}
	}()

	numGoroutines := 30
	numOperations := 50

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				multiSink.Write(AuditEntry{
					EventType: AuditEventBlockedTag,
					Message:   fmt.Sprintf("test-%d-%d", id, j),
				})
			}
		}(i)
	}

	wg.Wait()
}

// BenchmarkConcurrentCache benchmarks concurrent cache operations.
func BenchmarkConcurrentCache(b *testing.B) {
	cache := internal.NewCache(10000, time.Hour)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%1000)
			if i%2 == 0 {
				cache.Set(key, fmt.Sprintf("value-%d", i))
			} else {
				cache.Get(key)
			}
			i++
		}
	})
}

// BenchmarkConcurrentAuditCollector benchmarks concurrent audit recording.
func BenchmarkConcurrentAuditCollector(b *testing.B) {
	config := AuditConfig{Enabled: true}
	collector := NewAuditCollector(config)
	defer collector.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			collector.RecordBlockedTag(fmt.Sprintf("tag-%d", i))
			i++
		}
	})
}

// BenchmarkConcurrentProcessorExtraction benchmarks concurrent extraction.
func BenchmarkConcurrentProcessorExtraction(b *testing.B) {
	processor, _ := New()
	defer processor.Close()

	html := []byte(`<html><body><p>Test content for benchmarking</p></body></html>`)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = processor.Extract(html)
		}
	})
}

// TestConcurrentProcessorCreation tests concurrent processor creation and cleanup.
// This verifies that multiple processors can be created and used concurrently without issues.
func TestConcurrentProcessorCreation(t *testing.T) {
	t.Parallel()

	const numGoroutines = 50

	var wg sync.WaitGroup
	var errorCount atomic.Int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			p, err := New()
			if err != nil {
				errorCount.Add(1)
				return
			}
			defer p.Close()

			// Use the processor
			result, err := p.Extract([]byte("<html><body>Test</body></html>"))
			if err != nil {
				errorCount.Add(1)
				return
			}
			if result == nil {
				errorCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	if errorCount.Load() > 0 {
		t.Errorf("Concurrent processor creation had %d errors", errorCount.Load())
	}
}

// TestConcurrentBatchProcessing tests batch processing with concurrent workers.
func TestConcurrentBatchProcessing(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.WorkerPoolSize = 8

	p, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Prepare multiple HTML documents
	docs := make([][]byte, 50)
	for i := range docs {
		docs[i] = []byte(fmt.Sprintf("<html><body><h1>Doc %d</h1><p>Content</p></body></html>", i))
	}

	// Run multiple batch operations concurrently
	const numConcurrentBatches = 10

	var wg sync.WaitGroup
	var errorCount atomic.Int64

	for i := 0; i < numConcurrentBatches; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			results, err := p.ExtractBatch(docs)
			if err != nil {
				errorCount.Add(1)
				return
			}
			if len(results) != len(docs) {
				errorCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	if errorCount.Load() > 0 {
		t.Errorf("Concurrent batch processing had %d errors", errorCount.Load())
	}
}

// TestMemoryPressure tests behavior under memory pressure with large documents.
func TestMemoryPressure(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping memory pressure test in short mode")
	}

	config := DefaultConfig()
	config.MaxInputSize = 10 * 1024 * 1024 // 10MB

	p, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Create large HTML documents (100KB each)
	const docSize = 100 * 1024
	largeHTML := make([]byte, docSize)
	copy(largeHTML, "<html><body>")
	for i := len("<html><body>"); i < docSize-200; i += 100 {
		copy(largeHTML[i:], "<p>Test content with some text to make it larger.</p>")
	}
	copy(largeHTML[docSize-200:], "</body></html>")

	const numGoroutines = 10
	const docsPerGoroutine = 5

	var wg sync.WaitGroup
	var errorCount atomic.Int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < docsPerGoroutine; j++ {
				_, err := p.Extract(largeHTML)
				if err != nil {
					errorCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	if errorCount.Load() > 0 {
		t.Errorf("Memory pressure test had %d errors", errorCount.Load())
	}
}

// TestCacheEvictionUnderLoad tests cache eviction with concurrent access.
func TestCacheEvictionUnderLoad(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.MaxCacheEntries = 10 // Small cache to trigger evictions

	p, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Create unique HTML content to fill the cache
	const uniqueDocs = 50
	docs := make([][]byte, uniqueDocs)
	for i := range docs {
		docs[i] = []byte(fmt.Sprintf("<html><body><h1>Doc %d</h1><p>Unique content %d</p></body></html>", i, i))
	}

	const numGoroutines = 20

	var wg sync.WaitGroup
	var errorCount atomic.Int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Each goroutine processes all documents, causing cache churn
			for j := range docs {
				result, err := p.Extract(docs[j])
				if err != nil {
					errorCount.Add(1)
					return
				}
				if result == nil {
					errorCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	if errorCount.Load() > 0 {
		t.Errorf("Cache eviction test had %d errors", errorCount.Load())
	}
}
