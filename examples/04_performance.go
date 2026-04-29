//go:build example_04

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
	fmt.Println("=== Performance Optimization ===")
	fmt.Println()

	// ============================================================
	// 1. Processor Reuse (Critical for Performance)
	// ============================================================
	fmt.Println("1. Processor Reuse")
	fmt.Println("------------------")

	// BAD: Creating processor for each extraction
	fmt.Println("BAD: Creating new processor each time:")
	const benchDoc = `<html><body><article><h1>Go Performance Guide</h1><p>This article covers performance optimization techniques for Go applications, including profiling, benchmarking, and memory management strategies.</p><p>Understanding how the Go scheduler works is essential for writing efficient concurrent programs that make the most of available CPU resources.</p><p>Memory allocation patterns significantly impact GC pressure. Prefer stack-allocated values and sync.Pool for heap-heavy workloads.</p></article></body></html>`
	start := time.Now()
	for i := 0; i < 2000; i++ {
		p, _ := html.New()
		p.Extract([]byte(benchDoc))
		p.Close()
	}
	fmt.Printf("  2000 extractions: %v\n", time.Since(start))

	// GOOD: Reusing processor
	fmt.Println("GOOD: Reusing processor:")
	processor, _ := html.New()
	defer processor.Close()

	start = time.Now()
	for i := 0; i < 2000; i++ {
		processor.Extract([]byte(benchDoc))
	}
	fmt.Printf("  2000 extractions: %v\n\n", time.Since(start))

	// ============================================================
	// 2. Caching Benefits
	// ============================================================
	fmt.Println("2. Caching Benefits")
	fmt.Println("-------------------")

	// Cold cache: 100 unique documents (all cache misses)
	start = time.Now()
	for i := 0; i < 100; i++ {
		doc := []byte(fmt.Sprintf(`<html><body><article><h1>Cache Test %d</h1><p>This is a longer article used for caching demonstration. It contains multiple paragraphs to produce measurable extraction times on modern hardware. The cache stores results by content hash, so extracting the same document twice returns the cached result instantly.</p><p>Cache hits reduce both CPU time and memory allocations, which is especially valuable in web services processing repeated content.</p></article></body></html>`, i))
		processor.Extract(doc)
	}
	missTime := time.Since(start)

	// Warm cache: same document repeated (all cache hits)
	warmDoc := []byte(`<html><body><article><h1>Cache Warm</h1><p>This is a longer article used for caching demonstration. It contains multiple paragraphs to produce measurable extraction times on modern hardware. The cache stores results by content hash, so extracting the same document twice returns the cached result instantly.</p><p>Cache hits reduce both CPU time and memory allocations, which is especially valuable in web services processing repeated content.</p></article></body></html>`)
	processor.Extract(warmDoc) // populate cache
	start = time.Now()
	for i := 0; i < 10000; i++ {
		processor.Extract(warmDoc)
	}
	hitTime := time.Since(start)

	fmt.Printf("100 unique docs (all misses):   %v\n", missTime)
	fmt.Printf("10000 same docs  (all hits):  %v\n", hitTime)
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
	batchCfg := html.DefaultConfig()
	batchCfg.WorkerPoolSize = 4
	batchProcessor, _ := html.New(batchCfg)
	defer batchProcessor.Close()

	start = time.Now()
	batchResult := batchProcessor.ExtractBatch(docs)
	batchTime := time.Since(start)
	fmt.Printf("  100 docs: %v (%.2f docs/sec)\n\n", batchTime, float64(100)/batchTime.Seconds())

	fmt.Printf("  Success: %d/%d\n\n", batchResult.Success, len(batchResult.Results))

	// ============================================================
	// 4. Batch with Context (Cancellation)
	// ============================================================
	fmt.Println("4. Batch with Context")
	fmt.Println("---------------------")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctxResult := batchProcessor.ExtractBatchWithContext(ctx, docs)
	fmt.Printf("Success: %d, Failed: %d, Cancelled: %d\n",
		ctxResult.Success, ctxResult.Failed, ctxResult.Cancelled)

	// Show first 5 results
	fmt.Println("First 5 results:")
	for i := 0; i < 5 && i < len(ctxResult.Results); i++ {
		if ctxResult.Results[i] != nil {
			fmt.Printf("  [%d] %s\n", i+1, ctxResult.Results[i].Title)
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

	perfCfg := html.DefaultConfig()
	perfCfg.MaxCacheEntries = 5000     // More cache for repeated content
	perfCfg.CacheTTL = 2 * time.Hour   // Longer TTL for stable content
	perfCfg.WorkerPoolSize = 8         // Match CPU cores for CPU-bound work
	perfCfg.CacheCleanup = time.Minute // Frequent cleanup for memory efficiency

	perfProcessor, _ := html.New(perfCfg)
	defer perfProcessor.Close()

	fmt.Println("Performance-optimized config:")
	fmt.Printf("  MaxCacheEntries: %d\n", perfCfg.MaxCacheEntries)
	fmt.Printf("  CacheTTL: %v\n", perfCfg.CacheTTL)
	fmt.Printf("  WorkerPoolSize: %d\n", perfCfg.WorkerPoolSize)
	fmt.Printf("  CacheCleanup: %v\n\n", perfCfg.CacheCleanup)

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
	fmt.Println("  - Use html.Extract() or html.ExtractText() - no processor needed")
	fmt.Println()
	fmt.Println("Use Case: Multiple documents (repeated)")
	fmt.Println("  - Create one Processor, reuse it")
	fmt.Println("  - Cache provides significant speedup for repeated content")
	fmt.Println()
	fmt.Println("Use Case: Batch processing (10+ docs)")
	fmt.Println("  - Use ExtractBatch() for parallel processing")
	fmt.Println("  - Set WorkerPoolSize to match CPU cores")
	fmt.Println()
	fmt.Println("Use Case: Web server")
	fmt.Println("  - Share one Processor across goroutines")
	fmt.Println("  - Thread-safe by design")
}
