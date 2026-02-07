//go:build examples

package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
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
					<div class="related">You might also like...</div>
				</aside>
				<main>
					<article>
						<h1>Understanding Go Interfaces</h1>
						<p class="author">By Jane Doe • March 15, 2024</p>
						<p>Interfaces provide a way to specify the behavior of an object.</p>
						<img src="diagram.jpg" alt="Interface Diagram" width="600" height="400">
						<h2>Key Benefits</h2>
						<p>Decoupling and flexibility are the main advantages.</p>
						<table>
							<tr><th>Concept</th><th>Benefit</th></tr>
							<tr><td>Polymorphism</td><td>Code reuse</td></tr>
							<tr><td>Abstraction</td><td>Hiding complexity</td></tr>
						</table>
					</article>
				</main>
				<footer>Copyright 2024</footer>
			</body>
		</html>
	`

	processor, _ := html.New()
	defer processor.Close()

	// ============================================================
	// OPTION 1: Control image embedding in text
	// ============================================================
	fmt.Println("Option 1: Image Embedding in Text")
	fmt.Println("----------------------------------")

	imageFormats := []string{"none", "markdown", "html", "placeholder"}

	for _, format := range imageFormats {
		config := html.DefaultExtractConfig()
		config.InlineImageFormat = format

		result, err := processor.Extract([]byte(sampleHTML), config)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("\nFormat: %s\n", format)
		fmt.Printf("Output:  %s\n", truncate02(result.Text, 80))
	}

	// ============================================================
	// OPTION 2: Control table formatting
	// ============================================================
	fmt.Println("\n\nOption 2: Table Formatting")
	fmt.Println("--------------------------")

	tableHTML := `
		<html><body>
			<table>
				<tr><th>Product</th><th>Price</th></tr>
				<tr><td>Go Book</td><td>$29.99</td></tr>
			</table>
		</body></html>
	`

	tableFormats := []string{"markdown", "html"}

	for _, format := range tableFormats {
		config := html.DefaultExtractConfig()
		config.TableFormat = format

		result, err := processor.Extract([]byte(tableHTML), config)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("\nFormat: %s\n", format)
		fmt.Printf("Output:\n%s\n", result.Text)
	}

	// ============================================================
	// OPTION 3: Choose what to extract
	// ============================================================
	fmt.Println("\nOption 3: Selective Content Extraction")
	fmt.Println("--------------------------------------")

	extractOptions := []struct {
		name    string
		config  html.ExtractConfig
		comment string
	}{
		{
			name: "Text only (fastest)",
			config: html.ExtractConfig{
				ExtractArticle: true,
				PreserveImages: false,
				PreserveLinks:  false,
				PreserveVideos: false,
				PreserveAudios: false,
			},
			comment: "Extract only the main text content",
		},
		{
			name: "Article with images",
			config: html.ExtractConfig{
				ExtractArticle: true,
				PreserveImages: true,
				PreserveLinks:  false,
			},
			comment: "Extract article and keep image references",
		},
		{
			name: "Full content (images + links)",
			config: html.ExtractConfig{
				ExtractArticle: true,
				PreserveImages: true,
				PreserveLinks:  true,
			},
			comment: "Extract everything except media files",
		},
		{
			name: "Everything (includes videos/audio)",
			config: html.ExtractConfig{
				ExtractArticle: true,
				PreserveImages: true,
				PreserveLinks:  true,
				PreserveVideos: true,
				PreserveAudios: true,
			},
			comment: "Extract all content types",
		},
	}

	for _, opt := range extractOptions {
		result, err := processor.Extract([]byte(sampleHTML), opt.config)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("\n%s:\n", opt.name)
		fmt.Printf("  %s\n", opt.comment)
		fmt.Printf("  Title: %s\n", result.Title)
		fmt.Printf("  Images: %d, Links: %d, Videos: %d\n",
			len(result.Images), len(result.Links), len(result.Videos))
	}

	// ============================================================
	// OPTION 4: Extract specific content only
	// ============================================================
	fmt.Println("\n\nOption 4: Extract Specific Content Types")
	fmt.Println("----------------------------------------")

	// Extract only the article (removes nav, sidebar, footer)
	config := html.DefaultExtractConfig()
	config.ExtractArticle = true

	result, err := processor.Extract([]byte(sampleHTML), config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Article extraction (noise removed):")
	fmt.Printf("  Title: %s\n", result.Title)
	fmt.Printf("  Text preview: %s\n", truncate02(result.Text, 100))
	fmt.Println("\n  ✓ Navigation removed")
	fmt.Println("  ✓ Sidebar with ads removed")
	fmt.Println("  ✓ Footer removed")

}

func truncate02(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
