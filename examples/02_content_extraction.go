//go:build examples

package main

import (
	"fmt"
	"strings"

	"github.com/cybergodev/html"
)

// ContentExtraction demonstrates various content extraction options.
func main() {
	fmt.Println("=== Content Extraction Options ===\n ")

	blogHTML := `
		<html>
		<head><title>Go Interfaces Guide</title></head>
		<body>
			<nav>Navigation Menu</nav>
			<aside class="sidebar"><div class="ad">Advertisement</div></aside>
			<main>
				<article>
					<h1>Understanding Go Interfaces</h1>
					<p>Interfaces provide abstraction and polymorphism.</p>
					<img src="interface-diagram.png" alt="Interface Diagram">
					<h2>Key Concepts</h2>
					<p>An interface type is defined as a set of method signatures.</p>
					<p>For more info, visit the <a href="https://golang.org">Go website</a>.</p>
				</article>
			</main>
			<footer>Copyright 2024</footer>
		</body>
		</html>
	`

	processor := html.NewWithDefaults()
	defer processor.Close()

	// Example 1: Article extraction (removes navigation, ads, footer)
	fmt.Println("1. Smart article extraction:")
	config1 := html.DefaultExtractConfig()
	result1, _ := processor.Extract(blogHTML, config1)
	fmt.Printf("   Title: %s\n", result1.Title)
	fmt.Printf("   Text: %s\n\n", truncate2(result1.Text, 150))

	// Example 2: Inline images - Markdown format
	fmt.Println("2. Inline images (Markdown):")
	config2 := html.DefaultExtractConfig()
	config2.InlineImageFormat = "markdown"
	result2, _ := processor.Extract(blogHTML, config2)
	fmt.Printf("   %s\n\n", truncate2(result2.Text, 150))

	// Example 3: Inline images - HTML format
	fmt.Println("3. Inline images (HTML):")
	config3 := html.DefaultExtractConfig()
	config3.InlineImageFormat = "html"
	result3, _ := processor.Extract(blogHTML, config3)
	fmt.Printf("   %s\n\n", truncate2(result3.Text, 150))

	// Example 4: Inline images - Placeholder format
	fmt.Println("4. Inline images (Placeholder):")
	config4 := html.DefaultExtractConfig()
	config4.InlineImageFormat = "placeholder"
	result4, _ := processor.Extract(blogHTML, config4)
	fmt.Printf("   %s\n\n", truncate2(result4.Text, 150))

	// Example 5: Table formats
	fmt.Println("5. Table formats:")
	tableHTML := `
		<html><body>
			<table>
				<tr><th>Name</th><th>Value</th></tr>
				<tr><td>A</td><td>100</td></tr>
			</table>
		</body></html>
	`

	// Markdown tables (default)
	config5a := html.DefaultExtractConfig()
	config5a.TableFormat = "markdown"
	result5a, _ := processor.Extract(tableHTML, config5a)
	fmt.Println("   Markdown format:")
	fmt.Println("   " + result5a.Text)
	fmt.Println()

	// HTML tables
	config5b := html.DefaultExtractConfig()
	config5b.TableFormat = "html"
	result5b, _ := processor.Extract(tableHTML, config5b)
	fmt.Println("   HTML format:")
	fmt.Println("   " + result5b.Text)
	fmt.Println()

	// Example 6: Minimal extraction (no article detection)
	fmt.Println("6. Minimal extraction (all content):")
	config6 := html.ExtractConfig{
		ExtractArticle:    false, // Don't detect article
		PreserveImages:    false,
		PreserveLinks:     false,
		InlineImageFormat: "none",
	}
	result6, _ := processor.Extract(blogHTML, config6)
	fmt.Printf("   Text: %s\n\n", truncate2(result6.Text, 150))

	// Example 7: Preserve all media
	fmt.Println("7. Preserve all media and links:")
	config7 := html.DefaultExtractConfig()
	result7, _ := processor.Extract(blogHTML, config7)
	fmt.Printf("   Images: %d, Links: %d\n", len(result7.Images), len(result7.Links))
	for i, img := range result7.Images {
		if i >= 2 {
			fmt.Printf("   ... and %d more\n", len(result7.Images)-2)
			break
		}
		fmt.Printf("   - %s (alt: %s)\n", img.URL, img.Alt)
	}
}

func truncate2(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
