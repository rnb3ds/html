//go:build examples

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cybergodev/html"
)

// This example covers advanced features for power users.
// Learn about statistics, file extraction, and custom encoding.
func main() {
	fmt.Println("=== Advanced Features ===\n ")

	// ============================================================
	// Example 1: File extraction with encoding detection
	// ============================================================
	fmt.Println("Example 1: File Extraction")
	fmt.Println("-------------------------")

	processor, _ := html.New()
	defer processor.Close()

	fmt.Println("Usage:")
	fmt.Println("  result, err := processor.ExtractFromFile(\"article.html\")")
	fmt.Println()
	fmt.Println("Features:")
	fmt.Println("  • Auto-detects encoding (UTF-8, Windows-1252, GBK, etc.)")
	fmt.Println("  • Returns full Result struct with metadata")
	fmt.Println("  • Handles malformed HTML gracefully")
	fmt.Println()

	// ============================================================
	// Example 2: Processor statistics
	// ============================================================
	fmt.Println("Example 2: Processor Statistics")
	fmt.Println("----------------------------")

	stats := processor.GetStatistics()

	fmt.Printf("Extractions:    %d\n", stats.TotalProcessed)
	fmt.Printf("Cache Hits:      %d\n", stats.CacheHits)
	fmt.Printf("Cache Misses:    %d\n", stats.CacheMisses)

	if stats.TotalProcessed > 0 {
		hitRate := float64(stats.CacheHits) / float64(stats.TotalProcessed) * 100
		fmt.Printf("Hit Rate:        %.1f%%\n", hitRate)
	}

	fmt.Println("\nWhat caching means:")
	fmt.Println("  • Same HTML = extracted only once (10-100x faster)")
	fmt.Println("  • Cache key = SHA-256 hash of HTML content")
	fmt.Println("  • Automatic - no configuration needed")

	// ============================================================
	// Example 3: Custom configuration
	// ============================================================
	fmt.Println("\n\nExample 3: Custom Configuration")
	fmt.Println("------------------------------")

	config := html.DefaultConfig()
	config.MaxInputSize = 10 * 1024 * 1024 // 10 MB
	config.MaxDepth = 100                  // Lower limit
	config.CacheTTL = time.Hour            // Longer TTL

	fmt.Println("Custom settings:")
	fmt.Printf("  MaxInputSize:  %d MB\n", config.MaxInputSize/(1024*1024))
	fmt.Printf("  MaxDepth:     %d levels\n", config.MaxDepth)
	fmt.Printf("  Cache TTL:    %v\n", config.CacheTTL)

	customProcessor, err := html.New(config)
	if err != nil {
		log.Fatal(err)
	}
	defer customProcessor.Close()

	fmt.Println("\nWhen to customize:")
	fmt.Println("  • LargeInputSize: For big documents (100MB+)")
	fmt.Println("  • Lower MaxDepth: For deeply nested HTML")
	fmt.Println("  • More workers:  For batch processing")
	fmt.Println("  • Less cache:    For memory-constrained apps")

	_ = customProcessor

	// ============================================================
	// Example 4: Processing timeout
	// ============================================================
	fmt.Println("\n\nExample 4: Timeout Handling")
	fmt.Println("-----------------------")

	timeoutConfig := html.DefaultConfig()
	timeoutConfig.ProcessingTimeout = 5 * time.Second

	fmt.Println("Timeout settings:")
	fmt.Printf("  Default: 30 seconds (good for most)\n")
	fmt.Printf("  Custom:  %v (this example)\n", timeoutConfig.ProcessingTimeout)
	fmt.Println("\nWhy set timeout?")
	fmt.Println("  • Prevents hanging on malformed HTML")
	fmt.Println("  • Faster error detection")
	fmt.Println("  • Avoids resource exhaustion")

	// ============================================================
	// Example 5: Size limits
	// ============================================================
	fmt.Println("\n\nExample 5: Input Size Limits")
	fmt.Println("-------------------------")

	fmt.Println("Size limit configuration:")
	fmt.Printf("  Default:  50 MB (handles most pages)\n")
	fmt.Printf("  Custom:   Set via MaxInputSize in Config\n")
	fmt.Println()
	fmt.Println("Why limit size?")
	fmt.Println("  • Prevents DoS attacks")
	fmt.Println("  • Predictable memory usage")
	fmt.Println("  • Prevents crashes from huge files")

	// ============================================================
	// Summary
	// ============================================================
	fmt.Println("\n\n=== Key Takeaways ===")
	fmt.Println("1. ExtractFromFile() handles files automatically")
	fmt.Println("2. Statistics track cache effectiveness")
	fmt.Println("3. Customize Config for your specific needs")
	fmt.Println("4. Timeouts prevent hanging")
	fmt.Println("5. Size limits prevent memory issues")

}
