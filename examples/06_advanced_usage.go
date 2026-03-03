//go:build examples

package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cybergodev/html"
)

// This example demonstrates advanced features: custom scorers, audit logging, and security configuration.
func main() {
	fmt.Println("=== Advanced Features ===\n")

	// ============================================================
	// 1. Custom Scorer Implementation
	// ============================================================
	fmt.Println("1. Custom Scorer")
	fmt.Println("-----------------")

	scorer := &ArticleScorer{minParagraphLength: 50}
	scorerConfig := html.DefaultConfig()
	scorerConfig.Scorer = scorer
	scorerProcessor, err := html.New(scorerConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer scorerProcessor.Close()

	sampleHTML := "<html><body><nav>Navigation links</nav><article><h1>Article Title</h1><p>This is a substantial paragraph with meaningful content.</p></article><aside>Sidebar content</aside></body></html>"

	result, err := scorerProcessor.Extract([]byte(sampleHTML))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Extracted title: %s\n", result.Title)
	fmt.Printf("Content length: %d chars\n\n", len(result.Text))

	// ============================================================
	// 2. Statistics and Monitoring
	// ============================================================
	fmt.Println("2. Statistics & Monitoring")
	fmt.Println("-----------------------------")

	processor, _ := html.New()
	defer processor.Close()

	// Process documents to generate statistics
	for i := 0; i < 5; i++ {
		doc := []byte(fmt.Sprintf("<article><h1>Doc %d</h1><p>Content</p></article>", i))
		processor.Extract(doc)
		// Same document again (cache hit)
		processor.Extract(doc)
	}

	stats := processor.GetStatistics()
	fmt.Printf("Total Processed: %d\n", stats.TotalProcessed)
	fmt.Printf("Cache Hits: %d\n", stats.CacheHits)
	fmt.Printf("Cache Misses: %d\n", stats.CacheMisses)
	fmt.Printf("Avg Process Time: %v\n", stats.AverageProcessTime)

	if stats.TotalProcessed > 0 {
		hitRate := float64(stats.CacheHits) / float64(stats.TotalProcessed) * 100
		fmt.Printf("Cache Hit Rate: %.1f%%\n", hitRate)
	}

	// Clear and reset
	processor.ClearCache()
	processor.ResetStatistics()
	stats = processor.GetStatistics()
	fmt.Printf("\nAfter ClearCache/ResetStatistics: %d processed\n", stats.TotalProcessed)

	// ============================================================
	// 3. Audit System (Security Logging)
	// ============================================================
	fmt.Println("\n3. Audit System (Security Logging)")
	fmt.Println("---------------------------------")

	// Create channel sink for audit entries
	channelSink := html.NewChannelAuditSink(100)

	// Configure processor with audit
	auditConfig := html.DefaultConfig()
	auditConfig.Audit = html.HighSecurityAuditConfig()
	auditConfig.Audit.Sink = channelSink
	auditConfig.Audit.Enabled = true

	auditProcessor, _ := html.New(auditConfig)
	defer auditProcessor.Close()

	// Process potentially dangerous HTML
	dangerousHTML := "<html><body><script>alert('xss')</script><a href=\"javascript:void(0)\">Click</a><img src=\"x.png\" onerror=\"alert(1)\"></body></html>"

	auditProcessor.Extract([]byte(dangerousHTML))

	// Wait for async writes
	time.Sleep(100 * time.Millisecond)

	// Read audit entries
	auditLog := auditProcessor.GetAuditLog()
	fmt.Printf("Audit entries recorded: %d\n", len(auditLog))

	for _, entry := range auditLog {
		fmt.Printf("  [%s] %s: %s\n", entry.Level, entry.EventType, entry.Message)
	}

	// ============================================================
	// 4. Security Configuration
	// ============================================================
	fmt.Println("\n4. Security Configuration")
	fmt.Println("---------------------------")

	// High security config
	secureConfig := html.HighSecurityConfig()
	fmt.Println("High Security Config:")
	fmt.Printf("  MaxInputSize: %d MB\n", secureConfig.MaxInputSize/(1024*1024))
	fmt.Printf("  EnableSanitization: %v\n", secureConfig.EnableSanitization)
	fmt.Printf("  MaxDepth: %d\n", secureConfig.MaxDepth)

	secureProcessor, _ := html.New(secureConfig)
	defer secureProcessor.Close()

	// This will have stricter limits
	secureProcessor.Extract([]byte(dangerousHTML))

	// ============================================================
	// 5. File Processing Patterns
	// ============================================================
	fmt.Println("\n5. File Processing Patterns")
	fmt.Println("--------------------------")

	fmt.Println("Single file:")
	fmt.Println("  result, err := processor.ExtractFromFile(\"article.html\")")

	fmt.Println("\nBatch files:")
	fmt.Println("  results, err := processor.ExtractBatchFiles([]string{\"a.html\", \"b.html\"})")

	fmt.Println("\nWith context and timeout:")
	fmt.Println("  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)")
	fmt.Println("  defer cancel()")
	fmt.Println("  result := processor.ExtractBatchFilesWithContext(ctx, paths)")

	// ============================================================
	// Summary
	// ============================================================
	fmt.Println("\n=== Advanced Features Summary ===")
	fmt.Println("1. Custom Scorers: Implement html.Scorer interface for domain-specific extraction")
	fmt.Println("2. Statistics: Track processing metrics and cache hit rates")
	fmt.Println("3. Audit System: Monitor security events and blocked content")
	fmt.Println("4. Security Configs: Use HighSecurityConfig() for sensitive data processing")
	fmt.Println("5. File Operations: Single file, batch, and context-aware processing")
}

// ArticleScorer is a custom scorer that prioritizes article content.
type ArticleScorer struct {
	minParagraphLength int
}

// Score calculates a relevance score for a content node.
func (s *ArticleScorer) Score(node *html.Node) int {
	if node.Type != html.ElementNode {
		return 0
	}

	score := 0

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
		textLen := len(getTextContent(node))
		if textLen >= s.minParagraphLength {
			score += 30
		}
	}

	// Check class attributes
	for _, attr := range node.Attr {
		if attr.Key == "class" {
			classVal := strings.ToLower(attr.Val)
			if strings.Contains(classVal, "content") ||
				strings.Contains(classVal, "article") ||
				strings.Contains(classVal, "post") {
				score += 20
			}
			if strings.Contains(classVal, "sidebar") ||
				strings.Contains(classVal, "nav") ||
				strings.Contains(classVal, "footer") ||
				strings.Contains(classVal, "ad") {
				score -= 50
			}
		}
	}

	return score
}

// ShouldRemove determines if a node should be removed.
func (s *ArticleScorer) ShouldRemove(node *html.Node) bool {
	if node.Type != html.ElementNode {
		return false
	}

	switch node.Data {
	case "nav", "aside", "footer", "header":
		return true
	}

	for _, attr := range node.Attr {
		if attr.Key == "class" {
			classVal := strings.ToLower(attr.Val)
			if strings.Contains(classVal, "ad-") ||
				strings.Contains(classVal, "sponsor") ||
				strings.Contains(classVal, "promo") ||
				strings.Contains(classVal, "sidebar") {
				return true
			}
		}
	}

	return false
}

// getTextContent extracts text content from a node.
func getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var result string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result += getTextContent(c)
	}
	return result
}
