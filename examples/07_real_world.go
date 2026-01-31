//go:build examples

package main

import (
	"fmt"
	"strings"

	"github.com/cybergodev/html"
)

func fetchBlogHTML() string {
	return `
		<html>
		<head><title>Understanding Go Generics</title></head>
		<body>
			<div class="header">Logo • Menu</div>
			<div class="sidebar">
				<div class="ad">Advertisement</div>
				<div class="related">Related Posts</div>
			</div>
			<main>
				<article>
					<h1>Understanding Go Generics</h1>
					<p class="meta">By John Doe • January 15, 2024</p>
					<p>Generics in Go allow you to write code that works with multiple types.</p>
					<img src="generics-diagram.png" alt="Generics Type Parameters">
					<h2>Type Parameters</h2>
					<p>Type parameters enable you to write flexible functions...</p>
				</article>
			</main>
			<footer>Copyright 2024</footer>
		</body>
		</html>
	`
}

// RealWorld demonstrates practical use cases.
func main() {
	fmt.Println("=== Real-World Use Cases ===\n ")

	processor := html.NewWithDefaults()
	defer processor.Close()

	// Use Case 1: Web scraping - Extract article from blog
	fmt.Println("1. Web scraping - Extract blog article:")
	blogHTML := fetchBlogHTML()
	result, _ := processor.ExtractWithDefaults(blogHTML)
	fmt.Printf("   Title: %s\n", result.Title)
	fmt.Printf("   Content length: %d characters\n", len(result.Text))
	fmt.Printf("   Images found: %d\n\n", len(result.Images))

	// Use Case 2: RSS feed item processing
	fmt.Println("2. RSS feed item processing:")
	rssItems := []string{
		`<item><title>Go 1.22 Released</title><description><p>Go 1.22 adds new features.</p></description></item>`,
		`<item><title>Go 1.21 Released</title><description><p>Go 1.21 improved performance.</p></description></item>`,
	}

	for _, item := range rssItems {
		// Extract from description content
		content := extractDescription(item)
		result, err := processor.ExtractWithDefaults(content)
		if err != nil {
			continue
		}
		fmt.Printf("   - %s (%d words)\n", result.Title, result.WordCount)
	}
	fmt.Println()

	// Use Case 3: Newsletter content extraction
	fmt.Println("3. Newsletter content extraction:")
	newsletterHTML := `
		<html>
		<body>
			<div style="display:none;">Unsubscribe | View in browser</div>
			<table width="600">
				<tr><td><h1>Weekly Tech Digest</h1></td></tr>
				<tr><td><p>This week in tech news...</p></td></tr>
			</table>
			<div class="footer">© 2024 Tech Digest</div>
		</body>
		</html>
	`

	result, _ = processor.ExtractWithDefaults(newsletterHTML)
	fmt.Printf("   Extracted: %s\n", result.Title)
	fmt.Printf("   Clean text: %d chars\n\n", len(strings.TrimSpace(result.Text)))

	// Use Case 4: SEO analysis
	fmt.Println("4. SEO analysis:")
	pageHTML := `
		<html>
		<head>
			<title>Product Page - Best Widgets</title>
			<meta name="description" content="Buy the best widgets at great prices">
		</head>
		<body>
			<h1>Premium Widget X1000</h1>
			<p>Our most popular widget with advanced features.</p>
			<img src="product.jpg" alt="Premium Widget X1000 Product Photo">
			<a href="/related">Related Products</a>
			<a href="https://partner.com">Partner Site</a>
		</body>
		</html>
	`

	result, _ = processor.ExtractWithDefaults(pageHTML)
	fmt.Printf("   Title length: %d chars (recommended: 50-60)\n", len(result.Title))
	fmt.Printf("   Word count: %d (SEO friendly: 300+)\n", result.WordCount)
	fmt.Printf("   Images: %d (all have alt? %v)\n",
		len(result.Images),
		allImagesHaveAlt(result.Images))
	fmt.Printf("   Internal links: %d, External: %d\n\n",
		countInternalLinks(result.Links),
		countExternalLinks(result.Links))

	// Use Case 5: Content summarization
	fmt.Println("5. Content summarization:")
	longArticle := strings.Repeat(`<p>This is detailed technical content about Go programming language features, best practices, and performance optimization techniques.</p>`, 10)
	longArticle = `<article><h1>Complete Go Guide</h1>` + longArticle + `</article>`

	result, _ = processor.ExtractWithDefaults(longArticle)
	fmt.Printf("   Full article: %d words\n", result.WordCount)
	fmt.Printf("   Reading time: %v\n", result.ReadingTime)
	fmt.Printf("   First 200 chars: %s\n\n", truncate7(result.Text, 200))

	// Use Case 6: Content aggregation
	fmt.Println("6. Content aggregation from multiple sources:")
	sources := map[string]string{
		"Blog A": `<article><h1>Post A1</h1><p>Content A1.</p></article>`,
		"Blog B": `<article><h1>Post B1</h1><p>Content B1.</p></article>`,
		"News C": `<article><h1>News C1</h1><p>Content C1.</p></article>`,
	}

	var aggregated []html.Result
	for source, content := range sources {
		result, err := processor.ExtractWithDefaults(content)
		if err != nil {
			continue
		}
		aggregated = append(aggregated, *result)
		fmt.Printf("   From %s: %s (%d words)\n", source, result.Title, result.WordCount)
	}
	fmt.Printf("   Aggregated %d articles\n\n", len(aggregated))

	// Use Case 7: Content comparison
	fmt.Println("7. Content comparison/deduplication:")
	articles := []string{
		`<article><h1>Original</h1><p>This is original content.</p></article>`,
		`<article><h1>Copy</h1><p>This is original content.</p></article>`,
		`<article><h1>Different</h1><p>This is different content.</p></article>`,
	}

	for i, art := range articles {
		result, _ := processor.ExtractWithDefaults(art)
		fmt.Printf("   Article %d: \"%s\" (hash: %x)\n", i+1,
			result.Title, checksum(result.Text))
	}

}

func extractDescription(item string) string {
	// Simple extraction - in real code use proper XML/HTML parsing
	start := strings.Index(item, "<description>")
	if start == -1 {
		return ""
	}
	start += len("<description>")
	end := strings.Index(item[start:], "</description>")
	if end == -1 {
		return ""
	}
	return item[start : start+end]
}

func allImagesHaveAlt(images []html.ImageInfo) bool {
	for _, img := range images {
		if img.Alt == "" {
			return false
		}
	}
	return true
}

func countInternalLinks(links []html.LinkInfo) int {
	count := 0
	for _, link := range links {
		if !link.IsExternal {
			count++
		}
	}
	return count
}

func countExternalLinks(links []html.LinkInfo) int {
	count := 0
	for _, link := range links {
		if link.IsExternal {
			count++
		}
	}
	return count
}

func checksum(s string) string {
	// Simple hash for demo - use crypto checksum in production
	return fmt.Sprintf("%x", len(s)*17)
}

func truncate7(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
