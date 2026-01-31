//go:build examples

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/cybergodev/html"
)

// OutputFormats demonstrates JSON and Markdown output options.
func main() {
	fmt.Println("=== Output Formats ===\n")

	htmlContent := `
		<html>
		<head><title>Go Concurrency Tutorial</title></head>
		<body>
			<article>
				<h1>Understanding Goroutines</h1>
				<p>Goroutines are lightweight threads.</p>
				<img src="goroutine.png" alt="Goroutine Diagram">
				<h2>Key Benefits</h2>
				<ul>
					<li>Low memory footprint</li>
					<li>Fast creation</li>
					<li>Efficient communication via channels</li>
				</ul>
				<table>
					<tr><th>Feature</th><th>Benefit</th></tr>
					<tr><td>Goroutines</td><td>Concurrent execution</td></tr>
					<tr><td>Channels</td><td>Safe communication</td></tr>
				</table>
			</article>
		</body>
		</html>
	`

	// Example 1: Extract to JSON (global function)
	fmt.Println("1. Extract to JSON:")
	jsonBytes, err := html.ExtractToJSON(htmlContent)
	if err != nil {
		log.Fatal(err)
	}

	// Pretty print JSON
	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &prettyJSON); err == nil {
		formatted, _ := json.MarshalIndent(prettyJSON, "   ", "  ")
		fmt.Printf("   %s\n\n", string(formatted))
	}

	// Example 2: Extract to Markdown (global function)
	fmt.Println("2. Extract to Markdown:")
	markdown, err := html.ExtractToMarkdown(htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   %s\n\n", truncate5(markdown, 200))

	// Example 3: Extract with custom configuration
	fmt.Println("3. Extract with custom configuration:")
	processor := html.NewWithDefaults()
	defer processor.Close()

	customConfig := html.DefaultExtractConfig()
	customConfig.PreserveImages = false
	customConfig.PreserveLinks = false
	customConfig.TableFormat = "html"

	result, err := processor.Extract(htmlContent, customConfig)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Title: %s\n", result.Title)
	fmt.Printf("   Word Count: %d\n", result.WordCount)
	fmt.Printf("   Reading Time: %v\n\n", result.ReadingTime)

	// Example 4: Markdown with inline images using Processor
	fmt.Println("4. Markdown with inline images (via Processor):")
	imageConfig := html.DefaultExtractConfig()
	imageConfig.InlineImageFormat = "markdown"

	result2, err := processor.Extract(htmlContent, imageConfig)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   %s\n\n", truncate5(result2.Text, 200))

	// Example 5: JSON from processor result
	fmt.Println("5. JSON from processor result:")
	result3, err := processor.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		log.Fatal(err)
	}

	// Convert Result to JSON manually
	jsonMap := map[string]interface{}{
		"title":       result3.Title,
		"text":        truncate5(result3.Text, 100),
		"wordCount":   result3.WordCount,
		"readingTime": result3.ReadingTime.String(),
		"images":      len(result3.Images),
		"links":       len(result3.Links),
	}
	jsonBytes2, _ := json.MarshalIndent(jsonMap, "   ", "  ")
	fmt.Printf("   %s\n\n", string(jsonBytes2))

	// Example 6: Compare outputs
	fmt.Println("6. Output format comparison:")
	fmt.Println("   Markdown vs JSON output:")

	markdownResult, _ := html.ExtractToMarkdown(htmlContent)
	jsonResult, _ := html.ExtractToJSON(htmlContent)

	fmt.Printf("   Markdown length: %d chars\n", len(markdownResult))
	fmt.Printf("   JSON length: %d bytes\n", len(jsonResult))
	fmt.Printf("   Markdown preview: %s\n", truncate5(markdownResult, 80))
	fmt.Printf("   JSON preview: %s\n\n", truncate5(string(jsonResult), 80))

}

func truncate5(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
