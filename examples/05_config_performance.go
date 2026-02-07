//go:build examples

package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/cybergodev/html"
)

// This example demonstrates configuration and performance optimization techniques.
// Learn how to tune the library for your specific use case.
func main() {
	fmt.Println("=== Configuration & Performance ===\n ")

	sampleHTML := `<article><h1>Test</h1><p>Content with some text for extraction.</p></article>`

	// ============================================================
	// PART 1: Understanding configuration options
	// ============================================================
	fmt.Println("Part 1: Configuration Options")
	fmt.Println("---------------------------")

	// Start with defaults and understand what each setting does
	config := html.DefaultConfig()

	fmt.Println("Default Configuration:")
	fmt.Printf("  MaxInputSize:     %d (50 MB)\n", config.MaxInputSize)
	fmt.Printf("  MaxCacheEntries:  %d\n", config.MaxCacheEntries)
	fmt.Printf("  Cache TTL:        %v\n", config.CacheTTL)
	fmt.Printf("  WorkerPoolSize:  %d\n", config.WorkerPoolSize)
	fmt.Printf("  EnableSanitiz:   %v\n", config.EnableSanitization)
	fmt.Printf("  MaxDepth:         %d\n", config.MaxDepth)
	fmt.Printf("  ProcessingTimeout: %v\n", config.ProcessingTimeout)

	// When to adjust each setting:
	fmt.Println("\nWhen to adjust settings:")
	fmt.Println("  MaxInputSize:")
	fmt.Println("    • Maximum allowed: 50MB")
	fmt.Println("    • Decrease to limit memory usage")
	fmt.Println()
	fmt.Println("  MaxCacheEntries:")
	fmt.Println("    • Increase for processing many unique documents")
	fmt.Println("    • Decrease to reduce memory footprint")
	fmt.Println()
	fmt.Println("  Cache TTL:")
	fmt.Println("    • Longer = better cache hit rate")
	fmt.Println("    • Shorter = fresher content")
	fmt.Println()
	fmt.Println("  WorkerPoolSize:")
	fmt.Println("    • More workers = faster batch processing")
	fmt.Println("    • More workers = higher memory usage")
	fmt.Println()
	fmt.Println("  EnableSanitization:")
	fmt.Println("    • Always keep enabled for security")
	fmt.Println("    • Only disable for trusted HTML sources")
	fmt.Println()
	fmt.Println("  MaxDepth:")
	fmt.Println("    • Most pages work fine with default (500)")
	fmt.Println("    • Reduce for deeply nested document structures")

	// ============================================================
	// PART 2: Caching benefits
	// ============================================================
	fmt.Println("\n\nPart 2: Caching Performance")
	fmt.Println("-----------------------")

	processor, _ := html.New(html.DefaultConfig())
	defer processor.Close()

	// First extraction - cache miss
	start := time.Now()
	processor.Extract([]byte(sampleHTML))
	missTime := time.Since(start)

	// Second extraction - cache hit
	start = time.Now()
	processor.Extract([]byte(sampleHTML))
	hitTime := time.Since(start)

	fmt.Printf("Cache miss: %v\n", missTime)
	fmt.Printf("Cache hit:  %v\n", hitTime)

	if hitTime > 0 && missTime > 0 {
		speedup := float64(missTime) / float64(hitTime)
		fmt.Printf("Speedup:    %.1fx faster\n", speedup)
	}

	fmt.Println("\nKey insight:")
	fmt.Println("  • Caching provides 10-100x speedup on repeated content")
	fmt.Println("  • Cache keys are based on HTML content (SHA-256)")
	fmt.Println("  • Same HTML = same cache key, even from different sources")

	// ============================================================
	// PART 3: Batch processing performance
	// ============================================================
	fmt.Println("\n\nPart 3: Batch Processing")
	fmt.Println("---------------------")

	// Create sample documents
	docCount := 100
	docs := make([][]byte, docCount)
	for i := 0; i < docCount; i++ {
		docs[i] = []byte(fmt.Sprintf(`<article><h1>Article %d</h1><p>Content for article %d.</p></article>`, i, i))
	}

	// Compare sequential vs batch processing
	config2 := html.DefaultConfig()
	config2.WorkerPoolSize = 4

	processor2, _ := html.New(config2)
	defer processor2.Close()

	// Sequential processing
	start = time.Now()
	sequentialCount := 0
	for _, doc := range docs {
		result, err := processor2.Extract(doc, html.DefaultExtractConfig())
		if err == nil && result != nil {
			sequentialCount++
		}
	}
	sequentialTime := time.Since(start)

	// Batch processing
	start = time.Now()
	results, _ := processor2.ExtractBatch(docs, html.DefaultExtractConfig())
	batchTime := time.Since(start)

	fmt.Printf("Sequential processing: %v (%d docs)\n", sequentialTime, sequentialCount)
	fmt.Printf("Batch processing:      %v (%d docs)\n", batchTime, len(results))

	if sequentialTime > 0 && batchTime > 0 {
		speedup := float64(sequentialTime) / float64(batchTime)
		fmt.Printf("Speedup:               %.1fx faster\n", speedup)
	}

	fmt.Println("\nKey insight:")
	fmt.Println("  • Batch processing is 2-4x faster for multiple documents")
	fmt.Println("  • Uses worker pool to process documents in parallel")
	fmt.Println("  • Ideal for processing 10+ documents at once")

	// ============================================================
	// PART 4: Concurrent processing (thread-safe)
	// ============================================================
	fmt.Println("\n\nPart 4: Concurrent Processing")
	fmt.Println("---------------------------")

	// Same processor can be used from multiple goroutines
	processor3, _ := html.New()
	defer processor3.Close()

	const numGoroutines = 10
	const docsPerGoroutine = 10

	var wg sync.WaitGroup
	concurrentStart := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < docsPerGoroutine; j++ {
				doc := []byte(fmt.Sprintf(`<article><h1>Doc %d-%d</h1><p>Content</p></article>`, id, j))
				_, _ = processor3.Extract(doc, html.DefaultExtractConfig())
			}
		}(i)
	}

	wg.Wait()
	concurrentTime := time.Since(concurrentStart)

	fmt.Printf("Processed %d documents concurrently in %v\n", numGoroutines*docsPerGoroutine, concurrentTime)
	fmt.Printf("Average: %v per document\n", concurrentTime/time.Duration(numGoroutines*docsPerGoroutine))

	fmt.Println("\nKey insight:")
	fmt.Println("  • Processor is thread-safe (goroutine-safe)")
	fmt.Println("  • Same processor can be shared across your application")
	fmt.Println("  • Cache and statistics are protected by locks")

	// ============================================================
	// PART 5: Memory optimization
	// ============================================================
	fmt.Println("\n\nPart 5: Memory Optimization")
	fmt.Println("-------------------------")

	// Compare memory usage with different configurations
	configs := []struct {
		name    string
		config  html.Config
		comment string
	}{
		{
			name: "Minimal (lowest memory)",
			config: html.Config{
				MaxInputSize:    1 * 1024 * 1024, // 1MB
				MaxCacheEntries: 0,               // No cache
				WorkerPoolSize:  1,               // Single worker
				MaxDepth:        100,             // Lower limit for simple docs
			},
			comment: "Best for memory-constrained environments",
		},
		{
			name:    "Balanced (default)",
			config:  html.DefaultConfig(),
			comment: "Good balance of speed and memory",
		},
		{
			name: "Performance (higher memory)",
			config: html.Config{
				MaxInputSize:    50 * 1024 * 1024, // 50MB (max allowed)
				MaxCacheEntries: 10000,            // Large cache
				WorkerPoolSize:  16,               // Many workers
				MaxDepth:        500,              // Standard limit
			},
			comment: "Best for high-performance servers",
		},
	}

	for _, cfg := range configs {
		processor4, err := html.New(cfg.config)
		if err != nil {
			fmt.Printf("\n%s: (config error: %v)\n", cfg.name, err)
			continue
		}
		processor4.Close()

		fmt.Printf("\n%s:\n", cfg.name)
		fmt.Printf("  %s\n", cfg.comment)
		fmt.Printf("  MaxInputSize: %d MB\n", cfg.config.MaxInputSize/(1024*1024))
		fmt.Printf("  MaxCacheEntries: %d\n", cfg.config.MaxCacheEntries)
		fmt.Printf("  WorkerPoolSize: %d\n", cfg.config.WorkerPoolSize)
	}

	fmt.Println("\n\nPart 6: Performance Recommendations")
	fmt.Println("-------------------------------")

	fmt.Println("Use Case: Single Page Extraction")
	fmt.Println("  • Use html.Extract() - simplest and fastest")
	fmt.Println("  • No need to create processor")
	fmt.Println()

	fmt.Println("Use Case: Multiple Pages (same site)")
	fmt.Println("  • Create one processor, reuse it")
	fmt.Println("  • Cache provides 10-100x speedup")
	fmt.Println("  • Example: Scraping articles from a blog")
	fmt.Println()

	fmt.Println("Use Case: Batch Processing (many pages)")
	fmt.Println("  • Use ExtractBatch() for 2-4x speedup")
	fmt.Println("  • Set WorkerPoolSize to number of CPU cores")
	fmt.Println("  • Example: Processing RSS feed items")
	fmt.Println()

	fmt.Println("Use Case: Concurrent Access")
	fmt.Println("  • Share single processor across goroutines")
	fmt.Println("  • Thread-safe by design")
	fmt.Println("  • Example: Web server extracting content")

}
