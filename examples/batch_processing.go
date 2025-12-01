//go:build examples

package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// BatchProcessing demonstrates parallel processing of multiple HTML documents
// using worker pools for efficient extraction.
func main() {
	processor := html.NewWithDefaults()
	defer processor.Close()

	// Multiple HTML documents to process
	htmlContents := []string{
		`<html><body><article><h1>Article 1</h1><p>Content about Go programming with detailed explanations and examples.</p></article></body></html>`,
		`<html><body><article><h1>Article 2</h1><p>Content about web development using modern frameworks and best practices.</p></article></body></html>`,
		`<html><body><article><h1>Article 3</h1><p>Content about database design with normalization and optimization techniques.</p></article></body></html>`,
		`<html><body><article><h1>Article 4</h1><p>Content about cloud architecture using microservices and containerization.</p></article></body></html>`,
		`<html><body><article><h1>Article 5</h1><p>Content about security best practices including authentication and encryption.</p></article></body></html>`,
	}

	fmt.Println("=== Batch Processing Example ===\n ")
	fmt.Printf("Processing %d documents in parallel...\n\n", len(htmlContents))

	// Process all documents in parallel
	config := html.DefaultExtractConfig()
	results, err := processor.ExtractBatch(htmlContents, config)
	if err != nil {
		log.Fatal(err)
	}

	// Display results
	for i, result := range results {
		if result != nil {
			fmt.Printf("Document %d:\n", i+1)
			fmt.Printf("  Title: %s\n", result.Title)
			fmt.Printf("  Word Count: %d\n", result.WordCount)
			fmt.Printf("  Reading Time: %v\n", result.ReadingTime)
			fmt.Printf("  Processing Time: %v\n", result.ProcessingTime)
			fmt.Println()
		} else {
			fmt.Printf("Document %d: Failed to process\n\n", i+1)
		}
	}

	// Display statistics
	stats := processor.GetStatistics()
	fmt.Println("=== Processing Statistics ===")
	fmt.Printf("Total Processed: %d\n", stats.TotalProcessed)
	fmt.Printf("Cache Hits: %d\n", stats.CacheHits)
	fmt.Printf("Cache Misses: %d\n", stats.CacheMisses)
	if stats.TotalProcessed > 0 {
		hitRate := float64(stats.CacheHits) / float64(stats.TotalProcessed) * 100
		fmt.Printf("Cache Hit Rate: %.1f%%\n", hitRate)
	}
	fmt.Printf("Average Processing Time: %v\n", stats.AverageProcessTime)
	fmt.Printf("Errors: %d\n", stats.ErrorCount)
}
