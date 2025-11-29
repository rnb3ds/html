package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cybergodev/html"
)

// CachingPerformance demonstrates the built-in caching system
// and how it improves performance for repeated content.
func main() {
	processor := html.NewWithDefaults()
	defer processor.Close()

	htmlContent := `
		<html>
		<body>
			<article>
				<h1>Performance Testing Article</h1>
				<p>This is a sample article for testing caching performance.</p>
				<p>The content is long enough to demonstrate the benefits of caching.</p>
				<img src="https://example.com/image1.jpg" alt="Image 1">
				<p>More content here with additional paragraphs and information.</p>
				<img src="https://example.com/image2.jpg" alt="Image 2">
			</article>
		</body>
		</html>
	`

	fmt.Println("=== Caching & Performance Example ===\n ")

	// First extraction (cache miss)
	fmt.Println("First extraction (cache miss)...")
	start := time.Now()
	result1, err := processor.ExtractWithDefaults(htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	duration1 := time.Since(start)
	fmt.Printf("  Title: %s\n", result1.Title)
	fmt.Printf("  Processing Time: %v\n", duration1)
	fmt.Println()

	// Second extraction (cache hit)
	fmt.Println("Second extraction (cache hit)...")
	start = time.Now()
	result2, err := processor.ExtractWithDefaults(htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	duration2 := time.Since(start)
	fmt.Printf("  Title: %s\n", result2.Title)
	fmt.Printf("  Processing Time: %v\n", duration2)
	fmt.Println()

	// Performance comparison
	if duration1 > duration2 {
		speedup := float64(duration1) / float64(duration2)
		fmt.Printf("Cache speedup: %.2fx faster\n\n", speedup)
	}

	// Display cache statistics
	stats := processor.GetStatistics()
	fmt.Println("=== Cache Statistics ===")
	fmt.Printf("Total Processed: %d\n", stats.TotalProcessed)
	fmt.Printf("Cache Hits: %d\n", stats.CacheHits)
	fmt.Printf("Cache Misses: %d\n", stats.CacheMisses)
	if stats.TotalProcessed > 0 {
		hitRate := float64(stats.CacheHits) / float64(stats.TotalProcessed) * 100
		fmt.Printf("Cache Hit Rate: %.1f%%\n", hitRate)
	}
	fmt.Printf("Average Processing Time: %v\n", stats.AverageProcessTime)
	fmt.Println()

	// Clear cache demonstration
	fmt.Println("Clearing cache...")
	processor.ClearCache()

	// Third extraction (cache miss after clear)
	fmt.Println("Third extraction (cache miss after clear)...")
	start = time.Now()
	result3, err := processor.ExtractWithDefaults(htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	duration3 := time.Since(start)
	fmt.Printf("  Title: %s\n", result3.Title)
	fmt.Printf("  Processing Time: %v\n", duration3)
}
