//go:build examples

package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// ArticleDetection demonstrates intelligent article content extraction
// that automatically removes navigation, ads, sidebars, and footer noise.
func main() {
	processor := html.NewWithDefaults()
	defer processor.Close()

	// Complex page with navigation, ads, sidebars
	htmlContent := `
		<html>
		<nav>Site Navigation | Home | About | Contact</nav>
		<aside class="sidebar">
			<div class="ad">Advertisement - Buy Now!</div>
			<div class="related">Related Posts...</div>
		</aside>
		<article>
			<h1>Understanding Go Interfaces</h1>
			<p>Interfaces are one of Go's most powerful features. They provide a way to specify the behavior of an object.</p>
			<p>An interface type is defined as a set of method signatures. A value of interface type can hold any value that implements those methods.</p>
		</article>
		<footer>© 2024 Example Site | Privacy Policy | Terms</footer>
		</html>
	`

	config := html.ExtractConfig{
		ExtractArticle: true, // Enable smart content detection
	}

	result, err := processor.Extract(htmlContent, config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Article Detection Example ===")
	fmt.Println("\nExtracted Title:", result.Title)
	fmt.Println("\nExtracted Text (navigation, ads, sidebar, and footer removed):")
	fmt.Println(result.Text)
	fmt.Println("\nWord Count:", result.WordCount)
	fmt.Println("Reading Time:", result.ReadingTime)
}
