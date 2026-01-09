//go:build examples

package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cybergodev/html"
)

// AdvancedUsage demonstrates advanced features including custom configuration,
// caching, batch processing, and concurrent usage.
func main() {
	fmt.Println("=== Advanced Usage Example ===\n ")

	// Example 1: Custom processor configuration
	fmt.Println("1. Custom processor configuration:")
	customConfig := html.Config{
		MaxInputSize:       10 * 1024 * 1024, // 10MB max input
		ProcessingTimeout:  15 * time.Second, // 15s timeout
		MaxCacheEntries:    500,              // Cache 500 results
		CacheTTL:           30 * time.Minute, // 30 min cache TTL
		WorkerPoolSize:     8,                // 8 parallel workers
		EnableSanitization: true,             // Sanitize HTML
		MaxDepth:           50,               // Max nesting depth
	}

	processor, err := html.New(customConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer processor.Close()

	fmt.Printf("   Max Input Size: %d bytes\n", customConfig.MaxInputSize)
	fmt.Printf("   Processing Timeout: %v\n", customConfig.ProcessingTimeout)
	fmt.Printf("   Max Cache Entries: %d\n", customConfig.MaxCacheEntries)
	fmt.Printf("   Worker Pool Size: %d\n\n", customConfig.WorkerPoolSize)

	// Example 2: Custom extraction configuration
	fmt.Println("2. Custom extraction configuration:")
	htmlContent := `
		<article>
			<h1>Custom Configuration Example</h1>
			<p>This example demonstrates custom extraction settings.</p>
			<img src="https://example.com/diagram.png" alt="Diagram">
			<p>Additional content with <a href="https://example.com">external link</a>.</p>
			<video src="video.mp4"></video>
		</article>
	`

	// Full extraction with Markdown images
	fullConfig := html.ExtractConfig{
		ExtractArticle:    true,
		PreserveImages:    true,
		PreserveLinks:     true,
		PreserveVideos:    true,
		PreserveAudios:    true,
		InlineImageFormat: "markdown",
	}

	result, err := processor.Extract(htmlContent, fullConfig)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Full extraction: %d images, %d links, %d videos\n\n",
		len(result.Images), len(result.Links), len(result.Videos))

	// Example 3: Caching performance
	fmt.Println("3. Caching performance:")
	testHTML := `<article><h1>Cache Test</h1><p>Testing caching performance with repeated content.</p></article>`

	// First extraction (cache miss)
	start := time.Now()
	_, _ = processor.ExtractWithDefaults(testHTML)
	duration1 := time.Since(start)

	// Second extraction (cache hit)
	start = time.Now()
	_, _ = processor.ExtractWithDefaults(testHTML)
	duration2 := time.Since(start)

	fmt.Printf("   First extraction (cache miss): %v\n", duration1)
	fmt.Printf("   Second extraction (cache hit): %v\n", duration2)
	if duration1 > duration2 {
		speedup := float64(duration1) / float64(duration2)
		fmt.Printf("   Cache speedup: %.2fx faster\n\n", speedup)
	}

	// Example 4: Batch processing
	fmt.Println("4. Batch processing:")
	htmlContents := []string{
		`<article><h1>Article 1</h1><p>Content about Go programming with detailed explanations.</p></article>`,
		`<article><h1>Article 2</h1><p>Content about web development using modern frameworks.</p></article>`,
		`<article><h1>Article 3</h1><p>Content about database design with optimization techniques.</p></article>`,
		`<article><h1>Article 4</h1><p>Content about cloud architecture using microservices.</p></article>`,
		`<article><h1>Article 5</h1><p>Content about security best practices and encryption.</p></article>`,
	}

	batchStart := time.Now()
	results, err := processor.ExtractBatch(htmlContents, html.DefaultExtractConfig())
	if err != nil {
		log.Fatal(err)
	}
	batchDuration := time.Since(batchStart)

	fmt.Printf("   Processed %d documents in %v\n", len(results), batchDuration)
	for i, result := range results {
		if result != nil {
			fmt.Printf("   Document %d: %s (%d words)\n", i+1, result.Title, result.WordCount)
		}
	}
	fmt.Println()

	// Example 5: Concurrent usage (thread-safe)
	fmt.Println("5. Concurrent usage (thread-safe):")
	var wg sync.WaitGroup
	concurrentResults := make(chan string, 20)

	concurrentStart := time.Now()
	for i := 1; i <= 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			html := fmt.Sprintf(`<article><h1>Article %d</h1><p>Concurrent processing test %d.</p></article>`, id, id)
			result, err := processor.ExtractWithDefaults(html)
			if err != nil {
				concurrentResults <- fmt.Sprintf("Error: %v", err)
				return
			}
			concurrentResults <- fmt.Sprintf("%s (%d words)", result.Title, result.WordCount)
		}(i)
	}

	go func() {
		wg.Wait()
		close(concurrentResults)
	}()

	count := 0
	for msg := range concurrentResults {
		count++
		if count <= 5 {
			fmt.Printf("   %s\n", msg)
		}
	}
	if count > 5 {
		fmt.Printf("   ... and %d more results\n", count-5)
	}
	concurrentDuration := time.Since(concurrentStart)
	fmt.Printf("   Processed %d documents concurrently in %v\n\n", count, concurrentDuration)

	// Example 6: Statistics monitoring
	fmt.Println("6. Statistics monitoring:")
	stats := processor.GetStatistics()
	fmt.Printf("   Total Processed: %d\n", stats.TotalProcessed)
	fmt.Printf("   Cache Hits: %d\n", stats.CacheHits)
	fmt.Printf("   Cache Misses: %d\n", stats.CacheMisses)
	if stats.TotalProcessed > 0 {
		hitRate := float64(stats.CacheHits) / float64(stats.TotalProcessed) * 100
		fmt.Printf("   Cache Hit Rate: %.1f%%\n", hitRate)
	}
	fmt.Printf("   Average Processing Time: %v\n", stats.AverageProcessTime)
	fmt.Printf("   Errors: %d\n", stats.ErrorCount)

	fmt.Println("\n✓ Advanced usage complete!")
}
