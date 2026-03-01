//go:build examples

package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// This example demonstrates different output formats: JSON, Markdown, and table formatting.
func main() {
	fmt.Println("=== Output Formats ===\n")

	htmlContent := `
		<html>
			<head><title>Go Concurrency</title></head>
			<body>
				<article>
					<h1>Understanding Goroutines</h1>
					<p>Goroutines are lightweight threads managed by the Go runtime.</p>
					<img src="diagram.png" alt="Goroutine Diagram">
					<table>
						<tr><th>Feature</th><th>Benefit</th></tr>
						<tr><td>Goroutines</td><td>Lightweight concurrency</td></tr>
						<tr><td>Channels</td><td>Safe communication</td></tr>
					</table>
				</article>
			</body>
		</html>
	`

	processor, _ := html.New()
	defer processor.Close()

	// ============================================================
	// 1. JSON Output
	// ============================================================
	fmt.Println("1. JSON Output")
	fmt.Println("---------------")

	jsonBytes, err := processor.ExtractToJSON([]byte(htmlContent))
	if err != nil {
		log.Fatal(err)
	}

	var data map[string]interface{}
	json.Unmarshal(jsonBytes, &data)
	pretty, _ := json.MarshalIndent(data, "", "  ")
	fmt.Printf("%s\n\n", string(pretty))

	// ============================================================
	// 2. Markdown Output
	// ============================================================
	fmt.Println("2. Markdown Output")
	fmt.Println("-------------------")

	markdown, err := processor.ExtractToMarkdown([]byte(htmlContent))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n\n", markdown)

	// ============================================================
	// 3. Table Formats
	// ============================================================
	fmt.Println("3. Table Formats")
	fmt.Println("-----------------")

	tableHTML := `
		<html><body>
			<table>
				<tr><th>Package</th><th>Purpose</th></tr>
				<tr><td>fmt</td><td>Formatting</td></tr>
				<tr><td>json</td><td>JSON encoding</td></tr>
			</table>
		</body></html>
	`

	// Markdown table format
	mdConfig := html.DefaultExtractConfig()
	mdConfig.TableFormat = "markdown"
	mdResult, _ := processor.Extract([]byte(tableHTML), mdConfig)
	fmt.Println("Markdown table:")
	fmt.Println(mdResult.Text)

	// HTML table format
	htmlConfig := html.DefaultExtractConfig()
	htmlConfig.TableFormat = "html"
	htmlResult, _ := processor.Extract([]byte(tableHTML), htmlConfig)
	fmt.Println("HTML table:")
	fmt.Println(htmlResult.Text)

	// ============================================================
	// 4. Inline Image Formats
	// ============================================================
	fmt.Println("4. Inline Image Formats")
	fmt.Println("-----------------------")

	imageHTML := `
		<html><body>
			<h1>Article</h1>
			<p>Text before image.</p>
			<img src="photo.jpg" alt="Sample Photo">
			<p>Text after image.</p>
		</body></html>
	`

	formats := []struct {
		name   string
		format string
	}{
		{"none", "none"},
		{"markdown", "markdown"},
		{"html", "html"},
		{"placeholder", "placeholder"},
	}

	for _, f := range formats {
		config := html.DefaultExtractConfig()
		config.InlineImageFormat = f.format
		result, _ := processor.Extract([]byte(imageHTML), config)
		fmt.Printf("\n%s format:\n%s\n", f.name, result.Text)
	}

	// ============================================================
	// Summary
	// ============================================================
	fmt.Println("\n=== Format Options ===")
	fmt.Println("Output functions:")
	fmt.Println("  ExtractToJSON()     - Structured JSON data")
	fmt.Println("  ExtractToMarkdown() - Markdown with tables")
	fmt.Println()
	fmt.Println("Table formats:")
	fmt.Println("  markdown - GitHub-flavored markdown tables")
	fmt.Println("  html     - Preserved HTML table structure")
	fmt.Println()
	fmt.Println("Image formats:")
	fmt.Println("  none       - No images in text output")
	fmt.Println("  markdown   - ![alt](src) syntax")
	fmt.Println("  html       - <img> tag preserved")
	fmt.Println("  placeholder - [IMAGE: alt] placeholder")
}
