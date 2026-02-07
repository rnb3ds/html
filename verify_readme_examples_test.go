package html

import (
	"testing"
	"time"
)

// TestREADMEExamples verifies that all code examples in README.md are accurate and runnable
func TestREADMEExamples(t *testing.T) {
	t.Run("Package-Level Functions Example", func(t *testing.T) {
		htmlContent := []byte(`
			<html>
				<nav>Navigation</nav>
				<article><h1>Hello World</h1><p>Content here...</p></article>
				<footer>Footer</footer>
			</html>
		`)

		// Test ExtractText
		text, err := ExtractText(htmlContent)
		if err != nil {
			t.Fatalf("ExtractText failed: %v", err)
		}
		if text == "" {
			t.Error("ExtractText returned empty string")
		}

		// Test Extract
		result, err := Extract(htmlContent)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}
		if result.Title == "" && result.Text == "" {
			t.Error("Extract returned empty result")
		}

		// Test ExtractAllLinks
		links, err := ExtractAllLinks(htmlContent)
		if err != nil {
			t.Fatalf("ExtractAllLinks failed: %v", err)
		}
		if links == nil {
			t.Error("ExtractAllLinks returned nil")
		}

		// Test ExtractToMarkdown
		markdown, err := ExtractToMarkdown(htmlContent)
		if err != nil {
			t.Fatalf("ExtractToMarkdown failed: %v", err)
		}
		if markdown == "" {
			t.Error("ExtractToMarkdown returned empty string")
		}

		// Test ExtractToJSON
		jsonData, err := ExtractToJSON(htmlContent)
		if err != nil {
			t.Fatalf("ExtractToJSON failed: %v", err)
		}
		if len(jsonData) == 0 {
			t.Error("ExtractToJSON returned empty data")
		}
	})

	t.Run("Processor Usage Example", func(t *testing.T) {
		htmlContent := []byte(`<html><body><h1>Test</h1><p>Content</p></body></html>`)

		processor, _ := New()
		defer processor.Close()

		// Test ExtractWithDefaults
		result, err := processor.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("ExtractWithDefaults failed: %v", err)
		}
		if result.Text == "" {
			t.Error("ExtractWithDefaults returned empty result")
		}

		// Test GetStatistics
		stats := processor.GetStatistics()
		if stats.TotalProcessed == 0 {
			t.Error("GetStatistics returned zero processed count")
		}

		// Test ResetStatistics
		processor.ResetStatistics()
		statsAfterReset := processor.GetStatistics()
		if statsAfterReset.TotalProcessed != 0 {
			t.Error("ResetStatistics did not reset TotalProcessed")
		}

		// Test ClearCache
		processor.ClearCache()
	})

	t.Run("Custom Configuration Example", func(t *testing.T) {
		htmlContent := []byte(`<html><body><h1>Test</h1></body></html>`)

		config := ExtractConfig{
			ExtractArticle:    true,
			PreserveImages:    true,
			PreserveLinks:     true,
			PreserveVideos:    false,
			PreserveAudios:    false,
			InlineImageFormat: "none",
			TableFormat:       "markdown",
		}

		result, err := Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract with config failed: %v", err)
		}
		if result == nil {
			t.Error("Extract returned nil result")
		}
	})

	t.Run("Processor Configuration Example", func(t *testing.T) {
		config := Config{
			MaxInputSize:       10 * 1024 * 1024,
			ProcessingTimeout:  30 * time.Second,
			MaxCacheEntries:    500,
			CacheTTL:           30 * time.Minute,
			WorkerPoolSize:     8,
			EnableSanitization: true,
			MaxDepth:           50,
		}

		processor, err := New(config)
		if err != nil {
			t.Fatalf("New with config failed: %v", err)
		}
		defer processor.Close()

		if processor == nil {
			t.Error("New returned nil processor")
		}
	})

	t.Run("Link Extraction Example", func(t *testing.T) {
		htmlContent := []byte(`
			<html><body>
				<a href="https://example.com">Link</a>
				<img src="image.jpg" alt="Image">
			</body></html>
		`)

		links, err := ExtractAllLinks(htmlContent)
		if err != nil {
			t.Fatalf("ExtractAllLinks failed: %v", err)
		}

		// Test GroupLinksByType
		byType := GroupLinksByType(links)
		if byType == nil {
			t.Error("GroupLinksByType returned nil")
		}

		// Test processor link extraction
		processor, _ := New()
		defer processor.Close()

		linkConfig := DefaultLinkExtractionConfig()
		links2, err := processor.ExtractAllLinks(htmlContent, linkConfig)
		if err != nil {
			t.Fatalf("Processor.ExtractAllLinks failed: %v", err)
		}
		if links2 == nil {
			t.Error("Processor.ExtractAllLinks returned nil")
		}
	})

	t.Run("Default Configurations", func(t *testing.T) {
		// Test DefaultConfig
		config := DefaultConfig()
		if config.MaxInputSize == 0 {
			t.Error("DefaultConfig returned zero MaxInputSize")
		}
		if config.MaxCacheEntries == 0 {
			t.Error("DefaultConfig returned zero MaxCacheEntries")
		}

		// Test DefaultExtractConfig
		extractConfig := DefaultExtractConfig()
		if !extractConfig.ExtractArticle {
			t.Error("DefaultExtractConfig ExtractArticle is false")
		}
		if !extractConfig.PreserveImages {
			t.Error("DefaultExtractConfig PreserveImages is false")
		}

		// Test DefaultLinkExtractionConfig
		linkConfig := DefaultLinkExtractionConfig()
		if !linkConfig.IncludeImages {
			t.Error("DefaultLinkExtractionConfig IncludeImages is false")
		}
	})

	t.Run("Batch Processing Example", func(t *testing.T) {
		htmlContents := [][]byte{
			[]byte(`<html><body><h1>Test 1</h1></body></html>`),
			[]byte(`<html><body><h1>Test 2</h1></body></html>`),
			[]byte(`<html><body><h1>Test 3</h1></body></html>`),
		}

		processor, _ := New()
		defer processor.Close()

		results, err := processor.ExtractBatch(htmlContents, DefaultExtractConfig())
		if err != nil {
			t.Fatalf("ExtractBatch failed: %v", err)
		}
		if len(results) != len(htmlContents) {
			t.Errorf("ExtractBatch returned %d results, want %d", len(results), len(htmlContents))
		}
	})

	t.Run("Thread Safety Example", func(t *testing.T) {
		htmlContent := []byte(`<html><body><h1>Concurrent Test</h1></body></html>`)
		processor, _ := New()
		defer processor.Close()

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_, _ = processor.ExtractWithDefaults(htmlContent)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			select {
			case <-done:
				// OK
			case <-time.After(5 * time.Second):
				t.Fatal("Concurrent test timeout")
			}
		}
	})
}
