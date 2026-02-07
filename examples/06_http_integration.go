//go:build examples

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/cybergodev/html"
)

// This example demonstrates real-world HTTP integration patterns.
// Learn how to fetch web pages and extract content efficiently.
func main() {
	fmt.Println("=== HTTP Integration Examples ===\n ")

	// ============================================================
	// Example 1: Fetch and extract from a single URL
	// ============================================================
	fmt.Println("Example 1: Fetch a single webpage")
	fmt.Println("--------------------------------")

	url := "https://example.com"

	content, err := fetchURL(url)
	if err != nil {
		log.Printf("Error: %v\n", err)
		fmt.Println("\nNote: This example requires internet access")
		fmt.Println("For testing, you can use a local file or mock HTML")
		return
	}

	result, err := html.Extract(content)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("URL: %s\n", url)
	fmt.Printf("Title: %s\n", result.Title)
	fmt.Printf("Content length: %d characters\n", len(result.Text))
	fmt.Printf("Word count: %d\n", result.WordCount)
	fmt.Printf("Processing time: %v\n", result.ProcessingTime)

	// ============================================================
	// Example 2: Process multiple URLs concurrently
	// ============================================================
	fmt.Println("\n\nExample 2: Process multiple URLs concurrently")
	fmt.Println("--------------------------------------------")

	urls := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page3",
	}

	processor, _ := html.New()
	defer processor.Close()

	start := time.Now()
	results := processURLsConcurrently(processor, urls)
	duration := time.Since(start)

	fmt.Printf("Processed %d URLs in %v\n", len(results), duration)
	fmt.Printf("Average: %v per URL\n", duration/time.Duration(len(results)))

	for _, r := range results {
		if r.Error != nil {
			fmt.Printf("  • %s: Error - %v\n", r.URL, r.Error)
		} else {
			fmt.Printf("  • %s: %s (%d words)\n", r.URL, r.Result.Title, r.Result.WordCount)
		}
	}

	// ============================================================
	// Example 3: Reusable HTTP client with timeout
	// ============================================================
	fmt.Println("\n\nExample 3: Configure HTTP client")
	fmt.Println("--------------------------------")

	httpClient := createHTTPClient(10*time.Second, 5)

	fmt.Println("HTTP Client Configuration:")
	fmt.Println("  Timeout: 10 seconds")
	fmt.Println("  Max Idle Conns: 5")
	fmt.Println("  Max Idle Conns Per Host: 5")
	fmt.Println("  Idle Conn Timeout: 90 seconds")

	// Use the custom client
	resp, err := httpClient.Get("https://example.com")
	if err != nil {
		log.Printf("Request failed: %v\n", err)
	} else {
		defer resp.Body.Close()
		fmt.Printf("\nResponse Status: %s\n", resp.Status)
		fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
		fmt.Printf("Content-Length: %s\n", resp.Header.Get("Content-Length"))

		if resp.StatusCode == 200 {
			body, _ := io.ReadAll(resp.Body)
			result, err := html.Extract(body)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("\nExtracted: %s (%d words)\n", result.Title, result.WordCount)
		}
	}

	// ============================================================
	// Example 4: Error handling patterns for HTTP
	// ============================================================
	fmt.Println("\n\nExample 4: Robust error handling")
	fmt.Println("--------------------------------")

	urls2 := []string{
		"https://example.com/good",
		"https://example.com/timeout",
		"https://example.com/notfound",
		"https://invalid-url-that-will-fail",
	}

	processor2, _ := html.New()
	defer processor2.Close()

	for _, url := range urls2 {
		content, err := fetchURLWithTimeout(url, 5*time.Second)

		if err != nil {
			fmt.Printf("  ✗ %s\n", url)
			fmt.Printf("    Error: %v\n\n", err)
			continue
		}

		result, err := processor2.Extract(content)
		if err != nil {
			fmt.Printf("  ✗ %s\n", url)
			fmt.Printf("    Extraction error: %v\n\n", err)
			continue
		}

		fmt.Printf("  ✓ %s\n", url)
		fmt.Printf("    Title: %s\n", result.Title)
		fmt.Printf("    Words: %d\n\n", result.WordCount)
	}

	// ============================================================
	// Example 5: Best practices for production use
	// ============================================================
	fmt.Println("Example 5: Production best practices")
	fmt.Println("------------------------------------")

	tips := []struct {
		category string
		tips     []string
	}{
		{
			category: "Performance",
			tips: []string{
				"Reuse processor across multiple requests",
				"Use ExtractBatch() for processing 10+ pages",
				"Set WorkerPoolSize to number of CPU cores",
				"Enable caching for repeated content",
			},
		},
		{
			category: "Reliability",
			tips: []string{
				"Always set timeouts on HTTP requests",
				"Implement retry logic for transient failures",
				"Limit concurrent requests to avoid rate limiting",
				"Validate responses before processing",
			},
		},
		{
			category: "Security",
			tips: []string{
				"Keep sanitization enabled (default)",
				"Validate and sanitize URLs before fetching",
				"Limit MaxInputSize to prevent memory issues",
				"Use HTTPS when possible",
			},
		},
		{
			category: "Monitoring",
			tips: []string{
				"Log extraction errors for debugging",
				"Track processing time metrics",
				"Monitor cache hit/miss rates",
				"Set up alerts for failure rates",
			},
		},
	}

	for _, cat := range tips {
		fmt.Printf("\n%s:\n", cat.category)
		for _, tip := range cat.tips {
			fmt.Printf("  • %s\n", tip)
		}
	}

	fmt.Println("\n✓ You now know how to integrate with HTTP!")
}

// fetchURL fetches content from a URL
func fetchURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// fetchURLWithTimeout fetches with a custom timeout
func fetchURLWithTimeout(url string, timeout time.Duration) ([]byte, error) {
	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// processURLsConcurrently processes multiple URLs concurrently
func processURLsConcurrently(processor *html.Processor, urls []string) []URLResult {
	var wg sync.WaitGroup
	results := make([]URLResult, len(urls))
	var mu sync.Mutex

	for i, url := range urls {
		wg.Add(1)
		go func(idx int, u string) {
			defer wg.Done()

			content, err := fetchURL(u)
			if err != nil {
				mu.Lock()
				results[idx] = URLResult{URL: u, Error: err}
				mu.Unlock()
				return
			}

			result, err := processor.Extract(content)
			if err != nil {
				mu.Lock()
				results[idx] = URLResult{URL: u, Error: err}
				mu.Unlock()
				return
			}

			mu.Lock()
			results[idx] = URLResult{URL: u, Result: result}
			mu.Unlock()
		}(i, url)
	}

	wg.Wait()
	return results
}

// createHTTPClient creates an optimized HTTP client
func createHTTPClient(timeout time.Duration, maxIdle int) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        maxIdle,
			MaxIdleConnsPerHost: maxIdle,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
		},
	}
}

// URLResult represents the result of processing a URL
type URLResult struct {
	URL    string
	Result *html.Result
	Error  error
}
