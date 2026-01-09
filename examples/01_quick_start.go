//go:build examples

package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// QuickStart demonstrates the simplest way to extract content from HTML.
// This is the recommended starting point for new users.
func main() {
	fmt.Println("=== Quick Start Example ===\n ")

	// Example 1: Extract content with defaults (simplest usage)
	fmt.Println("1. Simple extraction with defaults:")
	htmlContent := `
		<html>
		<body>
			<nav>Skip this navigation</nav>
			<article>
				<h1>Getting Started with Go</h1>
				<p>Go is a powerful language that emphasizes simplicity and efficiency.</p>
				<img src="https://example.com/diagram.png" alt="Architecture Diagram" width="800">
				<p>The key principles include clear syntax, fast compilation, and built-in concurrency.</p>
			</article>
			<aside>Advertisement</aside>
		</body>
		</html>
	`

	result, err := html.Extract(htmlContent)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Title: %s\n", result.Title)
	fmt.Printf("   Text: %s\n", result.Text)
	fmt.Printf("   Word Count: %d\n", result.WordCount)
	fmt.Printf("   Reading Time: %v\n", result.ReadingTime)
	fmt.Printf("   Images: %d\n\n", len(result.Images))

	// Example 2: Extract only text (even simpler)
	fmt.Println("2. Extract text only:")
	text, err := html.ExtractText(htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   %s\n\n", text)

	// Example 3: Extract from file
	fmt.Println("3. Extract from file:")
	// Create a temporary HTML file for demonstration
	tmpFile := "temp_example.html"
	if err := saveToFile(tmpFile, htmlContent); err != nil {
		log.Printf("   Skipping file example: %v\n\n", err)
	} else {
		defer removeFile(tmpFile)
		fileResult, err := html.ExtractFromFile(tmpFile)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("   Title: %s\n", fileResult.Title)
		fmt.Printf("   Word Count: %d\n\n", fileResult.WordCount)
	}

	// Example 4: Using processor for multiple operations
	fmt.Println("4. Using processor for multiple operations:")
	processor := html.NewWithDefaults()
	defer processor.Close()

	// Process multiple documents
	docs := []string{
		`<article><h1>Article 1</h1><p>First article content.</p></article>`,
		`<article><h1>Article 2</h1><p>Second article content.</p></article>`,
	}

	for i, doc := range docs {
		result, err := processor.ExtractWithDefaults(doc)
		if err != nil {
			log.Printf("   Error processing document %d: %v\n", i+1, err)
			continue
		}
		fmt.Printf("   Document %d: %s (%d words)\n", i+1, result.Title, result.WordCount)
	}

	fmt.Println("\n✓ Quick start complete! See other examples for advanced features.")
}

// Helper functions
func saveToFile(path, content string) error {
	// Implementation would use os.WriteFile
	return nil // Simplified for example
}

func removeFile(path string) {
	// Implementation would use os.Remove
}
