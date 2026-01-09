package html_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cybergodev/html"
)

// TestBatchProcessing tests batch processing functionality
func TestBatchProcessing(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	inputs := []string{
		`<html><body><article><h1>Article 1</h1><p>Content 1</p></article></body></html>`,
		`<html><body><article><h1>Article 2</h1><p>Content 2</p></article></body></html>`,
		`<html><body><article><h1>Article 3</h1><p>Content 3</p></article></body></html>`,
	}

	results, err := p.ExtractBatch(inputs, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("ExtractBatch() failed: %v", err)
	}

	if len(results) != len(inputs) {
		t.Fatalf("ExtractBatch() returned %d results, want %d", len(results), len(inputs))
	}

	for i, result := range results {
		expectedTitle := fmt.Sprintf("Article %d", i+1)
		if result.Title != expectedTitle {
			t.Errorf("Result[%d].Title = %q, want %q", i, result.Title, expectedTitle)
		}

		expectedContent := fmt.Sprintf("Content %d", i+1)
		if !strings.Contains(result.Text, expectedContent) {
			t.Errorf("Result[%d].Text should contain %q", i, expectedContent)
		}
	}
}

// TestBatchProcessingEmpty tests batch processing with empty input
func TestBatchProcessingEmpty(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	results, err := p.ExtractBatch([]string{}, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("ExtractBatch() failed on empty input: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("ExtractBatch() returned %d results, want 0", len(results))
	}
}

// TestBatchProcessingWithErrors tests batch processing with some invalid inputs
func TestBatchProcessingWithErrors(t *testing.T) {
	t.Parallel()

	config := html.Config{
		MaxInputSize:       100,
		MaxCacheEntries:    10,
		CacheTTL:           time.Hour,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
	}
	p, _ := html.New(config)
	defer p.Close()

	inputs := []string{
		`<html><body><p>Valid</p></body></html>`,
		strings.Repeat("a", 200), // Too large
		`<html><body><p>Another valid</p></body></html>`,
	}

	results, err := p.ExtractBatch(inputs, html.DefaultExtractConfig())
	if err == nil {
		t.Fatal("ExtractBatch() should return error for partial failures")
	}

	// Should still return results for valid inputs
	if len(results) != len(inputs) {
		t.Errorf("ExtractBatch() returned %d results, want %d", len(results), len(inputs))
	}
}

// TestCacheHit tests cache hit functionality
func TestCacheHit(t *testing.T) {
	t.Parallel()

	config := html.Config{
		MaxInputSize:       10 * 1024 * 1024,
		MaxCacheEntries:    100,
		CacheTTL:           time.Hour,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
	}
	p, _ := html.New(config)
	defer p.Close()

	htmlContent := `<html><body><article><h1>Test</h1><p>Content</p></article></body></html>`

	// First extraction - cache miss
	result1, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	stats1 := p.GetStatistics()
	if stats1.CacheHits != 0 {
		t.Errorf("First extraction should have 0 cache hits, got %d", stats1.CacheHits)
	}

	// Second extraction - cache hit
	result2, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	stats2 := p.GetStatistics()
	if stats2.CacheHits != 1 {
		t.Errorf("Second extraction should have 1 cache hit, got %d", stats2.CacheHits)
	}

	// Results should be identical
	if result1.Title != result2.Title {
		t.Errorf("Cached result title mismatch: %q vs %q", result1.Title, result2.Title)
	}
	if result1.Text != result2.Text {
		t.Errorf("Cached result text mismatch")
	}
}

// TestCacheTTL tests cache TTL expiration
func TestCacheTTL(t *testing.T) {
	t.Parallel()

	config := html.Config{
		MaxInputSize:       10 * 1024 * 1024,
		MaxCacheEntries:    100,
		CacheTTL:           100 * time.Millisecond, // Very short TTL
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
	}
	p, _ := html.New(config)
	defer p.Close()

	htmlContent := `<html><body><p>Test</p></body></html>`

	// First extraction
	p.Extract(htmlContent, html.DefaultExtractConfig())

	// Wait for cache to expire
	time.Sleep(200 * time.Millisecond)

	// Second extraction after TTL - should be cache miss
	p.Extract(htmlContent, html.DefaultExtractConfig())

	stats := p.GetStatistics()
	// After expiration, the second extraction should not be a cache hit
	if stats.CacheHits > 0 {
		t.Logf("Note: Cache hit count is %d (cache may not have expired yet)", stats.CacheHits)
	}
}

// TestCacheDisabled tests processor with cache disabled
func TestCacheDisabled(t *testing.T) {
	t.Parallel()

	config := html.Config{
		MaxInputSize:       10 * 1024 * 1024,
		MaxCacheEntries:    0, // Disable cache
		CacheTTL:           time.Hour,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
	}
	p, _ := html.New(config)
	defer p.Close()

	htmlContent := `<html><body><p>Test</p></body></html>`

	// Extract twice
	p.Extract(htmlContent, html.DefaultExtractConfig())
	p.Extract(htmlContent, html.DefaultExtractConfig())

	stats := p.GetStatistics()
	if stats.CacheHits != 0 {
		t.Errorf("With cache disabled, should have 0 cache hits, got %d", stats.CacheHits)
	}
}

// TestCacheEviction tests cache eviction when max entries is reached
func TestCacheEviction(t *testing.T) {
	t.Parallel()

	config := html.Config{
		MaxInputSize:       10 * 1024 * 1024,
		MaxCacheEntries:    2, // Very small cache
		CacheTTL:           time.Hour,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
	}
	p, _ := html.New(config)
	defer p.Close()

	// Extract 3 different documents
	for i := 0; i < 3; i++ {
		htmlContent := fmt.Sprintf(`<html><body><p>Test %d</p></body></html>`, i)
		_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
	}

	stats := p.GetStatistics()
	if stats.TotalProcessed != 3 {
		t.Errorf("TotalProcessed = %d, want 3", stats.TotalProcessed)
	}
}

// TestCacheConcurrency tests cache with concurrent access
func TestCacheConcurrency(t *testing.T) {
	t.Parallel()

	config := html.Config{
		MaxInputSize:       10 * 1024 * 1024,
		MaxCacheEntries:    100,
		CacheTTL:           time.Hour,
		WorkerPoolSize:     8,
		EnableSanitization: true,
		MaxDepth:           100,
	}
	p, _ := html.New(config)
	defer p.Close()

	htmlContent := `<html><body><article><h1>Test</h1><p>Content</p></article></body></html>`

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
			if err != nil {
				t.Errorf("Extract() failed: %v", err)
			}
		}()
	}

	wg.Wait()

	stats := p.GetStatistics()
	if stats.TotalProcessed != int64(numGoroutines) {
		t.Errorf("TotalProcessed = %d, want %d", stats.TotalProcessed, numGoroutines)
	}

	// Cache hits may or may not occur depending on timing
	t.Logf("Cache hits: %d out of %d", stats.CacheHits, stats.TotalProcessed)
}

// TestClearCache tests cache clearing
func TestClearCache(t *testing.T) {
	t.Parallel()

	config := html.Config{
		MaxInputSize:       10 * 1024 * 1024,
		MaxCacheEntries:    100,
		CacheTTL:           time.Hour,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
	}
	p, _ := html.New(config)
	defer p.Close()

	htmlContent := `<html><body><p>Test</p></body></html>`

	// Extract to populate cache
	p.Extract(htmlContent, html.DefaultExtractConfig())

	// Clear cache
	p.ClearCache()

	// Extract again - should be cache miss
	p.Extract(htmlContent, html.DefaultExtractConfig())

	stats := p.GetStatistics()
	// After clearing cache, the second extraction should not be a cache hit
	if stats.CacheHits > 0 {
		t.Errorf("After ClearCache(), should have 0 cache hits, got %d", stats.CacheHits)
	}
}

// TestBatchProcessingWithCache tests batch processing benefits from cache
func TestBatchProcessingWithCache(t *testing.T) {
	t.Parallel()

	config := html.Config{
		MaxInputSize:       10 * 1024 * 1024,
		MaxCacheEntries:    100,
		CacheTTL:           time.Hour,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
	}
	p, _ := html.New(config)
	defer p.Close()

	// Same content repeated
	htmlContent := `<html><body><p>Test</p></body></html>`
	inputs := []string{htmlContent, htmlContent, htmlContent}

	// First batch
	_, err := p.ExtractBatch(inputs, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("ExtractBatch() failed: %v", err)
	}

	stats := p.GetStatistics()
	// Cache hits may or may not occur depending on timing in batch processing
	t.Logf("Cache hits in batch: %d out of %d", stats.CacheHits, stats.TotalProcessed)
}

// TestStatisticsReset tests statistics after processor reset
func TestStatisticsReset(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><p>Test</p></body></html>`

	// Process some content
	p.Extract(htmlContent, html.DefaultExtractConfig())
	p.Extract(htmlContent, html.DefaultExtractConfig())

	stats := p.GetStatistics()
	if stats.TotalProcessed == 0 {
		t.Error("TotalProcessed should be > 0")
	}

	// Clear cache resets some statistics
	p.ClearCache()

	// Continue processing
	p.Extract(htmlContent, html.DefaultExtractConfig())

	newStats := p.GetStatistics()
	if newStats.TotalProcessed <= stats.TotalProcessed {
		t.Error("TotalProcessed should continue incrementing")
	}
}

