package html_test

// html_enhanced_test.go - Enhanced comprehensive tests for cybergodev/html
// This file contains:
// - Comprehensive error handling tests
// - Security tests
// - Edge case tests
// - Concurrency tests

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cybergodev/html"
)

// ============================================================================
// COMPREHENSIVE ERROR HANDLING TESTS
// ============================================================================

func TestErrorHandlingComprehensive(t *testing.T) {
	t.Parallel()

	t.Run("ErrInputTooLarge - oversized input rejected", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxInputSize = 10000 // Set lower limit for faster test
		p, err := html.New(config)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		largeHTML := strings.Repeat("<div>test content here</div>", 10000)

		_, err = p.ExtractWithDefaults([]byte(largeHTML))
		if err == nil {
			t.Errorf("Expected error for large input, got nil")
		}
	})

	t.Run("ErrInputTooLarge - exact limit boundary", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxInputSize = 5000
		p, err := html.New(config)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		validHTML := strings.Repeat("<div>a</div>", 100)
		_, err = p.ExtractWithDefaults([]byte(validHTML))
		if err != nil {
			t.Errorf("Should accept input at MaxInputSize boundary, got: %v", err)
		}

		oversizedHTML := strings.Repeat("<div>a</div>", 1000)
		_, err = p.ExtractWithDefaults([]byte(oversizedHTML))
		if err == nil {
			t.Errorf("Expected error for oversize input, got nil")
		}
	})

	t.Run("ErrInvalidHTML - malformed HTML handling", func(t *testing.T) {
		p, _ := html.New()
		defer p.Close()

		malformedCases := []struct {
			name string
			html string
		}{
			{"unclosed div", "<div>test"},
			{"triple angle brackets", "<<<>>>"},
			{"invalid tag", "<div test>test</div>"},
		}

		for _, tc := range malformedCases {
			t.Run(tc.name, func(t *testing.T) {
				// The library is tolerant and tries to extract anyway
				result, err := p.ExtractWithDefaults([]byte(tc.html))
				if err != nil {
					t.Errorf("Library should tolerate malformed HTML, got: %v", err)
				}
				if result == nil {
					t.Error("Result should not be nil even for malformed HTML")
				}
			})
		}

		t.Run("excessively deep nesting", func(t *testing.T) {
			config := html.DefaultConfig()
			config.MaxDepth = 100
			p, err := html.New(config)
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()

			deepHTML := strings.Repeat("<div>", 200) + "test" + strings.Repeat("</div>", 200)
			_, err = p.ExtractWithDefaults([]byte(deepHTML))
			// This should error due to max depth
			if err == nil {
				t.Error("Expected error for excessively deep nesting")
			}
		})
	})

	t.Run("ErrInvalidConfig - all validation failures", func(t *testing.T) {
		invalidConfigs := []struct {
			name   string
			config html.Config
		}{
			{"negative MaxInputSize", html.Config{MaxInputSize: -100, WorkerPoolSize: 4, MaxDepth: 100}},
			{"zero MaxInputSize", html.Config{MaxInputSize: 0, WorkerPoolSize: 4, MaxDepth: 100}},
			{"too large MaxInputSize", html.Config{MaxInputSize: 100 * 1024 * 1024, WorkerPoolSize: 4, MaxDepth: 100}},
			{"negative MaxCacheEntries", html.Config{MaxInputSize: 1000, MaxCacheEntries: -1, WorkerPoolSize: 4, MaxDepth: 100}},
			{"negative CacheTTL", html.Config{MaxInputSize: 1000, MaxCacheEntries: 100, CacheTTL: -1, WorkerPoolSize: 4, MaxDepth: 100}},
			{"zero WorkerPoolSize", html.Config{MaxInputSize: 1000, WorkerPoolSize: 0, MaxDepth: 100}},
			{"too large WorkerPoolSize", html.Config{MaxInputSize: 1000, WorkerPoolSize: 300, MaxDepth: 100}},
			{"zero MaxDepth", html.Config{MaxInputSize: 1000, WorkerPoolSize: 4, MaxDepth: 0}},
			{"too large MaxDepth", html.Config{MaxInputSize: 1000, WorkerPoolSize: 4, MaxDepth: 1000}},
			{"negative ProcessingTimeout", html.Config{MaxInputSize: 1000, WorkerPoolSize: 4, MaxDepth: 100, ProcessingTimeout: -1}},
		}

		for _, tc := range invalidConfigs {
			t.Run(tc.name, func(t *testing.T) {
				_, err := html.New(tc.config)
				if err == nil {
					t.Error("Expected error for invalid config, got nil")
				}
			})
		}
	})

	t.Run("ErrProcessingTimeout - timeout enforcement", func(t *testing.T) {
		config := html.DefaultConfig()
		config.ProcessingTimeout = 1 * time.Nanosecond
		p, err := html.New(config)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		largeHTML := strings.Repeat("<div>"+strings.Repeat("test ", 100)+"</div>", 1000)

		_, err = p.ExtractWithDefaults([]byte(largeHTML))
		if err != html.ErrProcessingTimeout {
			t.Errorf("Expected ErrProcessingTimeout, got: %v", err)
		}
	})

	t.Run("ErrFileNotFound - non-existent file", func(t *testing.T) {
		p, _ := html.New()
		defer p.Close()

		_, err := p.ExtractFromFile("non-existent-file-12345.html")
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
		if !errors.Is(err, html.ErrFileNotFound) {
			t.Errorf("Expected ErrFileNotFound, got: %v", err)
		}
	})

	t.Run("ErrInvalidFilePath - path traversal attempts", func(t *testing.T) {
		p, _ := html.New()
		defer p.Close()

		pathTraversalAttempts := []string{
			"../../../etc/passwd",
			"../../test.html",
			"../test.html",
			"./../../test.html",
			"test/../../../etc/passwd",
			"..\\windows\\system32",
		}

		for _, path := range pathTraversalAttempts {
			t.Run(path, func(t *testing.T) {
				_, err := p.ExtractFromFile(path)
				if err == nil {
					t.Error("Expected error for path traversal attempt, got nil")
				}
				if !errors.Is(err, html.ErrInvalidFilePath) {
					t.Errorf("Expected ErrInvalidFilePath for %q, got: %v", path, err)
				}
			})
		}
	})

	t.Run("ErrInvalidFilePath - empty and invalid paths", func(t *testing.T) {
		p, _ := html.New()
		defer p.Close()

		// Empty string should return ErrInvalidFilePath
		_, err := p.ExtractFromFile("")
		if !errors.Is(err, html.ErrInvalidFilePath) {
			t.Errorf("Expected ErrInvalidFilePath for empty path, got: %v", err)
		}

		// Whitespace paths may return OS-level errors (file not found)
		// This is acceptable behavior
		_, err = p.ExtractFromFile("   ")
		if err == nil {
			t.Error("Expected error for whitespace path")
		}
	})

	t.Run("ErrProcessorClosed - operations after close", func(t *testing.T) {
		p, _ := html.New()
		p.Close()

		operations := []func() error{
			func() error { _, err := p.ExtractWithDefaults([]byte("<html><body>test</body></html>")); return err },
			func() error { _, err := p.ExtractFromFile("test.html"); return err },
			func() error { _, err := p.ExtractBatch([][]byte{[]byte("<html><body>test</body></html>")}); return err },
			func() error { _, err := p.ExtractAllLinks([]byte("<html><body><a href='#'>link</a></body></html>")); return err },
		}

		for i, op := range operations {
			t.Run(fmt.Sprintf("operation_%d", i), func(t *testing.T) {
				err := op()
				if err != html.ErrProcessorClosed {
					t.Errorf("Expected ErrProcessorClosed, got: %v", err)
				}
			})
		}

		// Close() is idempotent - calling it again should not error
		if err := p.Close(); err != nil {
			t.Errorf("Close() should be idempotent, got: %v", err)
		}
	})
}

// ============================================================================
// SECURITY TESTS (Main Package Level)
// ============================================================================

func TestSecurityComprehensive(t *testing.T) {
	t.Parallel()

	t.Run("XSS prevention via script tags", func(t *testing.T) {
		xssAttempts := []string{
			`<script>alert('XSS')</script>`,
			`<SCRIPT>alert('XSS')</SCRIPT>`,
			`<script src="evil.js"></script>`,
			`<img src="x" onerror="alert('XSS')">`,
			`<svg onload="alert('XSS')">`,
			`<body onload="alert('XSS')">`,
			`<div onclick="alert('XSS')">click</div>`,
		}

		for _, xss := range xssAttempts {
			t.Run(fmt.Sprintf("xss_%d", len(xss)), func(t *testing.T) {
				p, _ := html.New()
				defer p.Close()

				htmlContent := "<html><body>" + xss + "</body></html>"
				result, err := p.ExtractWithDefaults([]byte(htmlContent))
				if err != nil {
					t.Fatalf("Extract() failed: %v", err)
				}

				if strings.Contains(strings.ToLower(result.Text), "script") {
					t.Errorf("Script content should be removed from: %s", result.Text)
				}
				if strings.Contains(strings.ToLower(result.Text), "alert") {
					t.Errorf("Alert content should be removed from: %s", result.Text)
				}
			})
		}
	})

	t.Run("Large input DoS prevention", func(t *testing.T) {
		p, _ := html.New()
		defer p.Close()

		hugeHTML := strings.Repeat("<div>"+strings.Repeat("test", 1000)+"</div>", 100000)

		_, err := p.ExtractWithDefaults([]byte(hugeHTML))
		if err == nil {
			t.Errorf("Expected error for huge input, got nil")
		}
	})

	t.Run("Data URL injection prevention", func(t *testing.T) {
		dangerousDataURLs := []string{
			`<img src="data:text/html,<script>alert('XSS')</script>">`,
			`<img src="data:text/javascript,alert('XSS')">`,
			`<a href="data:text/html,<script>alert('XSS')</script>">link</a>`,
		}

		for _, htmlContent := range dangerousDataURLs {
			t.Run(fmt.Sprintf("data_url_%d", len(htmlContent)), func(t *testing.T) {
				p, _ := html.New()
				defer p.Close()

				fullHTML := "<html><body>" + htmlContent + "</body></html>"
				result, err := p.ExtractWithDefaults([]byte(fullHTML))
				if err != nil {
					t.Fatalf("Extract() failed: %v", err)
				}

				if strings.Contains(result.Text, "data:text/html") ||
					strings.Contains(result.Text, "data:text/javascript") {
					t.Error("Dangerous data URLs should be removed")
				}
			})
		}
	})
}

// ============================================================================
// EDGE CASE TESTS
// ============================================================================

func TestEdgeCasesComprehensive(t *testing.T) {
	t.Parallel()

	t.Run("Empty and whitespace-only HTML", func(t *testing.T) {
		p, _ := html.New()
		defer p.Close()

		emptyCases := []string{
			"",
			"   ",
			"\t\n",
			"<!DOCTYPE html>",
			"<!-- comment only -->",
			"<html></html>",
			"<html><body></body></html>",
		}

		for _, htmlContent := range emptyCases {
			t.Run(fmt.Sprintf("empty_%d", len(htmlContent)), func(t *testing.T) {
				result, err := p.ExtractWithDefaults([]byte(htmlContent))
				if err != nil {
					t.Errorf("Empty HTML should return result, not error: %v", err)
				}
				if result == nil {
					t.Error("Result should not be nil for empty HTML")
				}
			})
		}
	})

	t.Run("Unicode content handling", func(t *testing.T) {
		p, _ := html.New()
		defer p.Close()

		unicodeHTML := `<html><body>
			<h1>你好世界</h1>
			<p>Привет мир</p>
			<p>مرحبا بالعالم</p>
			<p>🎉🎊🎁 Emoji test</p>
			<p>עברית</p>
			<p>العربية</p>
		</body></html>`

		result, err := p.ExtractWithDefaults([]byte(unicodeHTML))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "你好世界") {
			t.Error("Should preserve Chinese characters")
		}
		if !strings.Contains(result.Text, "Привет мир") {
			t.Error("Should preserve Cyrillic characters")
		}
		if !strings.Contains(result.Text, "مرحبا بالعالم") {
			t.Error("Should preserve Arabic characters")
		}
		if !strings.Contains(result.Text, "🎉") {
			t.Error("Should preserve emoji")
		}
	})

	t.Run("Excessive whitespace handling", func(t *testing.T) {
		p, _ := html.New()
		defer p.Close()

		whitespaceHTML := `<html><body>
			<p>Text    with     many     spaces</p>
			<p>Text

with

newlines</p>
			<p>Text	with	tabs</p>
		</body></html>`

		result, err := p.ExtractWithDefaults([]byte(whitespaceHTML))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if strings.Contains(result.Text, "    ") {
			t.Error("Should collapse multiple spaces")
		}
		if strings.Contains(result.Text, "\t") {
			t.Error("Should replace tabs with spaces")
		}
	})

	t.Run("Mixed content and scripts", func(t *testing.T) {
		p, _ := html.New()
		defer p.Close()

		mixedHTML := `<html><body>
			<script>var x = "dangerous";</script>
			<style>.danger { color: red; }</style>
			<p>Valid content</p>
			<noscript>JavaScript required</noscript>
		</body></html>`

		result, err := p.ExtractWithDefaults([]byte(mixedHTML))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if strings.Contains(strings.ToLower(result.Text), "var x") {
			t.Error("Script content should be removed")
		}
		if !strings.Contains(result.Text, "Valid content") {
			t.Error("Valid content should be extracted")
		}
	})

	t.Run("Entity decoding edge cases", func(t *testing.T) {
		p, _ := html.New()
		defer p.Close()

		entityHTML := `<html><body>
			<p>&amp;&lt;&gt;&quot;&apos;</p>
			<p>&nbsp;&nbsp;&copy;&reg;&trade;</p>
			<p>&mdash;&ndash;&hellip;</p>
			<p>&euro;&pound;&yen;</p>
			<p>&#65;&#x41;</p>
		</body></html>`

		result, err := p.ExtractWithDefaults([]byte(entityHTML))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "&<>\"'") {
			t.Error("Basic XML entities should be decoded")
		}
		if !strings.Contains(result.Text, "©") {
			t.Error("Copyright entity should be decoded")
		}
		if !strings.Contains(result.Text, "—") {
			t.Error("Em dash entity should be decoded")
		}
	})
}

// ============================================================================
// CONCURRENCY TESTS
// ============================================================================

func TestConcurrencyComprehensive(t *testing.T) {
	t.Run("Multiple goroutines using same processor", func(t *testing.T) {
		p, _ := html.New()
		defer p.Close()

		htmlContent := []byte(`<html><body><h1>Concurrent Test</h1><p>Content</p></body></html>`)

		const numGoroutines = 50
		const iterations = 10

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*iterations)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					_, err := p.ExtractWithDefaults(htmlContent)
					if err != nil {
						errors <- fmt.Errorf("goroutine %d iteration %d: %w", id, j, err)
						return
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent access error: %v", err)
		}
	})

	t.Run("Concurrent cache access", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxCacheEntries = 100
		p, err := html.New(config)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		htmlContent := []byte(`<html><body><p>Cache test content</p></body></html>`)

		const numGoroutines = 20
		const iterations = 50

		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					p.ExtractWithDefaults(htmlContent)
				}
			}()
		}

		wg.Wait()

		stats := p.GetStatistics()
		if stats.TotalProcessed == 0 {
			t.Error("No operations processed")
		}
		if stats.CacheHits == 0 && stats.CacheMisses == 0 {
			t.Error("Cache not working")
		}
	})

	t.Run("Batch processing with concurrent operations", func(t *testing.T) {
		p, _ := html.New()
		defer p.Close()

		htmlContents := make([][]byte, 50)
		for i := range htmlContents {
			htmlContents[i] = []byte(fmt.Sprintf("<html><body><p>Content %d</p></body></html>", i))
		}

		results, err := p.ExtractBatch(htmlContents)
		if err != nil {
			t.Fatalf("ExtractBatch() failed: %v", err)
		}

		if len(results) != len(htmlContents) {
			t.Errorf("Got %d results, want %d", len(results), len(htmlContents))
		}

		for i, result := range results {
			if result == nil {
				t.Errorf("Result %d is nil", i)
			}
		}
	})

	t.Run("Close during active processing", func(t *testing.T) {
		p, _ := html.New()

		htmlContent := []byte(`<html><body><p>Test</p></body></html>`)

		const numGoroutines = 10
		var wg sync.WaitGroup
		started := make(chan struct{}, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				started <- struct{}{}
				time.Sleep(10 * time.Millisecond)
				p.ExtractWithDefaults(htmlContent)
			}()
		}

		for i := 0; i < numGoroutines; i++ {
			<-started
		}

		p.Close()

		wg.Wait()
	})

	t.Run("Statistics consistency under concurrency", func(t *testing.T) {
		p, _ := html.New()
		defer p.Close()

		htmlContent := []byte(`<html><body><p>Stats test</p></body></html>`)

		const numGoroutines = 20
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					p.ExtractWithDefaults(htmlContent)
				}
			}()
		}

		wg.Wait()

		stats := p.GetStatistics()

		if stats.TotalProcessed != numGoroutines*10 {
			t.Errorf("TotalProcessed = %d, want %d", stats.TotalProcessed, numGoroutines*10)
		}
		if stats.CacheHits < 0 {
			t.Error("CacheHits should be non-negative")
		}
		if stats.CacheMisses < 0 {
			t.Error("CacheMisses should be non-negative")
		}
	})
}
