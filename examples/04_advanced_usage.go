//go:build examples

package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cybergodev/html"
)

// AdvancedUsage demonstrates performance optimization features.
func main() {
	fmt.Println("=== Advanced Usage & Performance ===\n ")

	// Example 1: Custom processor configuration
	fmt.Println("1. Custom processor configuration:")
	config := html.Config{
		MaxInputSize:       10 * 1024 * 1024, // 10MB
		ProcessingTimeout:  30 * time.Second,
		MaxCacheEntries:    1000,
		CacheTTL:           time.Hour,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
	}

	processor, err := html.New(config)
	if err != nil {
		log.Fatal(err)
	}
	defer processor.Close()

	fmt.Printf("   Configured: %dMB input, %d workers, %d cache entries\n\n",
		config.MaxInputSize/(1024*1024), config.WorkerPoolSize, config.MaxCacheEntries)

	// Example 2: Caching performance
	fmt.Println("2. Caching performance:")
	testHTML := `<article><h1>Test Article</h1><p>Content for caching test.</p></article>`

	// First call - cache miss
	start := time.Now()
	processor.ExtractWithDefaults(testHTML)
	missTime := time.Since(start)

	// Second call - cache hit
	start = time.Now()
	processor.ExtractWithDefaults(testHTML)
	hitTime := time.Since(start)

	fmt.Printf("   Cache miss: %v\n", missTime)
	fmt.Printf("   Cache hit: %v (%.1fx faster)\n\n", hitTime, float64(missTime)/float64(hitTime))

	// Example 3: Batch processing
	fmt.Println("3. Batch processing:")
	docs := make([]string, 100)
	for i := range docs {
		docs[i] = fmt.Sprintf(`<article><h1>Article %d</h1><p>Content %d.</p></article>`, i, i)
	}

	start = time.Now()
	results, err := processor.ExtractBatch(docs, html.DefaultExtractConfig())
	if err != nil {
		log.Fatal(err)
	}
	batchTime := time.Since(start)

	fmt.Printf("   Processed %d documents in %v\n", len(results), batchTime)
	fmt.Printf("   Average: %v per document\n\n", batchTime/time.Duration(len(results)))

	// Example 4: Concurrent processing
	fmt.Println("4. Concurrent processing (thread-safe):")
	var wg sync.WaitGroup
	concurrentStart := time.Now()

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			html := fmt.Sprintf(`<article><h1>Article %d</h1><p>Concurrent test.</p></article>`, id)
			processor.ExtractWithDefaults(html)
		}(i)
	}

	wg.Wait()
	concurrentTime := time.Since(concurrentStart)
	fmt.Printf("   Processed 50 documents concurrently in %v\n\n", concurrentTime)

	// Example 5: Statistics monitoring
	fmt.Println("5. Processor statistics:")
	stats := processor.GetStatistics()
	fmt.Printf("   Total Processed: %d\n", stats.TotalProcessed)
	fmt.Printf("   Cache Hits: %d (%.1f%%)\n", stats.CacheHits,
		float64(stats.CacheHits)/float64(stats.TotalProcessed)*100)
	fmt.Printf("   Cache Misses: %d\n", stats.CacheMisses)
	fmt.Printf("   Total Errors: %d\n", stats.ErrorCount)
	if stats.TotalProcessed > 0 {
		fmt.Printf("   Avg Time: %v\n", stats.AverageProcessTime)
	}

}
