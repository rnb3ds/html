//go:build examples

package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// This example demonstrates different output formats.
// Learn how to convert HTML to JSON, Markdown, and more.
func main() {
	fmt.Println("=== Output Formats: JSON & Markdown ===\n ")

	htmlContent := `
		<html>
			<head><title>Go Concurrency</title></head>
			<body>
				<article>
					<h1>Understanding Goroutines</h1>
					<p>Goroutines are lightweight threads.</p>
					<img src="diagram.png" alt="Goroutine Diagram">
					<table>
						<tr><th>Feature</th><th>Benefit</th></tr>
						<tr><td>Goroutines</td><td>Concurrency</td></tr>
						<tr><td>Channels</td><td>Communication</td></tr>
					</table>
				</article>
			</body>
		</html>
	`

	processor, _ := html.New()
	defer processor.Close()

	// ============================================================
	// Part 1: JSON output
	// ============================================================
	fmt.Println("Part 1: JSON Output")
	fmt.Println("-------------------")

	jsonBytes, err := processor.ExtractToJSON([]byte(htmlContent))
	if err != nil {
		log.Fatal(err)
	}

	// Pretty print
	var data map[string]interface{}
	json.Unmarshal(jsonBytes, &data)
	pretty, _ := json.MarshalIndent(data, "", "  ")
	fmt.Printf("JSON Output:\n%s\n\n", string(pretty))

	fmt.Println("✓ Structured data perfect for:")
	fmt.Println("  • APIs")
	fmt.Println("  • Databases")
	fmt.Println("  • Data pipelines")

	// ============================================================
	// Part 2: Markdown output
	// ============================================================
	fmt.Println("\nPart 2: Markdown Output")
	fmt.Println("-----------------------")

	markdown, err := processor.ExtractToMarkdown([]byte(htmlContent))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Markdown Output:\n%s\n\n", markdown)

	fmt.Println("✓ Ready for:")
	fmt.Println("  • Documentation")
	fmt.Println("  • README files")
	fmt.Println("  • Static site generators")

	// ============================================================
	// Part 3: Table format options
	// ============================================================
	fmt.Println("\nPart 3: Table Formats")
	fmt.Println("---------------------")

	tableHTML := `
		<html><body>
			<table>
				<tr><th>Product</th><th>Price</th></tr>
				<tr><td>Go Book</td><td>$29.99</td></tr>
				<tr><td>Go Course</td><td>$49.99</td></tr>
			</table>
		</body></html>
	`

	for _, format := range []string{"markdown", "html"} {
		config := html.DefaultExtractConfig()
		config.TableFormat = format

		result, err := processor.Extract([]byte(tableHTML), config)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("\nFormat: %s\n", format)
		fmt.Printf("Output:\n%s\n", result.Text)
	}

	fmt.Println("\n✓ Markdown: Human-readable tables")
	fmt.Println("✓ HTML: Preserves structure")
}
