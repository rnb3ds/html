//go:build examples

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cybergodev/html"
)

// CustomConfiguration demonstrates how to customize processor and extraction
// settings for specific use cases.
func main() {
	// Custom processor configuration
	processorConfig := html.Config{
		MaxInputSize:       10 * 1024 * 1024, // 10MB max input
		ProcessingTimeout:  15 * time.Second, // 15s timeout
		MaxCacheEntries:    500,              // Cache 500 results
		CacheTTL:           30 * time.Minute, // 30 min cache TTL
		WorkerPoolSize:     8,                // 8 parallel workers
		EnableSanitization: true,             // Sanitize HTML
		MaxDepth:           50,               // Max nesting depth
	}

	processor, err := html.New(processorConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer processor.Close()

	htmlContent := `
		<html>
		<body>
			<article>
				<h1>Custom Configuration Example</h1>
				<p>This example demonstrates custom processor and extraction settings.</p>
				<img src="https://example.com/diagram.png" alt="Diagram">
				<p>Additional content with <a href="https://example.com">external link</a>.</p>
				<video src="video.mp4"></video>
			</article>
		</body>
		</html>
	`

	fmt.Println("=== Custom Configuration Example ===\n ")

	// Example 1: Article extraction with all media
	fmt.Println("Configuration 1: Article extraction with all media")
	extractConfig1 := html.ExtractConfig{
		ExtractArticle:    true,
		PreserveImages:    true,
		PreserveLinks:     true,
		PreserveVideos:    true,
		PreserveAudios:    true,
		InlineImageFormat: "markdown",
	}

	result1, err := processor.Extract(htmlContent, extractConfig1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Title: %s\n", result1.Title)
	fmt.Printf("  Images: %d, Links: %d, Videos: %d\n",
		len(result1.Images), len(result1.Links), len(result1.Videos))
	fmt.Println()

	// Example 2: Minimal extraction (text only)
	fmt.Println("Configuration 2: Minimal extraction (text only)")
	extractConfig2 := html.ExtractConfig{
		ExtractArticle:    false,
		PreserveImages:    false,
		PreserveLinks:     false,
		PreserveVideos:    false,
		PreserveAudios:    false,
		InlineImageFormat: "none",
	}

	result2, err := processor.Extract(htmlContent, extractConfig2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Title: %s\n", result2.Title)
	fmt.Printf("  Text: %s\n", result2.Text)
	fmt.Printf("  Images: %d, Links: %d, Videos: %d\n",
		len(result2.Images), len(result2.Links), len(result2.Videos))
	fmt.Println()

	// Example 3: Images only with HTML format
	fmt.Println("Configuration 3: Images only with HTML format")
	extractConfig3 := html.ExtractConfig{
		ExtractArticle:    true,
		PreserveImages:    true,
		PreserveLinks:     false,
		PreserveVideos:    false,
		PreserveAudios:    false,
		InlineImageFormat: "html",
	}

	result3, err := processor.Extract(htmlContent, extractConfig3)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Title: %s\n", result3.Title)
	fmt.Printf("  Text with inline HTML images:\n%s\n", result3.Text)
	fmt.Println()

	// Display processor configuration
	fmt.Println("=== Processor Configuration ===")
	fmt.Printf("Max Input Size: %d bytes\n", processorConfig.MaxInputSize)
	fmt.Printf("Processing Timeout: %v\n", processorConfig.ProcessingTimeout)
	fmt.Printf("Max Cache Entries: %d\n", processorConfig.MaxCacheEntries)
	fmt.Printf("Cache TTL: %v\n", processorConfig.CacheTTL)
	fmt.Printf("Worker Pool Size: %d\n", processorConfig.WorkerPoolSize)
	fmt.Printf("Sanitization Enabled: %v\n", processorConfig.EnableSanitization)
	fmt.Printf("Max Depth: %d\n", processorConfig.MaxDepth)
}
