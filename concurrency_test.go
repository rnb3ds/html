package html_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cybergodev/html"
)

// TestConcurrentExtractSameContent tests multiple goroutines extracting the same content
func TestConcurrentExtractSameContent(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><article><h1>Title</h1><p>Content with some text</p></article></body></html>`
	config := html.DefaultExtractConfig()

	const goroutines = 100
	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			result, err := p.Extract(htmlContent, config)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: %w", id, err)
				return
			}
			if result.Title != "Title" {
				errors <- fmt.Errorf("goroutine %d: wrong title %q", id, result.Title)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	stats := p.GetStatistics()
	if stats.TotalProcessed != goroutines {
		t.Errorf("TotalProcessed = %d, want %d", stats.TotalProcessed, goroutines)
	}
}

// TestConcurrentExtractDifferentContent tests multiple goroutines extracting different content
func TestConcurrentExtractDifferentContent(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	const goroutines = 50
	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			htmlContent := fmt.Sprintf(`<html><body><h1>Title %d</h1><p>Content %d</p></body></html>`, id, id)
			result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: %w", id, err)
				return
			}
			expectedTitle := fmt.Sprintf("Title %d", id)
			if result.Title != expectedTitle {
				errors <- fmt.Errorf("goroutine %d: title = %q, want %q", id, result.Title, expectedTitle)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

// TestConcurrentExtractWithDifferentConfigs tests concurrent extraction with different configs
func TestConcurrentExtractWithDifferentConfigs(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><img src="test.jpg"><a href="link.html">Link</a><p>Content</p></body></html>`

	const goroutines = 50
	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			config := html.DefaultExtractConfig()
			// Alternate between different configs
			config.PreserveImages = (id % 2) == 0
			config.PreserveLinks = (id % 3) == 0

			result, err := p.Extract(htmlContent, config)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: %w", id, err)
				return
			}

			if config.PreserveImages && len(result.Images) == 0 {
				errors <- fmt.Errorf("goroutine %d: expected images", id)
			}
			if !config.PreserveImages && len(result.Images) > 0 {
				errors <- fmt.Errorf("goroutine %d: unexpected images", id)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

// TestConcurrentBatchOperations tests concurrent batch operations
func TestConcurrentBatchOperations(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	const batches = 10
	const itemsPerBatch = 5
	var wg sync.WaitGroup
	errors := make(chan error, batches)

	for i := 0; i < batches; i++ {
		wg.Add(1)
		go func(batchID int) {
			defer wg.Done()

			htmlContents := make([]string, itemsPerBatch)
			for j := 0; j < itemsPerBatch; j++ {
				htmlContents[j] = fmt.Sprintf(`<html><body><p>Batch %d Item %d</p></body></html>`, batchID, j)
			}

			results, err := p.ExtractBatch(htmlContents, html.DefaultExtractConfig())
			if err != nil {
				errors <- fmt.Errorf("batch %d: %w", batchID, err)
				return
			}

			if len(results) != itemsPerBatch {
				errors <- fmt.Errorf("batch %d: got %d results, want %d", batchID, len(results), itemsPerBatch)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

// TestConcurrentCacheOperations tests concurrent cache operations
func TestConcurrentCacheOperations(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><p>Cache test</p></body></html>`

	// Pre-populate cache
	_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Initial extract failed: %v", err)
	}

	const goroutines = 100
	var wg sync.WaitGroup
	var successCount atomic.Int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
			if err == nil {
				successCount.Add(1)
			}
		}()
	}

	wg.Wait()

	if successCount.Load() != goroutines {
		t.Errorf("Success count = %d, want %d", successCount.Load(), goroutines)
	}

	stats := p.GetStatistics()
	if stats.CacheHits < goroutines {
		t.Errorf("CacheHits = %d, want at least %d", stats.CacheHits, goroutines)
	}
}

// TestConcurrentClearCache tests concurrent cache clearing
func TestConcurrentClearCache(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	const goroutines = 50
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			htmlContent := fmt.Sprintf(`<html><body><p>Content %d</p></body></html>`, id)
			p.Extract(htmlContent, html.DefaultExtractConfig())

			if id%10 == 0 {
				p.ClearCache()
			}
		}(i)
	}

	wg.Wait()

	// Should not crash or deadlock
	stats := p.GetStatistics()
	if stats.TotalProcessed != goroutines {
		t.Errorf("TotalProcessed = %d, want %d", stats.TotalProcessed, goroutines)
	}
}

// TestConcurrentGetStatistics tests concurrent statistics access
func TestConcurrentGetStatistics(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	const goroutines = 100
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			if id%2 == 0 {
				htmlContent := fmt.Sprintf(`<html><body><p>Content %d</p></body></html>`, id)
				p.Extract(htmlContent, html.DefaultExtractConfig())
			} else {
				p.GetStatistics()
			}
		}(i)
	}

	wg.Wait()

	stats := p.GetStatistics()
	if stats.TotalProcessed < 0 {
		t.Errorf("Invalid TotalProcessed: %d", stats.TotalProcessed)
	}
}

// TestConcurrentClose tests concurrent close operations
func TestConcurrentClose(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()

	const goroutines = 10
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.Close()
		}()
	}

	wg.Wait()

	// Should not crash or deadlock
	_, err := p.ExtractWithDefaults("<html><body>Test</body></html>")
	if err == nil {
		t.Error("Extract should fail after Close()")
	}
}

// TestConcurrentExtractAndClose tests extraction while closing
func TestConcurrentExtractAndClose(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()

	const goroutines = 50
	var wg sync.WaitGroup
	var extractCount atomic.Int64
	var closeCount atomic.Int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			if id == goroutines/2 {
				time.Sleep(10 * time.Millisecond)
				p.Close()
				closeCount.Add(1)
			} else {
				htmlContent := fmt.Sprintf(`<html><body><p>Content %d</p></body></html>`, id)
				_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
				if err == nil {
					extractCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	// Some extractions should succeed before close, some should fail after
	if extractCount.Load() == 0 {
		t.Error("Expected some successful extractions before close")
	}
	if closeCount.Load() != 1 {
		t.Errorf("Close count = %d, want 1", closeCount.Load())
	}
}

// TestConcurrentCacheEviction tests concurrent cache eviction
func TestConcurrentCacheEviction(t *testing.T) {
	t.Parallel()

	config := html.Config{
		MaxInputSize:       50 * 1024 * 1024,
		ProcessingTimeout:  30 * time.Second,
		MaxCacheEntries:    10, // Small cache to trigger eviction
		CacheTTL:           time.Hour,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
	}
	p, _ := html.New(config)
	defer p.Close()

	const goroutines = 50
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			htmlContent := fmt.Sprintf(`<html><body><p>Content %d</p></body></html>`, id)
			p.Extract(htmlContent, html.DefaultExtractConfig())
		}(i)
	}

	wg.Wait()

	stats := p.GetStatistics()
	if stats.TotalProcessed != goroutines {
		t.Errorf("TotalProcessed = %d, want %d", stats.TotalProcessed, goroutines)
	}
}

// TestConcurrentCacheTTLExpiration tests concurrent cache TTL expiration
func TestConcurrentCacheTTLExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TTL test in short mode")
	}

	config := html.Config{
		MaxInputSize:       50 * 1024 * 1024,
		ProcessingTimeout:  30 * time.Second,
		MaxCacheEntries:    100,
		CacheTTL:           50 * time.Millisecond,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
	}
	p, _ := html.New(config)
	defer p.Close()

	htmlContent := `<html><body><p>TTL test</p></body></html>`

	const goroutines = 20
	var wg sync.WaitGroup

	// First wave - populate cache
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.Extract(htmlContent, html.DefaultExtractConfig())
		}()
	}
	wg.Wait()

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Second wave - should miss cache
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.Extract(htmlContent, html.DefaultExtractConfig())
		}()
	}
	wg.Wait()

	stats := p.GetStatistics()
	if stats.TotalProcessed != goroutines*2 {
		t.Errorf("TotalProcessed = %d, want %d", stats.TotalProcessed, goroutines*2)
	}
}

// TestStressTest performs a stress test with mixed operations
func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	const duration = 2 * time.Second
	const workers = 20

	var wg sync.WaitGroup
	stop := make(chan struct{})

	var operations atomic.Int64
	var errors atomic.Int64

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for {
				select {
				case <-stop:
					return
				default:
					htmlContent := fmt.Sprintf(`<html><body><h1>Worker %d</h1><p>Content</p></body></html>`, id%10)
					_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
					if err != nil {
						errors.Add(1)
					}
					operations.Add(1)

					if id%5 == 0 {
						p.GetStatistics()
					}
					if id%10 == 0 {
						p.ClearCache()
					}
				}
			}
		}(i)
	}

	// Run for specified duration
	time.Sleep(duration)
	close(stop)
	wg.Wait()

	t.Logf("Completed %d operations with %d errors in %v", operations.Load(), errors.Load(), duration)

	if errors.Load() > 0 {
		t.Errorf("Stress test had %d errors", errors.Load())
	}

	stats := p.GetStatistics()
	if stats.TotalProcessed == 0 {
		t.Error("No operations were processed")
	}
}

// TestRaceConditionCacheExpiration tests race conditions during cache expiration
func TestRaceConditionCacheExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	config := html.Config{
		MaxInputSize:       50 * 1024 * 1024,
		ProcessingTimeout:  30 * time.Second,
		MaxCacheEntries:    100,
		CacheTTL:           50 * time.Millisecond,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
	}
	p, _ := html.New(config)
	defer p.Close()

	htmlContent := `<html><body><p>Race test</p></body></html>`

	const goroutines = 100
	var wg sync.WaitGroup
	var successCount atomic.Int64
	var errorCount atomic.Int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
				if err != nil {
					errorCount.Add(1)
				} else {
					successCount.Add(1)
				}
				if id%3 == 0 {
					time.Sleep(time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()

	if errorCount.Load() > 0 {
		t.Errorf("Race condition test had %d errors", errorCount.Load())
	}

	t.Logf("Completed %d successful operations", successCount.Load())
}

// TestConcurrentCacheEvictionUnderPressure tests cache eviction with high concurrency
func TestConcurrentCacheEvictionUnderPressure(t *testing.T) {
	t.Parallel()

	config := html.Config{
		MaxInputSize:       50 * 1024 * 1024,
		ProcessingTimeout:  30 * time.Second,
		MaxCacheEntries:    5,
		CacheTTL:           time.Hour,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
	}
	p, _ := html.New(config)
	defer p.Close()

	const goroutines = 50
	const iterations = 20
	var wg sync.WaitGroup
	var errorCount atomic.Int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				htmlContent := fmt.Sprintf(`<html><body><p>Content %d-%d</p></body></html>`, id, j)
				_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
				if err != nil {
					errorCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	if errorCount.Load() > 0 {
		t.Errorf("Cache eviction test had %d errors", errorCount.Load())
	}

	stats := p.GetStatistics()
	expectedTotal := int64(goroutines * iterations)
	if stats.TotalProcessed != expectedTotal {
		t.Errorf("TotalProcessed = %d, want %d", stats.TotalProcessed, expectedTotal)
	}
}

// TestAtomicStatisticsConsistency verifies statistics remain consistent under concurrency
func TestAtomicStatisticsConsistency(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	const goroutines = 50
	var wg sync.WaitGroup

	htmlContent := `<html><body><p>Stats test</p></body></html>`

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.Extract(htmlContent, html.DefaultExtractConfig())
		}()
	}

	wg.Wait()

	stats := p.GetStatistics()

	if stats.TotalProcessed != stats.CacheHits+stats.CacheMisses {
		t.Errorf("Statistics inconsistent: TotalProcessed=%d, CacheHits=%d, CacheMisses=%d",
			stats.TotalProcessed, stats.CacheHits, stats.CacheMisses)
	}

	if stats.TotalProcessed != goroutines {
		t.Errorf("TotalProcessed = %d, want %d", stats.TotalProcessed, goroutines)
	}
}

// TestHighContentionScenario simulates high contention on same cache entries
func TestHighContentionScenario(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContents := []string{
		`<html><body><p>Content A</p></body></html>`,
		`<html><body><p>Content B</p></body></html>`,
		`<html><body><p>Content C</p></body></html>`,
	}

	const goroutines = 100
	var wg sync.WaitGroup
	var errorCount atomic.Int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				content := htmlContents[j%len(htmlContents)]
				_, err := p.Extract(content, html.DefaultExtractConfig())
				if err != nil {
					errorCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	if errorCount.Load() > 0 {
		t.Errorf("High contention test had %d errors", errorCount.Load())
	}

	stats := p.GetStatistics()
	hitRatio := float64(stats.CacheHits) / float64(stats.TotalProcessed)
	if hitRatio < 0.9 {
		t.Logf("Cache hit ratio: %.2f%% (expected >90%%)", hitRatio*100)
	}
}
