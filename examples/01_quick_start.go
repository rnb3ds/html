//go:build examples

package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// QuickStart demonstrates the simplest ways to extract content from HTML.
// This is the recommended starting point for new users.
func main() {
	fmt.Println("=== Quick Start Guide ===\n ")

	// Example 1: Simplest extraction - just get the text
	fmt.Println("1. Extract text only (simplest):")
	htmlContent := `
		<html>
		<body>
			<article>
				<h1>Getting Started with Go</h1>
				<p>Go is a powerful language that emphasizes simplicity.</p>
			</article>
		</body>
		</html>
	`

	text, err := html.ExtractText(htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Text: %s\n\n", text)

	// Example 2: Extract with all metadata (recommended for most use cases)
	fmt.Println("2. Extract with metadata:")
	result, err := html.Extract(htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Title: %s\n", result.Title)
	fmt.Printf("   Word Count: %d\n", result.WordCount)
	fmt.Printf("   Reading Time: %v\n\n", result.ReadingTime)

	// Example 3: Real-world HTML with navigation and ads
	fmt.Println("3. Smart article extraction (removes noise):")
	blogHTML := `
		<html>
		<body>
			<nav>Menu • About • Contact</nav>
			<aside>Sidebar Ad</aside>
			<article>
				<h1>Understanding Go Concurrency</h1>
				<p>Go's goroutines make concurrent programming easy.</p>
				<img src="diagram.png" alt="Concurrency Diagram">
			</article>
			<footer>© 2024</footer>
		</body>
		</html>
	`

	result, err = html.Extract(blogHTML)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Title: %s\n", result.Title)
	fmt.Printf("   Text: %s\n", result.Text)
	fmt.Printf("   Images: %d\n\n", len(result.Images))

	// Example 4: Using processor for multiple extractions
	fmt.Println("4. Process multiple documents:")
	processor := html.NewWithDefaults()
	defer processor.Close()

	docs := []string{
		`<article><h1>First Post</h1><p>Content here.</p></article>`,
		`<article><h1>Second Post</h1><p>More content.</p></article>`,
	}

	for i, doc := range docs {
		result, err := processor.ExtractWithDefaults(doc)
		if err != nil {
			log.Printf("   Error processing doc %d: %v\n", i+1, err)
			continue
		}
		fmt.Printf("   Doc %d: %s (%d words)\n", i+1, result.Title, result.WordCount)
	}

}
