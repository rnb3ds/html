//go:build examples

package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cybergodev/html"
)

// This example covers advanced features for power users.
// Learn about statistics, batch processing, and custom configuration.
func main() {
	fmt.Println("=== Advanced Features ===\n ")

	// ============================================================
	// Example 1: Custom scorer for content extraction
	// ============================================================
	fmt.Println("Example 1: Custom Scorer")
	fmt.Println("------------------------")

	// Create a processor with a custom scorer
	scorer := &ArticleScorer{
		minParagraphLength: 50,
		preferredTags:      []string{"article", "main", "div"},
	}

	scorerProcessor, err := html.New(scorer)
	if err != nil {
		log.Fatal(err)
	}
	defer scorerProcessor.Close()

	// Test with sample HTML
	sampleHTML := `
		<html>
			<body>
				<nav>Navigation links here</nav>
				<article>
					<h1>Article Title</h1>
					<p>This is a substantial paragraph with enough content to be considered valuable.</p>
				</article>
				<aside>Sidebar content</aside>
			</body>
		</html>
	`

	result, err := scorerProcessor.Extract([]byte(sampleHTML))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Extracted title: %s\n", result.Title)
	fmt.Printf("Content length: %d chars\n\n", len(result.Text))

	// ============================================================
	// Example 2: Functional options configuration
	// ============================================================
	fmt.Println("\n\nExample 2: Functional Options Configuration")
	fmt.Println("--------------------------------------------")

	processor, err := html.New(
		html.WithMaxInputSize(10*1024*1024), // 10 MB
		html.WithMaxCacheEntries(5000),
		html.WithCacheTTL(2*time.Hour),
		html.WithWorkerPoolSize(8),
		html.WithProcessingTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer processor.Close()

	fmt.Println("Custom settings applied:")
	fmt.Println("  MaxInputSize: 10 MB")
	fmt.Println("  MaxCacheEntries: 5000")
	fmt.Println("  CacheTTL: 2 hours")
	fmt.Println("  WorkerPoolSize: 8")
	fmt.Println("  ProcessingTimeout: 10 seconds")

	// ============================================================
	// Example 3: Processor statistics
	// ============================================================
	fmt.Println("\n\nExample 3: Processor Statistics")
	fmt.Println("--------------------------------")

	// Process some content to generate stats
	statsHTML := `<html><body><article><h1>Test</h1><p>Content for statistics.</p></article></body></html>`
	processor.Extract([]byte(statsHTML))
	processor.Extract([]byte(statsHTML)) // Second call will hit cache

	stats := processor.GetStatistics()

	fmt.Printf("Total Processed: %d\n", stats.TotalProcessed)
	fmt.Printf("Cache Hits:      %d\n", stats.CacheHits)
	fmt.Printf("Cache Misses:    %d\n", stats.CacheMisses)
	fmt.Printf("Error Count:     %d\n", stats.ErrorCount)
	fmt.Printf("Average Time:    %v\n", stats.AverageProcessTime)

	if stats.TotalProcessed > 0 {
		hitRate := float64(stats.CacheHits) / float64(stats.TotalProcessed) * 100
		fmt.Printf("Hit Rate:        %.1f%%\n", hitRate)
	}

	// ============================================================
	// Example 4: Batch processing with context
	// ============================================================
	fmt.Println("\n\nExample 4: Batch Processing with Context")
	fmt.Println("-----------------------------------------")

	docs := make([][]byte, 5)
	for i := 0; i < 5; i++ {
		docs[i] = []byte(fmt.Sprintf(`<article><h1>Doc %d</h1><p>Content %d</p></article>`, i+1, i+1))
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	batchResult := processor.ExtractBatchWithContext(ctx, docs)

	fmt.Printf("Batch Results:\n")
	fmt.Printf("  Success:   %d\n", batchResult.Success)
	fmt.Printf("  Failed:    %d\n", batchResult.Failed)
	fmt.Printf("  Cancelled: %d\n", batchResult.Cancelled)

	for i, r := range batchResult.Results {
		if r != nil {
			fmt.Printf("  [%d] %s\n", i+1, r.Title)
		}
	}

	// ============================================================
	// Example 5: Clear cache and reset statistics
	// ============================================================
	fmt.Println("\n\nExample 5: Cache and Statistics Management")
	fmt.Println("-------------------------------------------")

	fmt.Println("Before reset:")
	fmt.Printf("  Total Processed: %d\n", stats.TotalProcessed)

	processor.ClearCache()
	processor.ResetStatistics()

	stats = processor.GetStatistics()
	fmt.Println("After reset:")
	fmt.Printf("  Total Processed: %d\n", stats.TotalProcessed)

	// ============================================================
	// Example 6: File extraction (demonstration)
	// ============================================================
	fmt.Println("\n\nExample 6: File Extraction")
	fmt.Println("--------------------------")

	fmt.Println("Usage:")
	fmt.Println("  result, err := processor.ExtractFromFile(\"article.html\")")
	fmt.Println()
	fmt.Println("Features:")
	fmt.Println("  • Auto-detects encoding (UTF-8, Windows-1252, GBK, Shift_JIS, etc.)")
	fmt.Println("  • Returns full Result struct with metadata")
	fmt.Println("  • Handles malformed HTML gracefully")
	fmt.Println()
	fmt.Println("Batch file processing:")
	fmt.Println("  results, err := processor.ExtractBatchFiles([]string{\"a.html\", \"b.html\"})")

	// ============================================================
	// Summary
	// ============================================================
	fmt.Println("\n\n=== Key Takeaways ===")
	fmt.Println("1. Use functional options for clean configuration")
	fmt.Println("2. Monitor statistics for performance tuning")
	fmt.Println("3. Use batch processing with context for control")
	fmt.Println("4. Implement custom scorer for specialized extraction")
	fmt.Println("5. Manage cache and statistics as needed")
}

// ArticleScorer is a custom scorer that prioritizes article content.
// This demonstrates how to implement the html.Scorer interface.
type ArticleScorer struct {
	minParagraphLength int
	preferredTags      []string
}

// Score calculates a relevance score for a content node.
// Higher scores indicate more likely main content.
func (s *ArticleScorer) Score(node *html.Node) int {
	if node.Type != html.ElementNode {
		return 0
	}

	score := 0

	// Check tag name
	switch node.Data {
	case "article":
		score += 100
	case "main":
		score += 90
	case "section":
		score += 50
	case "div":
		score += 10
	case "p":
		// Score paragraphs based on length
		textLen := len(GetTextContent(node))
		if textLen >= s.minParagraphLength {
			score += 30
		}
	}

	// Check class attributes for content indicators
	for _, attr := range node.Attr {
		if attr.Key == "class" {
			if containsContent(attr.Val, "content", "article", "post", "entry") {
				score += 20
			}
			if containsContent(attr.Val, "sidebar", "nav", "footer", "ad", "promo") {
				score -= 50
			}
		}
	}

	return score
}

// ShouldRemove determines if a node should be removed from the content tree.
func (s *ArticleScorer) ShouldRemove(node *html.Node) bool {
	if node.Type != html.ElementNode {
		return false
	}

	// Remove navigation and sidebar elements
	switch node.Data {
	case "nav", "aside", "footer", "header":
		return true
	}

	// Check for ad-related classes
	for _, attr := range node.Attr {
		if attr.Key == "class" {
			if containsContent(attr.Val, "ad-", "sponsor", "promo", "sidebar") {
				return true
			}
		}
	}

	return false
}

// containsContent checks if s contains any of the substrings.
func containsContent(s string, substrs ...string) bool {
	sLower := strings.ToLower(s)
	for _, substr := range substrs {
		if strings.Contains(sLower, substr) {
			return true
		}
	}
	return false
}

// GetTextContent extracts text content from a node (helper for scorer).
func GetTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var result string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result += GetTextContent(c)
	}
	return result
}
