//go:build examples

package main

import (
	"fmt"
	"strings"

	"github.com/cybergodev/html"
	"github.com/cybergodev/html/examples/truncate"
)

// This example shows real-world use cases.
// Learn practical patterns for common scenarios.
func main() {
	fmt.Println("=== Real-World Use Cases ===\n ")

	processor, _ := html.New()
	defer processor.Close()

	// ============================================================
	// Use Case 1: Blog article extraction
	// ============================================================
	fmt.Println("Use Case 1: Blog Scraping")
	fmt.Println("-------------------------")

	blogHTML := `
		<html>
			<head><title>Understanding Go Generics</title></head>
			<body>
				<div class="header">Logo • Menu</div>
				<div class="sidebar">
					<div class="ad">Sponsored</div>
					<div class="related">Related Posts</div>
				</div>
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
	fmt.Printf("Images: %d\n\n", len(result.Images))

	fmt.Println("✓ Removed noise: header, sidebar, footer")
	fmt.Println("✓ Extracted clean article content")

	// ============================================================
	// Use Case 2: Newsletter processing
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
				</table>
			</body>
		</html>
	`

	result2, _ := processor.Extract([]byte(newsletterHTML))
	fmt.Printf("Title: %s\n", result2.Title)
	fmt.Printf("Content: %s\n\n", truncate.Truncate(result2.Text, 80))

	fmt.Println("✓ Ignored hidden content (unsubscribe links)")
	fmt.Println("✓ Extracted main content from tables")

	// ============================================================
	// Use Case 3: RSS feed item processing
	// ============================================================
	fmt.Println("\nUse Case 3: RSS Feed Items")
	fmt.Println("---------------------------")

	rssItems := []string{
		`<item><title>Go 1.22 Released</title><description><p>New features.</p></description></item>`,
		`<item><title>Go 1.21 Released</title><description><p>Bug fixes.</p></description></item>`,
	}

	fmt.Println("Processing RSS feed items:")
	for i, item := range rssItems {
		content := extractDescription(item)
		result, err := processor.Extract([]byte(content))
		if err != nil {
			continue
		}
		fmt.Printf("  [%d] %s (%d words)\n", i+1, result.Title, result.WordCount)
	}

	// ============================================================
	// Use Case 4: Documentation content
	// ============================================================
	fmt.Println("\n\nUse Case 4: Documentation Content")
	fmt.Println("---------------------------------")

	docsHTML := `
		<html>
			<head><title>API Documentation</title></head>
			<body>
				<aside>Sidebar nav</aside>
				<main>
					<h1>API Reference</h1>
					<pre>func New() *Processor</pre>
					<p>Creates a new processor.</p>
					<h2>Examples</h2>
					<pre>processor := html.New()</pre>
				</main>
			</body>
		</html>
	`

	result3, _ := processor.Extract([]byte(docsHTML))
	fmt.Printf("Title: %s\n", result3.Title)
	fmt.Printf("Content length: %d characters\n\n", len(result3.Text))

	fmt.Println("✓ Removed sidebar navigation")
	fmt.Println("✓ Extracted documentation and code")

	// ============================================================
	// Use Case 5: Batch processing multiple pages
	// ============================================================
	fmt.Println("\nUse Case 5: Batch Processing")
	fmt.Println("----------------------------")

	pages := [][]byte{
		[]byte(`<html><body><article><h1>Page 1</h1><p>Content 1</p></article></body></html>`),
		[]byte(`<html><body><article><h1>Page 2</h1><p>Content 2</p></article></body></html>`),
		[]byte(`<html><body><article><h1>Page 3</h1><p>Content 3</p></article></body></html>`),
	}

	results, err := processor.ExtractBatch(pages)
	if err != nil {
		fmt.Printf("Batch processing error: %v\n", err)
	}

	fmt.Printf("Processed %d pages:\n", len(results))
	for i, r := range results {
		if r != nil {
			fmt.Printf("  [%d] %s (%d words)\n", i+1, r.Title, r.WordCount)
		}
	}

	// ============================================================
	// Summary
	// ============================================================
	fmt.Println("\n=== Common Patterns ===")
	fmt.Println("----------------------")
	fmt.Println()
	fmt.Println("1. Web scraping:")
	fmt.Println("   result, _ := html.Extract(fetchURL(url))")
	fmt.Println()
	fmt.Println("2. Email processing:")
	fmt.Println("   Extract content from HTML emails")
	fmt.Println()
	fmt.Println("3. Feed processing:")
	fmt.Println("   Loop through items and extract each")
	fmt.Println()
	fmt.Println("4. Documentation:")
	fmt.Println("   Extract clean text from docs")
	fmt.Println()
	fmt.Println("5. Batch processing:")
	fmt.Println("   results, _ := processor.ExtractBatch(pages)")
	fmt.Println("   results, _ := processor.ExtractBatchFiles(filePaths)")
	fmt.Println()
	fmt.Println("6. File extraction:")
	fmt.Println("   result, _ := processor.ExtractFromFile(\"article.html\")")
}

func extractDescription(item string) string {
	start := strings.Index(item, "<description>")
	end := strings.Index(item, "</description>")
	if start == -1 || end == -1 {
		return ""
	}
	return item[start+len("<description>") : end]
}
