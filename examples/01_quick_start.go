//go:build examples

package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/cybergodev/html"
)

// This example demonstrates the fastest way to get started with the library.
// Perfect for first-time users who want to see immediate results.
func main() {
	fmt.Println("=== Quick Start ===\n")
	fmt.Println("Learn cybergodev/html in 3 simple steps:\n")

	// ===== STEP 1: Extract text only (simplest) =====
	fmt.Println("Step 1: Extract plain text")
	fmt.Println("----------------------------")

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

	fmt.Printf("Input HTML:\n%s\n\n", strings.TrimSpace(htmlContent))
	fmt.Printf("Extracted Text:\n%s\n\n", text)

	// ===== STEP 2: Extract with metadata =====
	fmt.Println("\nStep 2: Extract title and metadata")
	fmt.Println("-----------------------------------")

	result, err := html.Extract([]byte(htmlContent))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Title: %s\n", result.Title)
	fmt.Printf("Word Count: %d\n", result.WordCount)
	fmt.Printf("Reading Time: %d minute(s)\n\n", result.ReadingTime)

	// ===== STEP 3: Extract from real-world HTML =====
	fmt.Println("Step 3: Extract from real-world page (with noise removal)")
	fmt.Println("-----------------------------------------------------------")

	blogHTML := `
		<html>
			<head><title>My Blog Post</title></head>
			<body>
				<nav>Home • About • Contact</nav>
				<div class="sidebar">
					<div class="ad">Advertisement</div>
					<div class="related">Related Posts</div>
				</div>
				<main>
					<article>
						<h1>Understanding Go Concurrency</h1>
						<p>Go makes concurrent programming easy with goroutines.</p>
						<img src="goroutine.png" alt="Goroutine Diagram">
					</article>
				</main>
				<footer>© 2024 My Blog</footer>
			</body>
		</html>
	`

	result, err = html.Extract([]byte(blogHTML))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Title: %s\n", result.Title)
	fmt.Printf("Content: %s\n", truncate01(result.Text, 100))
	fmt.Printf("Images Found: %d (noise removed: nav, sidebar, footer)\n", len(result.Images))

	// ===== BONUS: Reuse processor for multiple documents =====
	fmt.Println("\n\nBonus: Process multiple documents efficiently")
	fmt.Println("----------------------------------------------")

	processor, _ := html.New()
	defer processor.Close()

	docs := []string{
		`<article><h1>First Post</h1><p>Content 1</p></article>`,
		`<article><h1>Second Post</h1><p>Content 2</p></article>`,
		`<article><h1>Third Post</h1><p>Content 3</p></article>`,
	}

	for i, doc := range docs {
		result, err := processor.Extract([]byte(doc))
		if err != nil {
			continue
		}
		fmt.Printf("Document %d: %s (%d words)\n", i+1, result.Title, result.WordCount)
	}
}

// truncate01 shortens text for display
func truncate01(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
