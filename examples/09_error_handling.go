//go:build examples

package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/cybergodev/html"
)

// This example demonstrates proper error handling patterns.
// Learn how to handle errors gracefully and robustly.
func main() {
	fmt.Println("=== Error Handling Patterns ===\n ")

	// ============================================================
	// Pattern 1: Always check errors
	// ============================================================
	fmt.Println("Pattern 1: Basic Error Checking")
	fmt.Println("--------------------------------")

	htmlContent := `<html><body><h1>Hello</h1></body></html>`

	text, err := html.ExtractText([]byte(htmlContent))
	if err != nil {
		log.Fatalf("Failed to extract text: %v", err)
	}
	fmt.Printf("Extracted: %s\n\n", text)

	fmt.Println("✓ Always check errors from extraction functions")
	fmt.Println("✓ Use log.Fatal() for unrecoverable errors in main()")
	fmt.Println("✓ Return errors to callers in library code")

	// ============================================================
	// Pattern 2: Handle empty input errors
	// ============================================================
	fmt.Println("\nPattern 2: Handle Empty Input Errors")
	fmt.Println("-------------------------------------")

	_, err = html.ExtractText([]byte(""))
	if err != nil {
		fmt.Printf("Empty input handled: %v\n", err)
	}

	fmt.Println("\n✓ Check for empty input before processing")
	fmt.Println("✓ Handle invalid HTML gracefully")

	// ============================================================
	// Pattern 3: Validate input before processing
	// ============================================================
	fmt.Println("\n\nPattern 3: Input Validation")
	fmt.Println("----------------------------")

	emptyHTML := ""
	if len(emptyHTML) == 0 {
		fmt.Println("⚠ Empty HTML input - skipping extraction")
	}

	hugeHTML := strings.NewReader(strings.Repeat("<div>", 100_000_000))
	if hugeHTML.Size() > 50*1024*1024 { // 50MB
		fmt.Println("⚠ HTML too large - consider size limits")
	}

	malformedHTML := `<html><body><div>Unclosed tag`
	result, err := html.Extract([]byte(malformedHTML))
	if err != nil {
		fmt.Printf("⚠ Malformed HTML handled: %v\n", err)
	} else {
		fmt.Printf("✓ Extracted despite malformed HTML: %d chars\n", len(result.Text))
	}

	fmt.Println("\n✓ Validate inputs early")
	fmt.Println("✓ The library handles malformed HTML gracefully")

	// ============================================================
	// Pattern 4: Processor lifecycle errors
	// ============================================================
	fmt.Println("\n\nPattern 4: Processor Lifecycle")
	fmt.Println("------------------------------")

	processor, err := html.New()
	if err != nil {
		log.Fatalf("Failed to create processor: %v", err)
	}
	defer func() {
		if err := processor.Close(); err != nil {
			fmt.Printf("Warning: processor close failed: %v\n", err)
		}
	}()

	fmt.Println("✓ Check processor creation errors")
	fmt.Println("✓ Handle Close() errors in defer")
	fmt.Println("✓ Use defer to ensure cleanup")

	// ============================================================
	// Pattern 5: File handling errors
	// ============================================================
	fmt.Println("\n\nPattern 5: File Operation Errors")
	fmt.Println("----------------------------------")

	fmt.Println("Reading from file:")
	processor2, _ := html.New()
	defer processor2.Close()

	// Wrong path
	_, err = processor2.ExtractFromFile("nonexistent.html")
	if err != nil {
		fmt.Printf("✓ File not found handled: %v\n", err)
	}

	fmt.Println("\n✓ Always check file operation errors")
	fmt.Println("✓ Provide helpful error messages")
	fmt.Println("✓ Consider fallback options")

	// ============================================================
	// Pattern 6: Concurrent error handling
	// ============================================================
	fmt.Println("\n\nPattern 6: Concurrent Error Handling")
	fmt.Println("-------------------------------------")

	htmlDocs := [][]byte{
		[]byte("<html><body><p>Doc 1</p></body></html>"),
		[]byte(""), // Invalid
		[]byte("<html><body><p>Doc 3</p></body></html>"),
	}

	processor3, _ := html.New()
	defer processor3.Close()

	successCount := 0
	errorCount := 0

	for i, doc := range htmlDocs {
		_, err := processor3.Extract(doc)
		if err != nil {
			fmt.Printf("Doc %d: Error - %v\n", i+1, err)
			errorCount++
		} else {
			fmt.Printf("Doc %d: Success\n", i+1)
			successCount++
		}
	}

	fmt.Printf("\nResults: %d succeeded, %d failed\n", successCount, errorCount)
	fmt.Println("✓ Continue processing on individual failures")
	fmt.Println("✓ Track and report aggregate results")

	// ============================================================
	// Summary
	// ============================================================
	fmt.Println("\n=== Key Takeaways ===")
	fmt.Println("1. Always check and handle errors")
	fmt.Println("2. Use errors.As() for specific error types")
	fmt.Println("3. Validate inputs before processing")
	fmt.Println("4. Handle processor lifecycle properly")
	fmt.Println("5. Check file operation errors")
	fmt.Println("6. Continue on individual failures in batch processing")

}
