//go:build examples

package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/cybergodev/html"
)

// This example demonstrates error handling patterns.
func main() {
	fmt.Println("=== Error Handling ===\n")

	processor, _ := html.New()
	defer processor.Close()

	// ============================================================
	// 1. Basic Error Checking
	// ============================================================
	fmt.Println("1. Basic Error Checking")
	fmt.Println("-----------------------")

	htmlContent := `<html><body><h1>Hello</h1></body></html>`
	text, err := html.ExtractText([]byte(htmlContent))
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	fmt.Printf("Extracted: %s\n\n", text)
	fmt.Println("Always check errors from extraction functions")

	// ============================================================
	// 2. Sentinel Errors
	// ============================================================
	fmt.Println("\n2. Sentinel Errors")
	fmt.Println("-----------------")

	// Empty input
	_, err = processor.Extract([]byte(""))
	if err != nil {
		fmt.Printf("Empty input: %v\n", err)
	}

	// Input too large (demonstration)
	largeInput := strings.Repeat("<div>", 1000)
	if len(largeInput) > 50*1024*1024 {
		fmt.Println("Input validation prevents memory issues")
	}

	// ============================================================
	// 3. Custom Error Types
	// ============================================================
	fmt.Println("\n3. Custom Error Types")
	fmt.Println("--------------------")

	// Demonstrate InputError fields
	var inputErr html.InputError
	if inputErr.InputErr != nil {
		fmt.Printf("InputError: size=%d, maxSize=%d\n", inputErr.Size, inputErr.MaxSize)
	}

	// Demonstrate ConfigError fields
	var configErr html.ConfigError
	if configErr.Message != "" {
		fmt.Printf("ConfigError: field=%s, message=%s\n", configErr.Field, configErr.Message)
	}

	// Demonstrate FileError fields
	var fileErr html.FileError
	if fileErr.FileErr != nil {
		fmt.Printf("FileError: path=%s, error=%v\n", fileErr.Path, fileErr.FileErr)
	}

	fmt.Println("Use errors.Is() for sentinel errors")
	fmt.Println("Use errors.As() for custom error types")
	fmt.Println("Check error fields for specific error info")

	// ============================================================
	// 4. Processor Lifecycle
	// ============================================================
	fmt.Println("\n4. Processor Lifecycle")
	fmt.Println("---------------------")

	p, err := html.New()
	if err != nil {
		log.Fatalf("Failed to create processor: %v", err)
	}
	defer p.Close()

	fmt.Println("Always use defer for cleanup")
	fmt.Println("Check Close() error return")

	// ============================================================
	// 5. Batch Processing
	// ============================================================
	fmt.Println("\n5. Batch Processing")
	fmt.Println("-----------------------------")

	docs := [][]byte{
		[]byte("<html><body><p>Doc 1</p></body></html>"),
		[]byte(""), // Empty - causes error
		[]byte("<html><body><p>Doc 3</p></body></html>"),
	}

	results, err := processor.ExtractBatch(docs)
	if err != nil {
		fmt.Printf("Batch error: %v\n", err)
	} else {
		fmt.Printf("Batch: %d succeeded, %d results\n", len(results), len(results))
		for i, r := range results {
			if r != nil {
				fmt.Printf("  Doc %d: %s (%d words)\n", i+1, r.Title, r.WordCount)
			}
		}
	}

	fmt.Println("Continue processing on individual failures")
	fmt.Println("Track and report aggregate results")

	// ============================================================
	// Summary
	// ============================================================
	fmt.Println("\n=== Error Handling Summary ===")
	fmt.Println()
	fmt.Println("1. Always check errors from extraction functions")
	fmt.Println("2. Use errors.Is() for sentinel errors (ErrInputTooLarge, etc.)")
	fmt.Println("3. Use errors.As() for custom error types (InputError, ConfigError, FileError)")
	fmt.Println("4. Handle processor lifecycle with defer")
	fmt.Println("5. Check batch results for individual failures")
}
