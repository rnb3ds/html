//go:build examples

package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
	"github.com/cybergodev/html/examples/truncate"
)

// This example demonstrates how to customize content extraction.
// Learn how to control what gets extracted and how it's formatted.
func main() {
	fmt.Println("=== Content Extraction Options ===\n")

	sampleHTML := `
		<html>
			<head><title>Go Interfaces Guide</title></head>
			<body>
				<article>
					<h1>Understanding Go Interfaces</h1>
					<p>Interfaces provide a way to specify behavior.</p>
					<img src="diagram.jpg" alt="Interface Diagram" width="600">
					<a href="https://go.dev/tour/">Go Tour</a>
				</article>
			</body>
		</html>
	`

	// ============================================================
	// 1. Default extraction
	// ============================================================
	fmt.Println("1. Default extraction:")
	processor, _ := html.New()
	defer processor.Close()
	result, _ := processor.Extract([]byte(sampleHTML))
	fmt.Printf("   Words: %d, Images: %d, Links: %d\n\n",
		result.WordCount, len(result.Images), len(result.Links))

	// ============================================================
	// 2. Text-only (using preset config)
	// ============================================================
	fmt.Println("2. Text-only (using TextOnlyConfig()):")
	textOnlyProcessor, _ := html.New(html.TextOnlyConfig())
	defer textOnlyProcessor.Close()
	result, _ = textOnlyProcessor.Extract([]byte(sampleHTML))
	fmt.Printf("   %s\n\n", truncate.Truncate(result.Text, 60))

	// ============================================================
	// 3. Full content with markdown images
	// ============================================================
	fmt.Println("3. Full content (with markdown image format):")
	mdConfig := html.MarkdownConfig()
	mdProcessor, _ := html.New(mdConfig)
	defer mdProcessor.Close()
	result, _ = mdProcessor.Extract([]byte(sampleHTML))
	fmt.Printf("   %s\n\n", truncate.Truncate(result.Text, 80))

	// ============================================================
	// 4. Custom configuration
	// ============================================================
	fmt.Println("4. Custom configuration (images + links, no videos):")
	customConfig := html.DefaultConfig()
	customConfig.PreserveImages = true
	customConfig.PreserveLinks = true
	customConfig.PreserveVideos = false
	customConfig.PreserveAudios = false
	customConfig.ImageFormat = "markdown"
	customProcessor, _ := html.New(customConfig)
	defer customProcessor.Close()
	result, _ = customProcessor.Extract([]byte(sampleHTML))
	fmt.Printf("   %s\n\n", truncate.Truncate(result.Text, 80))

	// ============================================================
	// 5. Image format options
	// ============================================================
	fmt.Println("5. Image format options:")
	imageHTML := `<img src="photo.jpg" alt="Photo">`

	for _, format := range []string{"none", "markdown", "html", "placeholder"} {
		cfg := html.DefaultConfig()
		cfg.ImageFormat = format
		p, _ := html.New(cfg)
		result, _ := p.Extract([]byte(imageHTML))
		fmt.Printf("   %-12s: %s\n", format, truncate.Truncate(result.Text, 40))
		p.Close()
	}

	// ============================================================
	// 6. Encoding specification
	// ============================================================
	fmt.Println("\n6. Encoding specification (for non-UTF-8 HTML):")
	encConfig := html.DefaultConfig()
	encConfig.Encoding = "windows-1252" // Explicit encoding
	_, _ = html.New(encConfig)
	fmt.Println("   Set Encoding field for non-UTF-8 content")
	fmt.Println("   Supported: UTF-8, GBK, Big5, Shift_JIS, Windows-1250/1251/1252, ISO-8859-*")

	// ============================================================
	// 7. Extract from file
	// ============================================================
	fmt.Println("\n7. Extract from file:")
	fmt.Println("   result, err := processor.ExtractFromFile(\"article.html\")")
	fmt.Println("   // Auto-detects encoding (UTF-8, GBK, Shift_JIS, etc.)")

	// ============================================================
	// Summary
	// ============================================================
	fmt.Println("\n=== Configuration Summary ===")
	fmt.Println("• TextOnlyConfig()          - Plain text, no media")
	fmt.Println("• MarkdownConfig()          - Markdown image format")
	fmt.Println("• DefaultConfig()           - All media, customize as needed")
	fmt.Println("• ImageFormat: none | markdown | html | placeholder")
	fmt.Println("• TableFormat:    markdown | html")
	fmt.Println("• Encoding:       Specify for non-UTF-8 content")
}

// Demonstrate file extraction (commented to avoid file not found error)
func demonstrateFileExtraction(processor *html.Processor) {
	result, err := processor.ExtractFromFile("article.html")
	if err != nil {
		log.Printf("File error: %v", err)
		return
	}
	fmt.Printf("File content: %s\n", result.Title)
}
