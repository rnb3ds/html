//go:build examples

package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cybergodev/html"
)

// This example demonstrates performance optimization patterns.
// Learn how to tune the library for batch processing and high-throughput scenarios.
func main() {
	fmt.Println("=== Performance Optimization ===\n")

	// ============================================================
	// 1. Processor Reuse (Critical for Performance)
	// ============================================================
	fmt.Println("1. Processor Reuse")
	fmt.Println("------------------")

	// BAD: Creating processor for each extraction
	fmt.Println("BAD: Creating new processor each time:")
	start := time.Now()
	for i := 0; i < 10; i++ {
		p, _ := html.New()
		p.Extract([]byte("<html><body><p>Test</p></body></html>"))
		p.Close()
	}
	fmt.Printf("  10 extractions: %v\n", time.Since(start))

	// GOOD: Reusing processor
	fmt.Println("GOOD: Reusing processor:")
	processor, _ := html.New()
	defer processor.Close()

	start = time.Now()
	for i := 0; i < 10; i++ {
		processor.Extract([]byte("<html><body><p>Test</p></body></html>"))
	}
	fmt.Printf("  10 extractions: %v\n\n", time.Since(start))

	// ============================================================
	// 2. Caching Benefits
	// ============================================================
	fmt.Println("2. Caching Benefits")
	fmt.Println("-------------------")

	sampleHTML := `<article><h1>Test Article</h1><p>This is content for caching demonstration.</p></article>`

	// First extraction - cache miss
	start = time.Now()
	processor.Extract([]byte(sampleHTML))
	missTime := time.Since(start)

	// Second extraction - cache hit (same content)
	start = time.Now()
	processor.Extract([]byte(sampleHTML))
	hitTime := time.Since(start)

	fmt.Printf("Cache miss: %v\n", missTime)
	fmt.Printf("Cache hit:  %v\n", hitTime)
	if missTime > 0 && hitTime > 0 && missTime > hitTime {
		fmt.Printf("Speedup:    %.1fx\n\n", float64(missTime)/float64(hitTime))
	} else {
		fmt.Printf("(Cache hit is fast)\n\n")
	}

	// ============================================================
	// 3. Batch Processing
	// ============================================================
	fmt.Println("3. Batch Processing")
	fmt.Println("-------------------")

	// Create test documents
	docs := make([][]byte, 100)
	for i := 0; i < 100; i++ {
		docs[i] = []byte(fmt.Sprintf(`<article><h1>Doc %d</h1><p>Content %d</p></article>`, i, i))
	}

	// Sequential processing
	fmt.Println("Sequential (single goroutine):")
	start = time.Now()
	for _, doc := range docs[:20] {
		processor.Extract(doc)
	}
	seqTime := time.Since(start)
	fmt.Printf("  20 docs: %v\n", seqTime)

	// Batch processing with worker pool
	fmt.Println("Batch (worker pool):")
	batchConfig := html.DefaultConfig()
	batchConfig.WorkerPoolSize = 4
	batchProcessor, _ := html.New(batchConfig)
	defer batchProcessor.Close()

	start = time.Now()
	results, _ := batchProcessor.ExtractBatch(docs)
	batchTime := time.Since(start)
	fmt.Printf("  100 docs: %v (%.2f docs/sec)\n\n", batchTime, float64(100)/batchTime.Seconds())

	// Count successful
	success := 0
	for _, r := range results {
		if r != nil {
			success++
		}
	}
	fmt.Printf("  Success: %d/%d\n\n", success, len(results))

	// ============================================================
	// 4. Batch with Context (Cancellation)
	// ============================================================
	fmt.Println("4. Batch with Context")
	fmt.Println("---------------------")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	batchResult := batchProcessor.ExtractBatchWithContext(ctx, docs)
	fmt.Printf("Success: %d, Failed: %d, Cancelled: %d\n",
		batchResult.Success, batchResult.Failed, batchResult.Cancelled)

	// Show first 5 results
	fmt.Println("First 5 results:")
	for i := 0; i < 5 && i < len(batchResult.Results); i++ {
		if batchResult.Results[i] != nil {
			fmt.Printf("  [%d] %s\n", i+1, batchResult.Results[i].Title)
		}
	}
	fmt.Println()

	// ============================================================
	// 5. Concurrent Access (Thread-Safe)
	// ============================================================
	fmt.Println("5. Concurrent Access")
	fmt.Println("--------------------")

	processor2, _ := html.New()
	defer processor2.Close()

	const numGoroutines = 5
	const docsPerGoroutine = 20

	var wg sync.WaitGroup
	start = time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < docsPerGoroutine; j++ {
				doc := []byte(fmt.Sprintf(`<article><h1>Goroutine %d-%d</h1><p>Content</p></article>`, id, j))
				processor2.Extract(doc)
			}
		}(i)
	}

	wg.Wait()
	concurrentTime := time.Since(start)

	fmt.Printf("Processed %d docs concurrently in %v\n", numGoroutines*docsPerGoroutine, concurrentTime)
	fmt.Printf("(%.2f docs/sec)\n\n", float64(numGoroutines*docsPerGoroutine)/concurrentTime.Seconds())

	// ============================================================
	// 6. Configuration Tuning
	// ============================================================
	fmt.Println("6. Configuration Tuning")
	fmt.Println("-----------------------")

	perfConfig := html.DefaultConfig()
	perfConfig.MaxCacheEntries = 5000     // More cache for repeated content
	perfConfig.CacheTTL = 2 * time.Hour   // Longer TTL for stable content
	perfConfig.WorkerPoolSize = 8         // Match CPU cores for CPU-bound work
	perfConfig.CacheCleanup = time.Minute // Frequent cleanup for memory efficiency

	perfProcessor, _ := html.New(perfConfig)
	defer perfProcessor.Close()

	fmt.Println("Performance-optimized config:")
	fmt.Printf("  MaxCacheEntries: %d\n", perfConfig.MaxCacheEntries)
	fmt.Printf("  CacheTTL: %v\n", perfConfig.CacheTTL)
	fmt.Printf("  WorkerPoolSize: %d\n", perfConfig.WorkerPoolSize)
	fmt.Printf("  CacheCleanup: %v\n\n", perfConfig.CacheCleanup)

	// ============================================================
	// 7. Statistics Monitoring
	// ============================================================
	fmt.Println("7. Statistics Monitoring")
	fmt.Println("------------------------")

	// Process some documents
	for i := 0; i < 20; i++ {
		doc := []byte(fmt.Sprintf(`<article><h1>Doc %d</h1><p>Content</p></article>`, i))
		processor2.Extract(doc)
		// Same document again (cache hit)
		processor2.Extract(doc)
	}

	stats := processor2.GetStatistics()
	fmt.Printf("Total Processed: %d\n", stats.TotalProcessed)
	fmt.Printf("Cache Hits: %d\n", stats.CacheHits)
	fmt.Printf("Cache Misses: %d\n", stats.CacheMisses)
	fmt.Printf("Avg Process Time: %v\n", stats.AverageProcessTime)

	if stats.TotalProcessed > 0 {
		hitRate := float64(stats.CacheHits) / float64(stats.TotalProcessed) * 100
		fmt.Printf("Cache Hit Rate: %.1f%%\n", hitRate)
	}

	// Clear and reset
	processor2.ClearCache()
	processor2.ResetStatistics()
	stats = processor2.GetStatistics()
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
	fmt.Println("  • Cache provides significant speedup for repeated content")
	fmt.Println()
	fmt.Println("Use Case: Batch processing (10+ docs)")
	fmt.Println("  • Use ExtractBatch() for parallel processing")
	fmt.Println("  • Set WorkerPoolSize to match CPU cores")
	fmt.Println()
	fmt.Println("Use Case: Web server")
	fmt.Println("  • Share one Processor across goroutines")
	fmt.Println("  • Thread-safe by design")
}
