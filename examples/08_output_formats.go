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
					<table>
						<tr><th>Feature</th><th>Benefit</th></tr>
						<tr><td>Goroutines</td><td>Concurrency</td></tr>
					</table>
				</article>
			</body>
		</html>
	`

	// ============================================================
	// Part 1: JSON output
	// ============================================================
	fmt.Println("Part 1: JSON Output")
	fmt.Println("----------------")

	jsonBytes, err := html.ExtractToJSON([]byte(htmlContent))
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
	fmt.Println("----------------------")

	markdown, err := html.ExtractToMarkdown([]byte(htmlContent))
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
	fmt.Println("------------------")

	tableHTML := `<html><body><table><tr><th>A</th><tr><td>B</td></tr></table></body></html>`

	tableFormats := []string{"markdown", "html"}

	for _, format := range tableFormats {
		config := html.DefaultExtractConfig()
		config.TableFormat = format

		processor, _ := html.New()
		defer processor.Close()

		result, err := processor.Extract([]byte(tableHTML), config)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("\nFormat: %s\n", format)
		fmt.Printf("Output:  %s\n\n", result.Text)
	}

	fmt.Println("✓ Markdown: Human-readable tables")
	fmt.Println("✓ HTML:    Preserves structure")

}
