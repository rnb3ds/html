package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// InlineImages demonstrates different inline image formatting options
// for converting HTML to other formats like Markdown.
func main() {
	processor := html.NewWithDefaults()
	defer processor.Close()

	htmlContent := `
		<article>
			<h1>System Architecture</h1>
			<p>Introduction paragraph explaining the system.</p>
			<img src="https://example.com/architecture.png" alt="System Architecture Diagram">
			<p>As shown in the diagram above, the system consists of three main components.</p>
			<img src="https://example.com/flow.png" alt="Data Flow">
			<p>The data flows through these components as illustrated.</p>
		</article>
	`

	// Example 1: No inline images (default)
	fmt.Println("=== Format: none (default) ===")
	config := html.ExtractConfig{
		PreserveImages:    true,
		InlineImageFormat: "none",
	}
	result, err := processor.Extract(htmlContent, config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Text)
	fmt.Println()

	// Example 2: Placeholder format
	fmt.Println("=== Format: placeholder ===")
	config = html.ExtractConfig{
		PreserveImages:    true,
		InlineImageFormat: "placeholder",
	}
	result, err = processor.Extract(htmlContent, config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Text)
	fmt.Println()

	// Example 3: Markdown format
	fmt.Println("=== Format: markdown ===")
	config = html.ExtractConfig{
		PreserveImages:    true,
		InlineImageFormat: "markdown",
	}
	result, err = processor.Extract(htmlContent, config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Text)
	fmt.Println()

	// Example 4: HTML format
	fmt.Println("=== Format: html ===")
	config = html.ExtractConfig{
		PreserveImages:    true,
		InlineImageFormat: "html",
	}
	result, err = processor.Extract(htmlContent, config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Text)
}
