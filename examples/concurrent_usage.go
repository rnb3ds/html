//go:build examples

package main

import (
	"fmt"
	"sync"

	"github.com/cybergodev/html"
)

// ConcurrentUsage demonstrates that Processor is safe for concurrent use
// by multiple goroutines without external synchronization.
func main() {
	processor := html.NewWithDefaults()
	defer processor.Close()

	htmlTemplates := []string{
		`<html><body><article><h1>Article %d</h1><p>Content for article number %d with detailed information.</p></article></body></html>`,
		`<html><body><article><h1>Post %d</h1><p>Blog post number %d discussing various topics.</p></article></body></html>`,
		`<html><body><article><h1>Document %d</h1><p>Technical document %d with specifications.</p></article></body></html>`,
	}

	fmt.Println("=== Concurrent Usage Example ===\n ")
	fmt.Println("Processing 50 documents concurrently from 50 goroutines...\n ")

	var wg sync.WaitGroup
	results := make(chan string, 50)

	// Launch 50 goroutines
	for i := 1; i <= 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Use different templates
			template := htmlTemplates[id%len(htmlTemplates)]
			htmlContent := fmt.Sprintf(template, id, id)

			// Extract content (thread-safe)
			result, err := processor.ExtractWithDefaults(htmlContent)
			if err != nil {
				results <- fmt.Sprintf("Goroutine %d: ERROR - %v", id, err)
				return
			}

			results <- fmt.Sprintf("Goroutine %d: %s (%d words)", id, result.Title, result.WordCount)
		}(i)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and display results
	count := 0
	for msg := range results {
		count++
		if count <= 10 { // Show first 10 results
			fmt.Println(msg)
		}
	}

	if count > 10 {
		fmt.Printf("... and %d more results\n", count-10)
	}

	// Display statistics
	stats := processor.GetStatistics()
	fmt.Println("\n=== Processing Statistics ===")
	fmt.Printf("Total Processed: %d\n", stats.TotalProcessed)
	fmt.Printf("Cache Hits: %d\n", stats.CacheHits)
	fmt.Printf("Cache Misses: %d\n", stats.CacheMisses)
	if stats.TotalProcessed > 0 {
		hitRate := float64(stats.CacheHits) / float64(stats.TotalProcessed) * 100
		fmt.Printf("Cache Hit Rate: %.1f%%\n", hitRate)
	}
	fmt.Printf("Average Processing Time: %v\n", stats.AverageProcessTime)
	fmt.Printf("Errors: %d\n", stats.ErrorCount)

	fmt.Println("\nAll goroutines completed successfully!")
	fmt.Println("Processor is thread-safe for concurrent use")
}
