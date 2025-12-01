package html_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cybergodev/html"
)

func TestCacheBasic(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><p>Cached content</p></body></html>`

	// First extraction - cache miss
	result1, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	stats := p.GetStatistics()
	if stats.CacheMisses != 1 {
		t.Errorf("CacheMisses = %d, want 1", stats.CacheMisses)
	}
	if stats.CacheHits != 0 {
		t.Errorf("CacheHits = %d, want 0", stats.CacheHits)
	}

	// Second extraction - cache hit
	result2, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	stats = p.GetStatistics()
	if stats.CacheMisses != 1 {
		t.Errorf("CacheMisses = %d, want 1", stats.CacheMisses)
	}
	if stats.CacheHits != 1 {
		t.Errorf("CacheHits = %d, want 1", stats.CacheHits)
	}

	// Results should be identical
	if result1.Text != result2.Text {
		t.Errorf("Cached result differs from original")
	}
}

func TestCacheWithDifferentConfigs(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><img src="test.jpg"><p>Content</p></body></html>`

	// Extract with images
	config1 := html.DefaultExtractConfig()
	config1.PreserveImages = true
	result1, _ := p.Extract(htmlContent, config1)

	// Extract without images - should be different cache entry
	config2 := html.DefaultExtractConfig()
	config2.PreserveImages = false
	result2, _ := p.Extract(htmlContent, config2)

	stats := p.GetStatistics()
	if stats.CacheMisses != 2 {
		t.Errorf("CacheMisses = %d, want 2 (different configs)", stats.CacheMisses)
	}

	if len(result1.Images) == len(result2.Images) {
		t.Error("Results should differ based on config")
	}
}

func TestCacheDisabled(t *testing.T) {
	t.Parallel()

	config := html.Config{
		MaxInputSize: 50 * 1024 * 1024,

		MaxCacheEntries:    0, // Disable cache
		CacheTTL:           time.Hour,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
	}
	p, _ := html.New(config)
	defer p.Close()

	htmlContent := `<html><body><p>No cache</p></body></html>`

	// Extract twice
	p.Extract(htmlContent, html.DefaultExtractConfig())
	p.Extract(htmlContent, html.DefaultExtractConfig())

	stats := p.GetStatistics()
	if stats.CacheHits != 0 {
		t.Errorf("CacheHits = %d, want 0 (cache disabled)", stats.CacheHits)
	}
	if stats.CacheMisses != 2 {
		t.Errorf("CacheMisses = %d, want 2", stats.CacheMisses)
	}
}

func TestCacheTTL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TTL test in short mode")
	}

	config := html.Config{
		MaxInputSize: 50 * 1024 * 1024,

		MaxCacheEntries:    100,
		CacheTTL:           100 * time.Millisecond, // Short TTL for testing
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
	}
	p, _ := html.New(config)
	defer p.Close()

	htmlContent := `<html><body><p>TTL test</p></body></html>`

	// First extraction
	p.Extract(htmlContent, html.DefaultExtractConfig())

	stats := p.GetStatistics()
	if stats.CacheMisses != 1 {
		t.Errorf("CacheMisses = %d, want 1", stats.CacheMisses)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Second extraction - should be cache miss due to expiration
	p.Extract(htmlContent, html.DefaultExtractConfig())

	stats = p.GetStatistics()
	if stats.CacheMisses != 2 {
		t.Errorf("CacheMisses = %d, want 2 (after TTL expiration)", stats.CacheMisses)
	}
}

func TestCacheEviction(t *testing.T) {
	t.Parallel()

	config := html.Config{
		MaxInputSize: 50 * 1024 * 1024,

		MaxCacheEntries:    3, // Small cache for testing eviction
		CacheTTL:           time.Hour,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
	}
	p, _ := html.New(config)
	defer p.Close()

	// Fill cache to capacity
	for i := 0; i < 3; i++ {
		htmlContent := fmt.Sprintf(`<html><body><p>Content %d</p></body></html>`, i)
		_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
	}

	stats := p.GetStatistics()
	if stats.CacheMisses != 3 {
		t.Errorf("CacheMisses = %d, want 3", stats.CacheMisses)
	}

	// Add one more item - should trigger eviction
	htmlContent := `<html><body><p>Content 4</p></body></html>`
	_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	stats = p.GetStatistics()
	if stats.CacheMisses != 4 {
		t.Errorf("CacheMisses = %d, want 4", stats.CacheMisses)
	}
}

func TestCacheClear(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><p>Clear test</p></body></html>`

	// Extract and cache
	p.Extract(htmlContent, html.DefaultExtractConfig())

	stats := p.GetStatistics()
	if stats.CacheMisses != 1 {
		t.Errorf("CacheMisses = %d, want 1", stats.CacheMisses)
	}

	// Clear cache
	p.ClearCache()

	stats = p.GetStatistics()
	if stats.CacheHits != 0 || stats.CacheMisses != 0 {
		t.Error("ClearCache() should reset cache statistics")
	}

	// Extract again - should be cache miss
	p.Extract(htmlContent, html.DefaultExtractConfig())

	stats = p.GetStatistics()
	if stats.CacheMisses != 1 {
		t.Errorf("CacheMisses = %d, want 1 after clear", stats.CacheMisses)
	}
}

func TestCacheConcurrency(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><p>Concurrent cache test</p></body></html>`

	// First extract to populate cache
	_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Initial Extract() failed: %v", err)
	}

	const goroutines = 20
	done := make(chan bool, goroutines)

	// Multiple goroutines accessing same cached content
	for i := 0; i < goroutines; i++ {
		go func() {
			_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
			if err != nil {
				t.Errorf("Concurrent Extract() failed: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}

	stats := p.GetStatistics()
	if stats.TotalProcessed != goroutines+1 {
		t.Errorf("TotalProcessed = %d, want %d", stats.TotalProcessed, goroutines+1)
	}

	// Should have cache hits since content was pre-cached
	if stats.CacheHits < goroutines {
		t.Errorf("CacheHits = %d, want at least %d", stats.CacheHits, goroutines)
	}
}

func TestCacheKeyGeneration(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	// Same content, same config - should hit cache
	htmlContent := `<html><body><p>Test</p></body></html>`
	config := html.DefaultExtractConfig()

	p.Extract(htmlContent, config)
	p.Extract(htmlContent, config)

	stats := p.GetStatistics()
	if stats.CacheHits != 1 {
		t.Errorf("Same content and config should hit cache, got %d hits", stats.CacheHits)
	}

	// Different content - should miss cache
	p.Extract(`<html><body><p>Different</p></body></html>`, config)

	stats = p.GetStatistics()
	if stats.CacheMisses != 2 {
		t.Errorf("Different content should miss cache, got %d misses", stats.CacheMisses)
	}
}
