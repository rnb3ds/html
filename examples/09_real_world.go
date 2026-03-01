//go:build examples

package main

import (
	"fmt"
	"strings"

	"github.com/cybergodev/html"
	"github.com/cybergodev/html/examples/truncate"
)

// This example demonstrates real-world use cases.
// Perfect for understanding practical patterns.
func main() {
	fmt.Println("=== Real-World Use Cases ===\n")

	processor, _ := html.New()
	defer processor.Close()

	// ============================================================
	// Use Case 1: Blog Article Extraction
	// ============================================================
	fmt.Println("Use Case 1: Blog Article Extraction")
	fmt.Println("----------------------------")

	blogHTML := `
		<html>
			<head><title>Understanding Go Generics</title></head>
			<body>
				<nav>Menu | Home | About</nav>
				<main>
					<article>
						<h1>Understanding Go Generics</h1>
						<p>Generics allow writing flexible code.</p>
					</article>
				</main>
				<footer>© 2024</footer>
			</body>
		</html>
	`

	result, _ := processor.Extract([]byte(blogHTML))
	fmt.Printf("Title: %s\n", result.Title)
	fmt.Printf("Content: %s\n", truncate.Truncate(result.Text, 80))
	fmt.Printf("Images: %d (noise removed)\n\n", len(result.Images))
	for _, img := range result.Images {
		fmt.Printf("  Image: %s (alt: %q)\n", img.URL, img.Alt)
		if img.IsDecorative {
			fmt.Printf("  Decorative: %v\n", img.IsDecorative)
		}
	}

	// ============================================================
	// Use Case 2: Newsletter Processing
	// ============================================================
	fmt.Println("\nUse Case 2: Newsletter Content")
	fmt.Println("------------------------------")

	newsletterHTML := `
		<html>
			<body>
				<p style="display:none">Unsubscribe | View in browser</p>
				<table width="600">
					<tr><td><h1>Weekly Tech Digest</h1></td></tr>
					<tr><td><p>This week in tech...</p></td></tr>
					<tr><td><p>Subscribe to tutorials</p></td></tr>
				</table>
			</body>
		</html>
	`

	result, _ = processor.Extract([]byte(newsletterHTML))
	fmt.Printf("Title: %s\n", result.Title)
	fmt.Printf("Content: %s\n", truncate.Truncate(result.Text, 80))
	fmt.Println("  - Ignored hidden content")
	fmt.Println("  - Extracted main content from tables")
	fmt.Println()

	// ============================================================
	// Use Case 3: RSS Feed Processing
	// ============================================================
	fmt.Println("\nUse Case 3: RSS Feed Items")
	fmt.Println("------------------------------")

	rssItems := []string{
		`<item><title>Go 1.22 Released</title><description><p>New features.</p></description></item>`,
		`<item><title>Go 1.21 Released</title><description><p>Bug fixes.</p></description></item>`,
		`<item><title>Go 1.20 Released</title><description><p>Performance improvements.</p></description></item>`,
	}

	for i, item := range rssItems {
		content := extractDescription(item)
		result, err := processor.Extract([]byte(content))
		if err != nil {
			continue
		}
		fmt.Printf("  [%d] %s (%d words)\n", i+1, result.Title, result.WordCount)
	}
	fmt.Println()

	// ============================================================
	// Use Case 4: Documentation Content
	// ============================================================
	fmt.Println("\nUse Case 4: Documentation Extraction")
	fmt.Println("------------------------------")

	docsHTML := `
		<html>
			<head><title>API Reference</title></head>
			<body>
				<aside>Sidebar nav</aside>
				<main>
					<h1>API Reference</h1>
					<p>This document describes the API endpoints.</p>
					<p>Refer to the examples for practical implementations.</p>
					<pre>func New() *Processor</pre>
				</main>
			</body>
		</html>
	`

	result, _ = processor.Extract([]byte(docsHTML))
	fmt.Printf("Title: %s\n", result.Title)
	fmt.Printf("Content length: %d chars\n", len(result.Text))
	fmt.Println("  - Removed sidebar navigation")
	fmt.Println("  - Extracted documentation and code")
	fmt.Println()

	// ============================================================
	// Use Case 5: File Extraction Pattern
	// ============================================================
	fmt.Println("\nUse Case 5: File Extraction Pattern")
	fmt.Println("--------------------------------")

	fmt.Println("Single file:")
	fmt.Println("  result, err := processor.ExtractFromFile(\"article.html\")")

	fmt.Println("\nBatch files:")
	fmt.Println("  results, err := processor.ExtractBatchFiles([]string{\"a.html\", \"b.html\"})")
	fmt.Println()
	fmt.Println("  for i, r := range results {")
	fmt.Println("    if r != nil {")
	fmt.Println("      fmt.Printf(\"%s (%d words)\\n\", r.Title, r.WordCount)")
	fmt.Println("    }")
	fmt.Println("  }")

	fmt.Println("\n=== Summary ===")
	fmt.Println("1. Blog extraction: Clean content, noise removal")
	fmt.Println("2. Newsletter: Table content, hidden elements")
	fmt.Println("3. RSS feeds: Batch processing of descriptions")
	fmt.Println("4. Documentation: Code blocks, sidebars")
	fmt.Println("5. File operations: Single and batch extraction")
}

// extractDescription extracts content from RSS item description tag
func extractDescription(item string) string {
	start := strings.Index(item, "<description>")
	end := strings.Index(item, "</description>")
	if start == -1 || end == -1 {
		return ""
	}
	return item[start+len("<description>") : end]
}
