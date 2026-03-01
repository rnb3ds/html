//go:build examples

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/cybergodev/html"
)

// This example demonstrates real-world HTTP integration patterns.
// Learn how to fetch web pages and extract content efficiently.
func main() {
	fmt.Println("=== HTTP Integration Examples ===\n ")

	// Create a mock server for demonstration
	server := createMockServer()
	defer server.Close()

	// ============================================================
	// Example 1: Fetch and extract from a single URL
	// ============================================================
	fmt.Println("Example 1: Fetch a single webpage")
	fmt.Println("--------------------------------")

	url := server.URL + "/article"

	content, err := fetchURL(url)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
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
		server.URL + "/article",
		server.URL + "/blog",
		server.URL + "/docs",
	}

	processor, _ := html.New()
	defer processor.Close()

	start := time.Now()
	results := processURLsConcurrently(processor, urls)
	duration := time.Since(start)

	fmt.Printf("Processed %d URLs in %v\n", len(results), duration)

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

	httpClient := createHTTPClient(10 * time.Second)

	fmt.Println("HTTP Client Configuration:")
	fmt.Println("  Timeout: 10 seconds")
	fmt.Println("  Max Idle Conns: 10")
	fmt.Println("  Idle Conn Timeout: 90 seconds")

	resp, err := httpClient.Get(server.URL + "/article")
	if err != nil {
		log.Printf("Request failed: %v\n", err)
	} else {
		defer resp.Body.Close()
		fmt.Printf("\nResponse Status: %s\n", resp.Status)
		fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))

		if resp.StatusCode == 200 {
			body, _ := io.ReadAll(resp.Body)
			result, err := html.Extract(body)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Extracted: %s (%d words)\n", result.Title, result.WordCount)
		}
	}

	// ============================================================
	// Example 4: Batch processing with context
	// ============================================================
	fmt.Println("\n\nExample 4: Batch processing with context")
	fmt.Println("----------------------------------------")

	// Fetch multiple pages
	pages := make([][]byte, 3)
	for i, path := range []string{"/article", "/blog", "/docs"} {
		content, err := fetchURL(server.URL + path)
		if err != nil {
			log.Printf("Failed to fetch %s: %v\n", path, err)
			continue
		}
		pages[i] = content
	}

	// Process with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	batchResult := processor.ExtractBatchWithContext(ctx, pages)

	fmt.Printf("Batch Results:\n")
	fmt.Printf("  Success:   %d\n", batchResult.Success)
	fmt.Printf("  Failed:    %d\n", batchResult.Failed)
	fmt.Printf("  Cancelled: %d\n", batchResult.Cancelled)

	// ============================================================
	// Example 5: Best practices for production use
	// ============================================================
	fmt.Println("\n\nExample 5: Production best practices")
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
func createHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
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

// createMockServer creates a test server with sample HTML content
func createMockServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/article", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html>
				<head><title>Go Article</title></head>
				<body>
					<article>
						<h1>Understanding Go Concurrency</h1>
						<p>Go makes concurrent programming easy with goroutines and channels.</p>
						<p>This article explores the fundamentals of concurrent programming.</p>
					</article>
				</body>
			</html>
		`))
	})

	mux.HandleFunc("/blog", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html>
				<head><title>Blog Post</title></head>
				<body>
					<article>
						<h1>Web Scraping Best Practices</h1>
						<p>Learn how to scrape web content responsibly and efficiently.</p>
					</article>
				</body>
			</html>
		`))
	})

	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html>
				<head><title>Documentation</title></head>
				<body>
					<article>
						<h1>API Reference</h1>
						<p>This document describes the API endpoints and their usage.</p>
						<p>Refer to the examples for practical implementations.</p>
					</article>
				</body>
			</html>
		`))
	})

	return httptest.NewServer(mux)
}
