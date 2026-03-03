//go:build examples

package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// This example demonstrates the fastest way to get started with the library.
// Perfect for first-time users who want to see immediate results.
func main() {
	fmt.Println("=== Quick Start ===\n")

	// ============================================================
	// 1. Extract plain text (simplest approach)
	// ============================================================
	htmlContent := `
		<html>
			<body>
				<h1>Welcome to Go</h1>
				<p>Go is a powerful programming language.</p>
			</body>
		</html>
	`

	text, err := html.ExtractText([]byte(htmlContent))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("1. Plain text:\n   %s\n\n", text)

	// ============================================================
	// 2. Extract with metadata (title, word count, etc.)
	// ============================================================
	result, err := html.Extract([]byte(htmlContent))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("2. With metadata:\n")
	fmt.Printf("   Title: %s\n", result.Title)
	fmt.Printf("   Words: %d\n", result.WordCount)
	fmt.Printf("   Reading time: %v\n\n", result.ReadingTime)

	// ============================================================
	// 3. Reuse processor for multiple documents (efficient)
	// ============================================================
	processor, _ := html.New()
	defer processor.Close()

	docs := []string{
		`<article><h1>First Post</h1><p>Content 1</p></article>`,
		`<article><h1>Second Post</h1><p>Content 2</p></article>`,
	}

	fmt.Println("3. Process multiple documents:")
	for i, doc := range docs {
		result, err := processor.Extract([]byte(doc))
		if err != nil {
			continue
		}
		fmt.Printf("   Doc %d: %s (%d words)\n", i+1, result.Title, result.WordCount)
	}

	fmt.Println("\n✓ Ready to explore more? Check the other examples!")
}
