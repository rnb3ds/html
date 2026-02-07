package html_test

// concurrency_test.go - Comprehensive concurrency and stress tests
// Tests for thread safety, race conditions, and memory pressure

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cybergodev/html"
)

// TestConcurrentExtraction tests that multiple goroutines can safely use the same processor
func TestConcurrentExtraction(t *testing.T) {
	t.Parallel()

	p, err := html.New()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	htmlContent := []byte(`
		<html>
			<head><title>Test</title></head>
			<body>
				<h1>Concurrent Test</h1>
				<p>Content for concurrent extraction testing.</p>
			</body>
		</html>
	`)

	const numGoroutines = 100
	const iterationsPerGoroutine = 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*iterationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterationsPerGoroutine; j++ {
				result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
				if err != nil {
					errors <- fmt.Errorf("goroutine %d iteration %d: %w", id, j, err)
					return
				}
				if result == nil {
					errors <- fmt.Errorf("goroutine %d iteration %d: nil result", id, j)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent extraction error: %v", err)
	}
}

// TestConcurrentProcessorCreation tests concurrent processor creation and cleanup
func TestConcurrentProcessorCreation(t *testing.T) {
	t.Parallel()

	const numGoroutines = 50

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			p, err := html.New()
			if err != nil {
				t.Errorf("Goroutine %d: New() failed: %v", id, err)
				return
			}
			defer p.Close()

			// Use the processor
			result, err := p.Extract([]byte("<html><body>Test</body></html>"), html.DefaultExtractConfig())
			if err != nil {
				t.Errorf("Goroutine %d: Extract() failed: %v", id, err)
				return
			}
			if result == nil {
				t.Errorf("Goroutine %d: nil result", id)
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrentCacheAccess tests cache behavior under concurrent access
func TestConcurrentCacheAccess(t *testing.T) {
	t.Parallel()

	config := html.DefaultConfig()
	config.MaxCacheEntries = 100

	p, err := html.New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Create different HTML content for each goroutine
	htmlContents := make([][]byte, 20)
	for i := range htmlContents {
		htmlContents[i] = []byte(fmt.Sprintf("<html><body><h1>Content %d</h1><p>Test content</p></body></html>", i))
	}

	const numGoroutines = 50
	const iterationsPerGoroutine = 20

	var wg sync.WaitGroup
	successCount := int64(0)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterationsPerGoroutine; j++ {
				// Use different content to exercise cache
				contentIdx := (id + j) % len(htmlContents)
				result, err := p.Extract(htmlContents[contentIdx], html.DefaultExtractConfig())
				if err != nil {
					t.Errorf("Goroutine %d: Extract() failed: %v", id, err)
					return
				}
				if result != nil {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	expected := int64(numGoroutines * iterationsPerGoroutine)
	if got := atomic.LoadInt64(&successCount); got != expected {
		t.Errorf("Expected %d successful extractions, got %d", expected, got)
	}
}

// TestConcurrentBatchProcessing tests batch processing with concurrent workers
func TestConcurrentBatchProcessing(t *testing.T) {
	t.Parallel()

	config := html.DefaultConfig()
	config.WorkerPoolSize = 8

	p, err := html.New(config)
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
	for i := 0; i < numConcurrentBatches; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			results, err := p.ExtractBatch(docs, html.DefaultExtractConfig())
			if err != nil {
				t.Errorf("Batch %d: ExtractBatch() failed: %v", id, err)
				return
			}
			if len(results) != len(docs) {
				t.Errorf("Batch %d: expected %d results, got %d", id, len(docs), len(results))
			}
		}(i)
	}

	wg.Wait()
}

// TestProcessorCloseDuringOperations tests closing processor while operations are in progress
func TestProcessorCloseDuringOperations(t *testing.T) {
	t.Parallel()

	config := html.DefaultConfig()
	p, err := html.New(config)
	if err != nil {
		t.Fatal(err)
	}

	htmlContent := []byte("<html><body>Test content</body></html>")

	// Start goroutines that will try to use the processor
	const numGoroutines = 20
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Small delay to ensure Close() is called mid-operation
			time.Sleep(time.Microsecond * time.Duration(id*10))

			_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
			if err == html.ErrProcessorClosed {
				// Expected - processor was closed
				return
			}
			if err != nil {
				t.Errorf("Goroutine %d: unexpected error: %v", id, err)
			}
		}(i)
	}

	// Close the processor after a short delay
	time.Sleep(100 * time.Microsecond)
	p.Close()

	wg.Wait()
}

// TestMemoryPressure tests behavior under memory pressure with large documents
func TestMemoryPressure(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping memory pressure test in short mode")
	}

	config := html.DefaultConfig()
	config.MaxInputSize = 10 * 1024 * 1024 // 10MB

	p, err := html.New(config)
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
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < docsPerGoroutine; j++ {
				_, err := p.Extract(largeHTML, html.DefaultExtractConfig())
				if err != nil {
					t.Errorf("Goroutine %d doc %d: Extract() failed: %v", id, j, err)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestCacheEvictionUnderLoad tests cache eviction with concurrent access
func TestCacheEvictionUnderLoad(t *testing.T) {
	t.Parallel()

	config := html.DefaultConfig()
	config.MaxCacheEntries = 10 // Small cache to trigger evictions

	p, err := html.New(config)
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
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Each goroutine processes all documents, causing cache churn
			for j := range docs {
				result, err := p.Extract(docs[j], html.DefaultExtractConfig())
				if err != nil {
					t.Errorf("Goroutine %d doc %d: Extract() failed: %v", id, j, err)
					return
				}
				if result == nil {
					t.Errorf("Goroutine %d doc %d: nil result", id, j)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestStatisticsConcurrency tests GetStatistics under concurrent access
func TestStatisticsConcurrency(t *testing.T) {
	t.Parallel()

	p, err := html.New()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	htmlContent := []byte("<html><body>Test content</body></html>")

	const numGoroutines = 50
	var wg sync.WaitGroup

	// Mix of extraction and statistics queries
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if id%2 == 0 {
					// Even goroutines do extraction
					_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
					if err != nil {
						t.Errorf("Goroutine %d: Extract() failed: %v", id, err)
					}
				} else {
					// Odd goroutines query statistics
					stats := p.GetStatistics()
					// Statistics is never nil, it returns a struct
					_ = stats
				}
			}
		}(i)
	}

	wg.Wait()
}

// BenchmarkConcurrentExtraction benchmarks concurrent extraction performance
func BenchmarkConcurrentExtraction(b *testing.B) {
	p, _ := html.New()
	defer p.Close()

	htmlContent := []byte(`
		<html>
			<head><title>Test</title></head>
			<body>
				<h1>Concurrent Benchmark</h1>
				<p>Content for benchmarking.</p>
			</body>
		</html>
	`)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = p.Extract(htmlContent, html.DefaultExtractConfig())
		}
	})
}

// BenchmarkConcurrentCacheAccess benchmarks cache under concurrent load
func BenchmarkConcurrentCacheAccess(b *testing.B) {
	config := html.DefaultConfig()
	config.MaxCacheEntries = 100

	p, _ := html.New(config)
	defer p.Close()

	// Pre-warm cache
	htmlContents := make([][]byte, 10)
	for i := range htmlContents {
		htmlContents[i] = []byte(fmt.Sprintf("<html><body><h1>Content %d</h1></body></html>", i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			content := htmlContents[i%len(htmlContents)]
			_, _ = p.Extract(content, html.DefaultExtractConfig())
			i++
		}
	})
}
