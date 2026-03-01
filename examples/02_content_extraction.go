//go:build examples

package main

import (
	"fmt"

	"github.com/cybergodev/html"
	"github.com/cybergodev/html/examples/truncate"
)

// This example demonstrates how to customize content extraction.
// Learn how to control what gets extracted and how it's formatted.
func main() {
	fmt.Println("=== Content Extraction Options ===\n ")

	// Sample HTML with various elements
	sampleHTML := `
		<html>
			<head><title>Go Interfaces Guide</title></head>
			<body>
				<nav>Menu • About • Contact</nav>
				<aside>
					<div class="ad">Sponsored Content</div>
				</aside>
				<main>
					<article>
						<h1>Understanding Go Interfaces</h1>
						<p>Interfaces provide a way to specify the behavior of an object.</p>
						<img src="diagram.jpg" alt="Interface Diagram" width="600" height="400">
						<h2>Key Benefits</h2>
						<p>Decoupling and flexibility are the main advantages.</p>
					</article>
				</main>
				<footer>Copyright 2024</footer>
			</body>
		</html>
	`

	processor, _ := html.New()
	defer processor.Close()

	// ============================================================
	// OPTION 1: Quick text extraction
	// ============================================================
	fmt.Println("Option 1: Quick Text Extraction")
	fmt.Println("--------------------------------")

	// Use default config for text extraction
	result, _ := processor.Extract([]byte(sampleHTML))
	fmt.Printf("Default: %d words, %d images, %d links\n",
		result.WordCount, len(result.Images), len(result.Links))

	// ============================================================
	// OPTION 2: Text only (no media)
	// ============================================================
	fmt.Println("\nOption 2: Text Only")
	fmt.Println("-------------------")

	textOnlyConfig := html.ExtractConfig{
		ExtractArticle:    true,
		PreserveImages:    false,
		PreserveLinks:     false,
		PreserveVideos:    false,
		PreserveAudios:    false,
		InlineImageFormat: "none",
	}
	result, _ = processor.Extract([]byte(sampleHTML), textOnlyConfig)
	fmt.Printf("Text only: %d words, %d images\n", result.WordCount, len(result.Images))

	// ============================================================
	// OPTION 3: Full content with markdown images
	// ============================================================
	fmt.Println("\nOption 3: Full Content with Markdown")
	fmt.Println("------------------------------------")

	fullConfig := html.ExtractConfig{
		ExtractArticle:    true,
		PreserveImages:    true,
		PreserveLinks:     true,
		PreserveVideos:    true,
		PreserveAudios:    true,
		InlineImageFormat: "markdown",
		TableFormat:       "markdown",
	}
	result, _ = processor.Extract([]byte(sampleHTML), fullConfig)
	fmt.Printf("Full content: %s\n", truncate.Truncate(result.Text, 80))

	// ============================================================
	// OPTION 4: Image embedding formats
	// ============================================================
	fmt.Println("\nOption 4: Image Embedding Formats")
	fmt.Println("---------------------------------")

	imageFormats := []struct {
		name   string
		format string
	}{
		{"none", "none"},
		{"markdown", "markdown"},
		{"html", "html"},
		{"placeholder", "placeholder"},
	}

	for _, f := range imageFormats {
		config := html.DefaultExtractConfig()
		config.InlineImageFormat = f.format
		result, _ = processor.Extract([]byte(sampleHTML), config)
		fmt.Printf("  %s: %s\n", f.name, truncate.Truncate(result.Text, 50))
	}

	// ============================================================
	// OPTION 5: Table formatting
	// ============================================================
	fmt.Println("\nOption 5: Table Formatting")
	fmt.Println("--------------------------")

	tableHTML := `<html><body>
		<table>
			<tr><th>Product</th><th>Price</th></tr>
			<tr><td>Go Book</td><td>$29.99</td></tr>
		</table>
	</body></html>`

	for _, format := range []string{"markdown", "html"} {
		config := html.DefaultExtractConfig()
		config.TableFormat = format
		result, _ = processor.Extract([]byte(tableHTML), config)
		fmt.Printf("\nFormat: %s\n%s\n", format, result.Text)
	}

	// ============================================================
	// OPTION 6: Encoding specification
	// ============================================================
	fmt.Println("\nOption 6: Encoding Specification")
	fmt.Println("--------------------------------")

	// For non-UTF-8 HTML, specify the encoding
	encodedHTML := `<html><body><h1>Test</h1><p>Content</p></body></html>`

	encodingConfig := html.DefaultExtractConfig()
	encodingConfig.Encoding = "windows-1252" // Specify encoding explicitly
	result, _ = processor.Extract([]byte(encodedHTML), encodingConfig)
	fmt.Printf("With encoding: %s\n", result.Title)

	fmt.Println("\n✓ You now know all extraction options!")
}
