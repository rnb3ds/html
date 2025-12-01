//go:build examples

package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// BasicExtraction demonstrates simple content extraction from HTML.
func main() {
	processor := html.NewWithDefaults()
	defer processor.Close()

	htmlContent := `
		<html>
		<body>
			<nav>Skip this navigation</nav>
			<article>
				<h1>10 Tips for Better Go Code</h1>
				<p>Go is a powerful language that emphasizes simplicity...</p>
				<img src="https://example.com/diagram.png" alt="Architecture Diagram" width="800">
				<p>The key principles include...</p>
			</article>
			<aside>Advertisement</aside>
		</body>
		</html>
	`

	result, err := processor.ExtractWithDefaults(htmlContent)
	if err != nil {
		log.Fatal(err)
	}

	// Display extracted content
	fmt.Println("Title:", result.Title)
	fmt.Println("Text:", result.Text)
	fmt.Println("Word Count:", result.WordCount)
	fmt.Println("Reading Time:", result.ReadingTime)
	fmt.Println("Images:", len(result.Images))

	// Display image metadata
	for _, img := range result.Images {
		fmt.Printf("\nImage: %s (%s x %s)\n", img.URL, img.Width, img.Height)
		fmt.Printf("Alt: %s\n", img.Alt)
	}
}
