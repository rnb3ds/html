//go:build examples

package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cybergodev/html"
)

// This example demonstrates configuration options and performance optimization.
// Learn how to tune the library for different use cases.
func main() {
	fmt.Println("=== Configuration & Performance ===\n")

	// ============================================================
	// 1. Configuration Options
	// ============================================================
	fmt.Println("1. Configuration Options")
	fmt.Println("-----------------------")

	config := html.DefaultConfig()
	fmt.Printf("Default configuration:\n")
	fmt.Printf("  MaxInputSize:      %d MB\n", config.MaxInputSize/(1024*1024))
	fmt.Printf("  MaxCacheEntries:   %d\n", config.MaxCacheEntries)
	fmt.Printf("  CacheTTL:          %v\n", config.CacheTTL)
	fmt.Printf("  WorkerPoolSize:    %d\n", config.WorkerPoolSize)
	fmt.Printf("  EnableSanitization: %v\n", config.EnableSanitization)

	// ============================================================
	// 2. Custom Configuration (Struct-Based)
	// ============================================================
	fmt.Println("\n2. Custom Configuration (Struct-Based)")
	fmt.Println("-----------------------")

	customConfig := html.DefaultConfig()
	customConfig.MaxInputSize = 10 * 1024 * 1024  // 10 MB
	customConfig.MaxCacheEntries = 5000
	customConfig.CacheTTL = 2 * time.Hour
	customConfig.WorkerPoolSize = 8

	processor, _ := html.New(customConfig)
	defer processor.Close()

	fmt.Println("Created processor with custom settings")

	// ============================================================
	// 3. Caching Performance
	// ============================================================
	fmt.Println("\n3. Caching Performance")
	fmt.Println("-----------------------")

	sampleHTML := `<article><h1>Test</h1><p>Content for caching test.</p></article>`

	cacheConfig := html.DefaultConfig()
	cacheConfig.MaxCacheEntries = 1000
	cacheConfig.CacheTTL = 5 * time.Minute
	processor2, _ := html.New(cacheConfig)
	defer processor2.Close()

	// First extraction - cache miss
	start := time.Now()
	processor2.Extract([]byte(sampleHTML))
	missTime := time.Since(start)

	// Second extraction - cache hit
	start = time.Now()
	processor2.Extract([]byte(sampleHTML))
	hitTime := time.Since(start)

	fmt.Printf("Cache miss: %v\n", missTime)
	fmt.Printf("Cache hit:  %v\n", hitTime)
	if missTime > 0 && hitTime > 0 {
		fmt.Printf("Speedup:  %.1fx\n", float64(missTime)/float64(hitTime))
	}

	// ============================================================
	// 4. Batch Processing
	// ============================================================
	fmt.Println("\n4. Batch Processing")
	fmt.Println("---------------------")

	docs := make([][]byte, 50)
	for i := 0; i < 50; i++ {
		docs[i] = []byte(fmt.Sprintf(`<article><h1>Doc %d</h1><p>Content %d</p></article>`, i, i))
	}

	batchConfig := html.DefaultConfig()
	batchConfig.WorkerPoolSize = 4
	processor3, _ := html.New(batchConfig)
	defer processor3.Close()

	batchStart := time.Now()
	results, _ := processor3.ExtractBatch(docs, html.DefaultExtractConfig())
	batchTime := time.Since(batchStart)

	fmt.Printf("Processed %d docs in %v\n", len(results), batchTime)

	// ============================================================
	// 5. Batch Processing with Context
	// ============================================================
	fmt.Println("\n5. Batch with Context (cancellation)")
	fmt.Println("----------------------------------------------")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	batchResult := processor3.ExtractBatchWithContext(ctx, docs)
	fmt.Printf("Success: %d, Failed: %d, Cancelled: %d\n",
		batchResult.Success, batchResult.Failed, batchResult.Cancelled)

	for i, r := range batchResult.Results {
		if r != nil {
			fmt.Printf("  [%d] %s\n", i+1, r.Title)
		}
	}

	// ============================================================
	// 6. Concurrent Access (Thread-Safe)
	// ============================================================
	fmt.Println("\n6. Concurrent Access")
	fmt.Println("-------------------------")

	processor4, _ := html.New()
	defer processor4.Close()

	const numGoroutines = 5
	const docsPerGoroutine = 20

	var wg sync.WaitGroup
	concurrentStart := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < docsPerGoroutine; j++ {
				doc := []byte(fmt.Sprintf(`<article><h1>Goroutine %d-%d</h1><p>Content</p></article>`, id, j))
				processor4.Extract(doc)
			}
		}(i)
	}

	wg.Wait()
	concurrentTime := time.Since(concurrentStart)

	fmt.Printf("Processed %d docs concurrently in %v\n", numGoroutines*docsPerGoroutine, concurrentTime)

	// ============================================================
	// 7. Statistics Monitoring
	// ============================================================
	fmt.Println("\n7. Statistics Monitoring")
	fmt.Println("--------------------------")

	processor5, _ := html.New()
	defer processor5.Close()

	// Process some documents
	for i := 0; i < 20; i++ {
		doc := []byte(fmt.Sprintf(`<article><h1>Doc %d</h1><p>Content</p></article>`, i))
		processor5.Extract(doc)
	}

	stats := processor5.GetStatistics()
	fmt.Printf("Total Processed: %d\n", stats.TotalProcessed)
	fmt.Printf("Cache Hits: %d\n", stats.CacheHits)
	fmt.Printf("Cache Misses: %d\n", stats.CacheMisses)
	fmt.Printf("Avg Process Time: %v\n", stats.AverageProcessTime)

	if stats.TotalProcessed > 0 {
		hitRate := float64(stats.CacheHits) / float64(stats.TotalProcessed) * 100
		fmt.Printf("Cache Hit Rate: %.1f%%\n", hitRate)
	}

	// Clear and reset
	processor5.ClearCache()
	processor5.ResetStatistics()
	stats = processor5.GetStatistics()
	fmt.Printf("\nAfter ClearCache/ResetStatistics: %d processed\n", stats.TotalProcessed)

	// ============================================================
	// Summary
	// ============================================================
	fmt.Println("\n=== Performance Recommendations ===")
	fmt.Println()
	fmt.Println("Use Case: Single extraction")
	fmt.Println("  • Use html.Extract() or html.ExtractText() - no processor needed")
	fmt.Println()
	fmt.Println("Use Case: Multiple documents (repeated)")
	fmt.Println("  • Create one Processor, reuse it")
	fmt.Println("  • Cache provides 10-100x speedup")
	fmt.Println()
	fmt.Println("Use Case: Batch processing (10+ docs)")
	fmt.Println("  • Use ExtractBatch() for parallel processing")
	fmt.Println("  • Set WorkerPoolSize to CPU cores")
	fmt.Println()
	fmt.Println("Use Case: Web server")
	fmt.Println("  • Share one Processor across goroutines")
	fmt.Println("  • Thread-safe by design")
}
