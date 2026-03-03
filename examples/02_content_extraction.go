//go:build examples

package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// This example demonstrates content extraction options and output formats.
// Learn how to customize extraction and produce different output formats.
func main() {
	fmt.Println("=== Content Extraction & Output Formats ===\n")

	sampleHTML := `
		<html>
			<head><title>Go Interfaces Guide</title></head>
			<body>
				<article>
					<h1>Understanding Go Interfaces</h1>
					<p>Interfaces provide a way to specify behavior.</p>
					<img src="diagram.jpg" alt="Interface Diagram" width="600">
					<a href="https://go.dev/tour/">Go Tour</a>
					<table>
						<tr><th>Feature</th><th>Benefit</th></tr>
						<tr><td>Goroutines</td><td>Lightweight concurrency</td></tr>
						<tr><td>Channels</td><td>Safe communication</td></tr>
					</table>
				</article>
			</body>
		</html>
	`

	// ============================================================
	// 1. Preset Configurations
	// ============================================================
	fmt.Println("1. Preset Configurations")
	fmt.Println("-----------------------")

	// Default: all media preserved
	fmt.Println("DefaultConfig():  All media (images, links, videos, audios)")

	// Text-only: no media
	textOnlyProcessor, _ := html.New(html.TextOnlyConfig())
	defer textOnlyProcessor.Close()
	result, _ := textOnlyProcessor.Extract([]byte(sampleHTML))
	fmt.Printf("TextOnlyConfig(): %d chars, %d images\n\n", len(result.Text), len(result.Images))

	// ============================================================
	// 2. Custom Configuration
	// ============================================================
	fmt.Println("2. Custom Configuration")
	fmt.Println("-----------------------")

	customConfig := html.DefaultConfig()
	customConfig.PreserveImages = true
	customConfig.PreserveLinks = true
	customConfig.PreserveVideos = false
	customConfig.PreserveAudios = false
	customConfig.ImageFormat = "markdown"
	customConfig.LinkFormat = "markdown"

	customProcessor, _ := html.New(customConfig)
	defer customProcessor.Close()
	result, _ = customProcessor.Extract([]byte(sampleHTML))
	fmt.Printf("Images: %d, Links: %d\n\n", len(result.Images), len(result.Links))

	// ============================================================
	// 3. Image Format Options
	// ============================================================
	fmt.Println("3. Image Format Options")
	fmt.Println("-----------------------")

	imageHTML := `<img src="photo.jpg" alt="Photo" width="800">`
	formats := []string{"none", "markdown", "html", "placeholder"}

	for _, format := range formats {
		cfg := html.DefaultConfig()
		cfg.ImageFormat = format
		p, _ := html.New(cfg)
		r, _ := p.Extract([]byte(imageHTML))
		fmt.Printf("  %-12s: %s\n", format, r.Text)
		p.Close()
	}

	// ============================================================
	// 4. Table Format Options
	// ============================================================
	fmt.Println("\n4. Table Format Options")
	fmt.Println("-----------------------")

	// Markdown table (default)
	mdProcessor, _ := html.New(html.MarkdownConfig())
	defer mdProcessor.Close()
	mdResult, _ := mdProcessor.Extract([]byte(sampleHTML))
	fmt.Println("Markdown table:")
	fmt.Println(mdResult.Text[:min(150, len(mdResult.Text))] + "...")

	// ============================================================
	// 5. JSON Output
	// ============================================================
	fmt.Println("\n5. JSON Output")
	fmt.Println("--------------")

	processor, _ := html.New()
	defer processor.Close()

	jsonBytes, err := processor.ExtractToJSON([]byte(sampleHTML))
	if err != nil {
		log.Fatal(err)
	}

	var data map[string]interface{}
	json.Unmarshal(jsonBytes, &data)
	pretty, _ := json.MarshalIndent(data, "", "  ")
	fmt.Printf("%s\n", string(pretty[:min(400, len(pretty))])+"...")

	// ============================================================
	// 6. Markdown Output
	// ============================================================
	fmt.Println("\n6. Markdown Output")
	fmt.Println("------------------")

	markdown, _ := processor.ExtractToMarkdown([]byte(sampleHTML))
	fmt.Printf("%s\n", markdown[:min(200, len(markdown))]+"...")

	// ============================================================
	// 7. File Operations
	// ============================================================
	fmt.Println("\n7. File Operations")
	fmt.Println("------------------")
	fmt.Println("  processor.ExtractFromFile(\"article.html\")")
	fmt.Println("  processor.ExtractToJSONFromFile(\"article.html\")")
	fmt.Println("  processor.ExtractToMarkdownFromFile(\"article.html\")")

	// ============================================================
	// 8. Encoding Support
	// ============================================================
	fmt.Println("\n8. Encoding Support")
	fmt.Println("-------------------")
	fmt.Println("  Auto-detects: UTF-8, GBK, Big5, Shift_JIS, Windows-1250/1251/1252, ISO-8859-*")
	fmt.Println("  Specify explicitly:")
	fmt.Println("    cfg := html.DefaultConfig()")
	fmt.Println("    cfg.Encoding = \"windows-1252\"")

	// ============================================================
	// Summary
	// ============================================================
	fmt.Println("\n=== Quick Reference ===")
	fmt.Println("Preset configs: DefaultConfig(), TextOnlyConfig(), MarkdownConfig(), HighSecurityConfig()")
	fmt.Println("Image formats:  none | markdown | html | placeholder")
	fmt.Println("Link formats:   none | markdown | html")
	fmt.Println("Table formats:  markdown | html")
	fmt.Println("Output methods: Extract(), ExtractToJSON(), ExtractToMarkdown()")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
