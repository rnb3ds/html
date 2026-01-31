//go:build examples

package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cybergodev/html"
)

// ErrorHandling demonstrates common error handling patterns.
func main() {
	fmt.Println("=== Error Handling Patterns ===\n ")

	// Example 1: Handle invalid HTML
	fmt.Println("1. Handle malformed HTML:")
	malformedHTML := `<div><p>Unclosed tag<div><p>More content</p></div>`

	processor := html.NewWithDefaults()
	defer processor.Close()

	result, err := processor.ExtractWithDefaults(malformedHTML)
	if err != nil {
		log.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Success: Extracted %d words\n", result.WordCount)
		fmt.Printf("   Library handles malformed HTML gracefully\n\n")
	}

	// Example 2: Handle empty input
	fmt.Println("2. Handle empty input:")
	emptyResult, err := processor.ExtractWithDefaults("")
	if err != nil {
		log.Printf("   Error: %v\n", err)
	}
	if emptyResult.Text == "" {
		fmt.Printf("   Empty text returned (expected)\n\n")
	}

	// Example 3: Handle oversized input
	fmt.Println("3. Handle oversized input:")
	largeHTML := strings.Repeat("<p>Large content</p>", 1000000) // ~20MB

	_, err = processor.ExtractWithDefaults(largeHTML)
	if err != nil {
		fmt.Printf("   Caught error: %v\n", err)
		fmt.Printf("   Use Config.MaxInputSize to adjust limit\n\n")
	}

	// Example 4: Check specific error types
	fmt.Println("4. Check specific error types:")
	_, err = processor.ExtractWithDefaults("<html></html>")
	if errors.Is(err, html.ErrInputTooLarge) {
		fmt.Printf("   Input too large\n")
	} else if errors.Is(err, html.ErrInvalidHTML) {
		fmt.Printf("   Invalid HTML\n")
	} else if errors.Is(err, html.ErrProcessorClosed) {
		fmt.Printf("   Processor closed\n")
	} else if err == nil {
		fmt.Printf("   No error (empty but valid HTML)\n\n")
	}

	// Example 5: Handle closed processor
	fmt.Println("5. Handle closed processor:")
	p2 := html.NewWithDefaults()
	p2.Close()

	_, err = p2.ExtractWithDefaults("<p>Test</p>")
	if errors.Is(err, html.ErrProcessorClosed) {
		fmt.Printf("   Processor is closed: %v\n\n", err)
	}

	// Example 6: Retry pattern with timeout
	fmt.Println("6. Retry with timeout (short timeout demo):")
	config := html.DefaultConfig()
	config.ProcessingTimeout = 1 * time.Microsecond // Very short for demo
	p3, err := html.New(config)
	if err != nil {
		log.Printf("   Failed to create processor: %v\n\n", err)
	} else {
		defer p3.Close()

		_, err = p3.ExtractWithDefaults(strings.Repeat("<p>Content</p>", 100000))
		if err != nil {
			fmt.Printf("   Timeout error: %v\n", err)
			fmt.Printf("   Increase ProcessingTimeout if needed\n\n")
		}
	}

	// Example 7: Safe extraction wrapper
	fmt.Println("7. Safe extraction wrapper:")
	htmls := []string{
		`<article><h1>Valid</h1><p>Content</p></article>`,
		``,
		`<div>Malformed`,
		`<article><h1>Another Valid</h1><p>Content</p></article>`,
	}

	for i, h := range htmls {
		result, err := safeExtract(processor, h)
		if err != nil {
			fmt.Printf("   [%d] Error: %v\n", i+1, err)
		} else {
			fmt.Printf("   [%d] OK: %d words\n", i+1, result.WordCount)
		}
	}

}

// safeExtract wraps extraction with error handling
func safeExtract(p *html.Processor, htmlContent string) (*html.Result, error) {
	if strings.TrimSpace(htmlContent) == "" {
		return nil, fmt.Errorf("empty HTML content")
	}

	result, err := p.ExtractWithDefaults(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	if result.WordCount == 0 {
		return result, fmt.Errorf("no content extracted")
	}

	return result, nil
}
