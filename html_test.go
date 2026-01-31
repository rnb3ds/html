package html_test

// html_test.go - Comprehensive test suite for the html package
// This file consolidates and improves all main package tests to provide:
// - Better coverage of public APIs
// - Elimination of duplicate tests
// - Improved maintainability
// - Effective regression testing

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cybergodev/html"
	stdhtml "golang.org/x/net/html"
)

// ============================================================================
// PROCESSOR LIFECYCLE AND CONFIGURATION TESTS
// ============================================================================

func TestProcessorLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("New with valid config", func(t *testing.T) {
		config := html.DefaultConfig()
		p, err := html.New(config)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()
		if p == nil {
			t.Fatal("New() returned nil processor")
		}
	})

	t.Run("NewWithDefaults", func(t *testing.T) {
		p := html.NewWithDefaults()
		if p == nil {
			t.Fatal("NewWithDefaults() returned nil")
		}
		defer p.Close()
	})

	t.Run("Close idempotent", func(t *testing.T) {
		p := html.NewWithDefaults()
		if err := p.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
		if err := p.Close(); err != nil {
			t.Fatalf("Close() should be idempotent: %v", err)
		}
	})

	t.Run("Extract after close fails", func(t *testing.T) {
		p := html.NewWithDefaults()
		p.Close()
		_, err := p.ExtractWithDefaults("<html><body>Test</body></html>")
		if err != html.ErrProcessorClosed {
			t.Errorf("Extract() should fail with ErrProcessorClosed, got: %v", err)
		}
	})
}

func TestConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("DefaultConfig values", func(t *testing.T) {
		config := html.DefaultConfig()
		if config.MaxInputSize <= 0 {
			t.Error("MaxInputSize should be positive")
		}
		if config.MaxCacheEntries < 0 {
			t.Error("MaxCacheEntries should be non-negative")
		}
		if config.CacheTTL < 0 {
			t.Error("CacheTTL should be non-negative")
		}
		if config.WorkerPoolSize <= 0 {
			t.Error("WorkerPoolSize should be positive")
		}
		if !config.EnableSanitization {
			t.Error("EnableSanitization should be true by default")
		}
		if config.MaxDepth <= 0 {
			t.Error("MaxDepth should be positive")
		}
		if config.ProcessingTimeout != 30*time.Second {
			t.Errorf("ProcessingTimeout = %v, want %v", config.ProcessingTimeout, 30*time.Second)
		}
	})

	t.Run("DefaultExtractConfig values", func(t *testing.T) {
		config := html.DefaultExtractConfig()
		if !config.ExtractArticle {
			t.Error("ExtractArticle should be true")
		}
		if !config.PreserveImages {
			t.Error("PreserveImages should be true")
		}
		if !config.PreserveLinks {
			t.Error("PreserveLinks should be true")
		}
		if !config.PreserveVideos {
			t.Error("PreserveVideos should be true")
		}
		if !config.PreserveAudios {
			t.Error("PreserveAudios should be true")
		}
		if config.InlineImageFormat != "none" {
			t.Errorf("InlineImageFormat = %q, want 'none'", config.InlineImageFormat)
		}
		if config.TableFormat != "markdown" {
			t.Errorf("TableFormat = %q, want 'markdown'", config.TableFormat)
		}
	})

	t.Run("Invalid configs rejected", func(t *testing.T) {
		tests := []struct {
			name   string
			modify func(*html.Config)
		}{
			{
				name:   "negative MaxInputSize",
				modify: func(c *html.Config) { c.MaxInputSize = -1 },
			},
			{
				name:   "zero WorkerPoolSize",
				modify: func(c *html.Config) { c.WorkerPoolSize = 0 },
			},
			{
				name:   "zero MaxDepth",
				modify: func(c *html.Config) { c.MaxDepth = 0 },
			},
			{
				name:   "negative CacheTTL",
				modify: func(c *html.Config) { c.CacheTTL = -1 },
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				config := html.DefaultConfig()
				tt.modify(&config)
				_, err := html.New(config)
				if !errors.Is(err, html.ErrInvalidConfig) {
					t.Errorf("New() error = %v, want ErrInvalidConfig", err)
				}
			})
		}
	})

	t.Run("Config presets", func(t *testing.T) {
		t.Run("ConfigForRSS", func(t *testing.T) {
			config := html.ConfigForRSS()
			if config.ExtractArticle {
				t.Error("ConfigForRSS should disable article extraction")
			}
			if !config.PreserveImages {
				t.Error("ConfigForRSS should preserve images")
			}
		})

		t.Run("ConfigForSearchIndex", func(t *testing.T) {
			config := html.ConfigForSearchIndex()
			if !config.ExtractArticle {
				t.Error("ConfigForSearchIndex should enable article extraction")
			}
			if !config.PreserveVideos {
				t.Error("ConfigForSearchIndex should preserve videos")
			}
		})

		t.Run("ConfigForSummary", func(t *testing.T) {
			config := html.ConfigForSummary()
			if config.PreserveImages {
				t.Error("ConfigForSummary should not preserve images")
			}
			if config.PreserveLinks {
				t.Error("ConfigForSummary should not preserve links")
			}
		})

		t.Run("ConfigForMarkdown", func(t *testing.T) {
			config := html.ConfigForMarkdown()
			if config.InlineImageFormat != "markdown" {
				t.Errorf("InlineImageFormat = %q, want 'markdown'", config.InlineImageFormat)
			}
		})
	})
}

// ============================================================================
// BASIC EXTRACTION TESTS
// ============================================================================

func TestBasicExtraction(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	t.Run("simple HTML", func(t *testing.T) {
		result, err := p.ExtractWithDefaults(`<html><body><p>Hello World</p></body></html>`)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if !strings.Contains(result.Text, "Hello World") {
			t.Errorf("Text should contain 'Hello World', got %q", result.Text)
		}
	})

	t.Run("empty HTML", func(t *testing.T) {
		result, err := p.ExtractWithDefaults("")
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if result.Text != "" {
			t.Errorf("Text should be empty, got %q", result.Text)
		}
	})

	t.Run("whitespace only", func(t *testing.T) {
		result, err := p.ExtractWithDefaults("   \n\t  ")
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if result.Text != "" {
			t.Errorf("Text should be empty, got %q", result.Text)
		}
	})

	t.Run("malformed HTML handled gracefully", func(t *testing.T) {
		result, err := p.ExtractWithDefaults(`<html><body><p>Unclosed paragraph`)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if !strings.Contains(result.Text, "Unclosed paragraph") {
			t.Error("Text should contain 'Unclosed paragraph'")
		}
	})
}

// ============================================================================
// INPUT VALIDATION TESTS
// ============================================================================

func TestInputValidation(t *testing.T) {
	t.Parallel()

	t.Run("input size limit enforced", func(t *testing.T) {
		config := html.Config{
			MaxInputSize:       100,
			MaxCacheEntries:    10,
			CacheTTL:           time.Hour,
			WorkerPoolSize:     4,
			EnableSanitization: true,
			MaxDepth:           100,
		}
		p, _ := html.New(config)
		defer p.Close()

		largeHTML := strings.Repeat("a", 200)
		_, err := p.Extract(largeHTML, html.DefaultExtractConfig())
		if err == nil {
			t.Fatal("Extract() should fail with input too large")
		}
	})

	t.Run("max depth enforced", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxDepth = 5
		p, _ := html.New(config)
		defer p.Close()

		deepHTML := "<div>" + strings.Repeat("<div>", 10) + "content" + strings.Repeat("</div>", 10) + "</div>"
		_, err := p.Extract(deepHTML, html.DefaultExtractConfig())
		if err != html.ErrMaxDepthExceeded {
			t.Errorf("Expected ErrMaxDepthExceeded, got: %v", err)
		}
	})

	t.Run("processing timeout", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping timeout test in short mode")
		}

		var sb strings.Builder
		for i := 0; i < 500; i++ {
			sb.WriteString("<div>")
		}
		sb.WriteString("Content")
		for i := 0; i < 500; i++ {
			sb.WriteString("</div>")
		}

		config := html.DefaultConfig()
		config.ProcessingTimeout = 1 * time.Nanosecond
		p, _ := html.New(config)
		defer p.Close()

		_, err := p.Extract(sb.String(), html.DefaultExtractConfig())
		if err != html.ErrProcessingTimeout {
			t.Errorf("Expected ErrProcessingTimeout, got: %v", err)
		}
	})

	t.Run("disabled timeout works", func(t *testing.T) {
		htmlContent := `<html><body><article><h1>Test</h1><p>Content</p></article></body></html>`
		config := html.DefaultConfig()
		config.ProcessingTimeout = 0
		p, _ := html.New(config)
		defer p.Close()

		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if result.Title != "Test" {
			t.Errorf("Title = %q, want 'Test'", result.Title)
		}
	})
}

// ============================================================================
// FILE EXTRACTION TESTS
// ============================================================================

func TestFileExtraction(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	t.Run("empty file path", func(t *testing.T) {
		_, err := p.ExtractFromFile("", html.DefaultExtractConfig())
		if err == nil {
			t.Fatal("ExtractFromFile() should fail with empty path")
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := p.ExtractFromFile("nonexistent.html", html.DefaultExtractConfig())
		if err == nil {
			t.Fatal("ExtractFromFile() should fail with non-existent file")
		}
	})

	t.Run("valid file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.html")
		content := `<html><body><h1>Test</h1><p>Content</p></body></html>`

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		result, err := p.ExtractFromFile(filePath, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("ExtractFromFile() failed: %v", err)
		}
		if result.Title != "Test" {
			t.Errorf("Title = %q, want 'Test'", result.Title)
		}
	})

	t.Run("ExtractFromFile package function", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.html")
		content := `<html><body><h1>Package Test</h1><p>Content</p></body></html>`

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		result, err := html.ExtractFromFile(filePath)
		if err != nil {
			t.Fatalf("ExtractFromFile() failed: %v", err)
		}
		if result.Title != "Package Test" {
			t.Errorf("Title = %q, want 'Package Test'", result.Title)
		}
	})
}

// ============================================================================
// BATCH PROCESSING TESTS
// ============================================================================

func TestBatchProcessing(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	t.Run("normal batch", func(t *testing.T) {
		inputs := []string{
			`<html><body><article><h1>A1</h1><p>C1</p></article></body></html>`,
			`<html><body><article><h1>A2</h1><p>C2</p></article></body></html>`,
			`<html><body><article><h1>A3</h1><p>C3</p></article></body></html>`,
		}

		results, err := p.ExtractBatch(inputs, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("ExtractBatch() failed: %v", err)
		}
		if len(results) != 3 {
			t.Errorf("Got %d results, want 3", len(results))
		}
		for i, r := range results {
			expected := fmt.Sprintf("A%d", i+1)
			if r.Title != expected {
				t.Errorf("Result[%d].Title = %q, want %q", i, r.Title, expected)
			}
		}
	})

	t.Run("empty batch", func(t *testing.T) {
		results, err := p.ExtractBatch([]string{}, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("ExtractBatch() failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Got %d results, want 0", len(results))
		}
	})

	t.Run("batch with partial failures", func(t *testing.T) {
		config := html.Config{
			MaxInputSize:       100,
			MaxCacheEntries:    10,
			CacheTTL:           time.Hour,
			WorkerPoolSize:     4,
			EnableSanitization: true,
			MaxDepth:           100,
		}
		p, _ := html.New(config)
		defer p.Close()

		inputs := []string{
			`<html><body><p>Valid</p></body></html>`,
			strings.Repeat("a", 200), // Too large
			`<html><body><p>Another valid</p></body></html>`,
		}

		results, err := p.ExtractBatch(inputs, html.DefaultExtractConfig())
		if err == nil {
			t.Fatal("ExtractBatch() should return error for partial failures")
		}
		if len(results) != 3 {
			t.Errorf("Got %d results, want 3", len(results))
		}
		if results[0] == nil || results[2] == nil {
			t.Error("Valid results should not be nil")
		}
	})

	t.Run("ExtractBatchFiles", func(t *testing.T) {
		tmpDir := t.TempDir()
		files := []string{
			filepath.Join(tmpDir, "file1.html"),
			filepath.Join(tmpDir, "file2.html"),
		}

		for i, file := range files {
			content := fmt.Sprintf(`<html><body><h1>Title %d</h1></body></html>`, i+1)
			if err := os.WriteFile(file, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}

		results, err := p.ExtractBatchFiles(files, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("ExtractBatchFiles() failed: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Got %d results, want 2", len(results))
		}
	})
}

// ============================================================================
// CONTENT EXTRACTION TESTS
// ============================================================================

func TestContentExtraction(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	t.Run("article with title", func(t *testing.T) {
		htmlContent := `<html><head><title>Page Title</title></head><body><article><h1>Article</h1><p>Content</p></article></body></html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if result.Title != "Page Title" {
			t.Errorf("Title = %q, want 'Page Title'", result.Title)
		}
		if !strings.Contains(result.Text, "Content") {
			t.Error("Text should contain 'Content'")
		}
	})

	t.Run("multiple paragraphs", func(t *testing.T) {
		htmlContent := `<html><body><article><h1>Title</h1><p>P1</p><p>P2</p><p>P3</p></article></body></html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		// Check that paragraphs are present
		if !strings.Contains(result.Text, "P1") || !strings.Contains(result.Text, "P2") || !strings.Contains(result.Text, "P3") {
			t.Error("Text should contain all paragraphs")
		}
	})

	t.Run("nested elements", func(t *testing.T) {
		htmlContent := `<html><body><div><span><strong>Bold text</strong></span></div></body></html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if !strings.Contains(result.Text, "Bold text") {
			t.Error("Text should contain 'Bold text'")
		}
	})

	t.Run("scripts and styles removed", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<script>alert('test');</script>
				<style>body{color:red;}</style>
				<p>Visible content</p>
			</body></html>
		`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if strings.Contains(result.Text, "alert") || strings.Contains(result.Text, "color:red") {
			t.Error("Text should not contain script or style content")
		}
		if !strings.Contains(result.Text, "Visible content") {
			t.Error("Text should contain visible content")
		}
	})

	t.Run("word count calculated", func(t *testing.T) {
		htmlContent := `<html><body><p>This is a test with several words.</p></body></html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if result.WordCount == 0 {
			t.Error("WordCount should be > 0")
		}
	})

	t.Run("reading time calculated", func(t *testing.T) {
		htmlContent := `<html><body><p>Word1 word2 word3 word4 word5</p></body></html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if result.ReadingTime == 0 {
			t.Error("ReadingTime should be > 0")
		}
	})
}

// ============================================================================
// MEDIA EXTRACTION TESTS
// ============================================================================

func TestMediaExtraction(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	t.Run("images extracted", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<img src="img1.jpg" alt="Image 1" width="800" height="600">
				<img src="img2.png" alt="Image 2">
			</body></html>
		`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if len(result.Images) != 2 {
			t.Errorf("Got %d images, want 2", len(result.Images))
		}
		if result.Images[0].URL != "img1.jpg" {
			t.Errorf("Images[0].URL = %q, want 'img1.jpg'", result.Images[0].URL)
		}
		if result.Images[0].Alt != "Image 1" {
			t.Errorf("Images[0].Alt = %q, want 'Image 1'", result.Images[0].Alt)
		}
		if result.Images[0].Width != "800" {
			t.Errorf("Images[0].Width = %q, want '800'", result.Images[0].Width)
		}
	})

	t.Run("videos extracted", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<video src="video.mp4" poster="poster.jpg"></video>
				<video><source src="video2.webm" type="video/webm"></video>
			</body></html>
		`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if len(result.Videos) == 0 {
			t.Error("Should extract videos")
		}
	})

	t.Run("audios extracted", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<audio src="audio.mp3"></audio>
				<audio><source src="audio2.ogg" type="audio/ogg"></audio>
			</body></html>
		`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if len(result.Audios) == 0 {
			t.Error("Should extract audios")
		}
	})

	t.Run("iframe embed videos extracted", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<iframe src="https://www.youtube.com/embed/test123" width="640" height="480"></iframe>
				<iframe src="https://player.vimeo.com/video/456789"></iframe>
			</body></html>
		`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if len(result.Videos) == 0 {
			t.Error("Should extract iframe videos")
		}
	})

	t.Run("non-video iframe ignored", func(t *testing.T) {
		htmlContent := `<html><body><iframe src="https://example.com/page.html"></iframe></body></html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if len(result.Videos) != 0 {
			t.Errorf("Got %d videos, want 0", len(result.Videos))
		}
	})
}

// ============================================================================
// LINK EXTRACTION TESTS
// ============================================================================

func TestLinkExtraction(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	t.Run("links extracted with details", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<a href="https://example.com" title="Example">External Link</a>
				<a href="/internal" rel="nofollow">Internal Link</a>
			</body></html>
		`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if len(result.Links) != 2 {
			t.Errorf("Got %d links, want 2", len(result.Links))
		}
		if !result.Links[0].IsExternal {
			t.Error("First link should be external")
		}
		if !result.Links[1].IsNoFollow {
			t.Error("Second link should have nofollow")
		}
	})

	t.Run("empty link handled", func(t *testing.T) {
		htmlContent := `<html><body><a href=""></a></body></html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if len(result.Links) != 0 {
			t.Errorf("Got %d links, want 0", len(result.Links))
		}
	})
}

// ============================================================================
// PARAGRAPH SPACING TESTS
// ============================================================================

func TestParagraphSpacing(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	t.Run("paragraphs separated by double newlines", func(t *testing.T) {
		htmlContent := `<html><body><p>First paragraph.</p><p>Second paragraph.</p></body></html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Check for double newlines between paragraphs
		if !strings.Contains(result.Text, "\n\n") {
			t.Error("Paragraphs should be separated by double newlines")
		}
		// Verify both paragraphs are present
		if !strings.Contains(result.Text, "First paragraph") {
			t.Error("First paragraph should be present")
		}
		if !strings.Contains(result.Text, "Second paragraph") {
			t.Error("Second paragraph should be present")
		}
	})

	t.Run("long paragraph text from example", func(t *testing.T) {
		htmlContent := `<html><body>
			<p>We provide our customers with a suite of broadband connectivity services, including fixed Internet, WiFi and mobile, which when bundled together provides our customers with a differentiated converged connectivity experience while saving consumers money.</p>
			<p>This is another paragraph with different content.</p>
		</body></html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Check for double newlines between paragraphs
		if !strings.Contains(result.Text, "\n\n") {
			t.Error("Paragraphs should be separated by double newlines")
		}
		// Verify the long text is preserved
		if !strings.Contains(result.Text, "We provide our customers") {
			t.Error("Long paragraph content should be preserved")
		}
		if !strings.Contains(result.Text, "differentiated converged connectivity experience") {
			t.Error("Long paragraph should be complete")
		}
	})

	t.Run("multiple consecutive paragraphs", func(t *testing.T) {
		htmlContent := `<html><body>
			<p>Paragraph 1</p>
			<p>Paragraph 2</p>
			<p>Paragraph 3</p>
		</body></html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Count double newlines - should have at least 2 for 3 paragraphs
		doubleNewlineCount := strings.Count(result.Text, "\n\n")
		if doubleNewlineCount < 2 {
			t.Errorf("Expected at least 2 double newlines for 3 paragraphs, got %d", doubleNewlineCount)
		}
	})

	t.Run("headings and paragraphs", func(t *testing.T) {
		htmlContent := `<html><body>
			<h1>Title</h1>
			<p>First paragraph after heading.</p>
			<p>Second paragraph.</p>
		</body></html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Check for double newlines
		if !strings.Contains(result.Text, "\n\n") {
			t.Error("Headings and paragraphs should be separated by double newlines")
		}
		// Verify content
		if !strings.Contains(result.Text, "Title") {
			t.Error("Heading should be present")
		}
		if !strings.Contains(result.Text, "First paragraph") {
			t.Error("First paragraph should be present")
		}
	})

	t.Run("divs as paragraphs", func(t *testing.T) {
		htmlContent := `<html><body>
			<div>First div content</div>
			<div>Second div content</div>
		</body></html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Check for double newlines between divs
		if !strings.Contains(result.Text, "\n\n") {
			t.Error("Divs should be separated by double newlines")
		}
	})

	t.Run("list items should not have double spacing", func(t *testing.T) {
		htmlContent := `<html><body>
			<ul>
				<li>Item 1</li>
				<li>Item 2</li>
				<li>Item 3</li>
			</ul>
		</body></html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// List items should not have double newlines between them
		// They should be separated by single newlines
		lines := strings.Split(result.Text, "\n")
		nonEmptyLines := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				nonEmptyLines++
			}
		}
		// For list items, we expect them to be on separate lines but not paragraph-separated
		if nonEmptyLines < 3 {
			t.Errorf("Expected at least 3 non-empty lines for list items, got %d", nonEmptyLines)
		}
	})

	t.Run("article with multiple sections", func(t *testing.T) {
		htmlContent := `<!DOCTYPE html>
		<html>
		<head><title>Test Article</title></head>
		<body>
			<article>
				<h1>Main Title</h1>
				<p>We provide our customers with a suite of broadband connectivity services.</p>
				<p>This is the second paragraph of the article.</p>
				<h2>Section Title</h2>
				<p>Content under section heading.</p>
			</article>
		</body>
		</html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Check for proper paragraph spacing throughout
		doubleNewlineCount := strings.Count(result.Text, "\n\n")
		if doubleNewlineCount < 3 {
			t.Errorf("Expected multiple double newlines for article structure, got %d", doubleNewlineCount)
		}
		// Verify all content is preserved
		if !strings.Contains(result.Text, "Main Title") {
			t.Error("Main title should be present")
		}
		if !strings.Contains(result.Text, "broadband connectivity services") {
			t.Error("First paragraph content should be present")
		}
		if !strings.Contains(result.Text, "Section Title") {
			t.Error("Section title should be present")
		}
	})

	t.Run("mixed content with inline elements", func(t *testing.T) {
		htmlContent := `<html><body>
			<p>This is a <strong>paragraph</strong> with <em>inline</em> elements.</p>
			<p>This is another <a href="#">paragraph</a> with links.</p>
		</body></html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Check for double newlines
		if !strings.Contains(result.Text, "\n\n") {
			t.Error("Paragraphs with inline elements should still be separated by double newlines")
		}
		// Verify inline content is preserved
		if !strings.Contains(result.Text, "paragraph with inline elements") {
			t.Error("Inline elements should be preserved in text")
		}
	})

	t.Run("blockquote should create paragraph separation", func(t *testing.T) {
		htmlContent := `<html><body>
			<p>Regular paragraph before quote.</p>
			<blockquote>This is a quoted text.</blockquote>
			<p>Regular paragraph after quote.</p>
		</body></html>`
		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Blockquotes should create paragraph separation
		if !strings.Contains(result.Text, "\n\n") {
			t.Error("Blockquotes should create paragraph separation")
		}
		if !strings.Contains(result.Text, "quoted text") {
			t.Error("Blockquote content should be present")
		}
	})
}

// ============================================================================
// TABLE FORMAT TESTS
// ============================================================================

func TestTableFormats(t *testing.T) {
	t.Parallel()

	t.Run("markdown table with alignment", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr><th align="left">Name</th><th align="center">Age</th><th align="right">Score</th></tr>
					<tr><td>Alice</td><td>25</td><td>95</td></tr>
					<tr><td>Bob</td><td>30</td><td>87</td></tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "markdown"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Check for alignment markers
		if !strings.Contains(result.Text, ":---") {
			t.Error("Markdown table should have left alignment marker")
		}
		if !strings.Contains(result.Text, ":--:") {
			t.Error("Markdown table should have center alignment marker")
		}
		if !strings.Contains(result.Text, "---:") {
			t.Error("Markdown table should have right alignment marker")
		}

		// Check content is preserved
		if !strings.Contains(result.Text, "Alice") {
			t.Error("Table content should be preserved")
		}
	})

	t.Run("html table format with alignment", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr><th align="left">Name</th><th align="center">Value</th></tr>
					<tr><td>Item1</td><td>100</td></tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Should have HTML table tags
		if !strings.Contains(result.Text, "<table>") {
			t.Error("HTML format should contain <table> tag")
		}
		if !strings.Contains(result.Text, "style=\"text-align:left\"") {
			t.Error("HTML table should preserve left alignment")
		}
		if !strings.Contains(result.Text, "style=\"text-align:center\"") {
			t.Error("HTML table should preserve center alignment")
		}
	})

	t.Run("html table with cell merging", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<th colspan="2">Merged Header</th>
					</tr>
					<tr>
						<td rowspan="2">Merged Cell</td>
						<td>Cell 2</td>
					</tr>
					<tr>
						<td>Cell 3</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Should preserve colspan and rowspan
		if !strings.Contains(result.Text, "colspan=") {
			t.Error("HTML table should preserve colspan")
		}
		if !strings.Contains(result.Text, "rowspan=") {
			t.Error("HTML table should preserve rowspan")
		}
		if !strings.Contains(result.Text, "Merged Header") {
			t.Error("Table content should be preserved")
		}
	})

	t.Run("table with style attribute alignment", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr><th style="text-align:right">Total</th></tr>
					<tr><td style="text-align:right">$100</td></tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "markdown"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Should detect alignment from style attribute
		if !strings.Contains(result.Text, "---:") {
			t.Error("Should detect right alignment from style attribute")
		}
	})

	t.Run("table format validation", func(t *testing.T) {
		p := html.NewWithDefaults()
		defer p.Close()

		htmlContent := `<html><body><table><tr><td>Test</td></tr></table></body></html>`

		// Invalid format should default to markdown
		config := html.DefaultExtractConfig()
		config.TableFormat = "invalid"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Should still produce markdown output
		if !strings.Contains(result.Text, "|") {
			t.Error("Invalid format should default to markdown")
		}
	})

	t.Run("empty table handling", func(t *testing.T) {
		htmlContent := `<html><body><table></table></body></html>`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "markdown"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Empty table should not produce table output
		if strings.Contains(result.Text, "| --- |") {
			t.Error("Empty table should not produce separator")
		}
	})

	t.Run("default format is markdown", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr><th>Header</th></tr>
					<tr><td>Data</td></tr>
				</table>
			</body></html>
		`

		result, err := html.Extract(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Default should be markdown
		if !strings.Contains(result.Text, "| --- |") {
			t.Error("Default table format should be markdown")
		}
	})
}

// ============================================================================
// TABLE COLUMN WIDTH TESTS
// ============================================================================

func TestTableColumnWidths(t *testing.T) {
	t.Parallel()

	t.Run("html table preserves width from style attribute", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<th style="width:1.0%">Name</th>
						<th style="width:50%">Value</th>
						<th style="width:49%">Notes</th>
					</tr>
					<tr>
						<td>Item 1</td>
						<td>100</td>
						<td>First</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Check for width preservation in style attribute
		if !strings.Contains(result.Text, "width:1.0%") {
			t.Error("HTML table should preserve width:1.0%")
		}
		if !strings.Contains(result.Text, "width:50%") {
			t.Error("HTML table should preserve width:50%")
		}
		if !strings.Contains(result.Text, "width:49%") {
			t.Error("HTML table should preserve width:49%")
		}
	})

	t.Run("html table preserves width from width attribute", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<th width="100">Column 1</th>
						<th width="200px">Column 2</th>
					</tr>
					<tr>
						<td>Data 1</td>
						<td>Data 2</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Check for width preservation from width attribute
		if !strings.Contains(result.Text, "width:100") {
			t.Error("HTML table should preserve width=100")
		}
		if !strings.Contains(result.Text, "width:200px") {
			t.Error("HTML table should preserve width=200px")
		}
	})

	t.Run("html table combines width with alignment", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<th style="width:30%; text-align:left">Name</th>
						<th style="width:70%; text-align:right">Value</th>
					</tr>
					<tr>
						<td>Item 1</td>
						<td>100</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Check that both width and alignment are preserved in the same style attribute
		if !strings.Contains(result.Text, "width:30%") {
			t.Error("HTML table should preserve width")
		}
		if !strings.Contains(result.Text, "text-align:left") {
			t.Error("HTML table should preserve alignment")
		}
		// Check they are in the same style attribute with semicolon separator
		if !strings.Contains(result.Text, "text-align:left;") && !strings.Contains(result.Text, ";text-align:left") {
			if !strings.Contains(result.Text, "width:30%;") && !strings.Contains(result.Text, ";width:30%") {
				t.Error("Width and alignment should be in same style attribute separated by semicolon")
			}
		}
	})

	t.Run("markdown table includes width comment", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<th style="width:1.0%">Name</th>
						<th style="width:98%">Value</th>
					</tr>
					<tr>
						<td>Item 1</td>
						<td>Data</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "markdown"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Check for width comment
		if !strings.Contains(result.Text, "<!-- Table metadata:") {
			t.Error("Markdown table should include table metadata comment")
		}
		if !strings.Contains(result.Text, "col:1=width:1.0%") {
			t.Error("Markdown table should include first column width")
		}
		if !strings.Contains(result.Text, "col:2=width:98%") {
			t.Error("Markdown table should include second column width")
		}
	})

	t.Run("table without width attributes works normally", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<th>Name</th>
						<th>Value</th>
					</tr>
					<tr>
						<td>Item 1</td>
						<td>100</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		// Test HTML format
		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Should not have width in output
		if strings.Contains(result.Text, "width:") {
			t.Error("Table without width should not add width attributes")
		}

		// Test Markdown format
		config.TableFormat = "markdown"
		result, err = p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Should not have width comment
		if strings.Contains(result.Text, "<!-- Column widths:") {
			t.Error("Markdown table without widths should not include width comment")
		}
	})

	t.Run("complex table with mixed width units", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<td style="width:50px">ID</td>
						<td style="width:20%">Name</td>
						<td style="width:1.5*">Score</td>
						<td>Description</td>
					</tr>
					<tr>
						<td>1</td>
						<td>Item</td>
						<td>95</td>
						<td>Test item</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Check for various width units
		if !strings.Contains(result.Text, "width:50px") {
			t.Error("HTML table should preserve pixel width")
		}
		if !strings.Contains(result.Text, "width:20%") {
			t.Error("HTML table should preserve percentage width")
		}
		// The description column has no width, so it shouldn't appear
	})

	t.Run("table preserves width with colspan", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<th style="width:50%" colspan="2">Merged Header</th>
						<th style="width:50%">Value</th>
					</tr>
					<tr>
						td>A</td>
						<td>B</td>
						<td>C</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Should preserve both width and colspan
		if !strings.Contains(result.Text, "colspan=") {
			t.Error("HTML table should preserve colspan")
		}
		if !strings.Contains(result.Text, "width:50%") {
			t.Error("HTML table should preserve width with colspan")
		}
	})
}

// ============================================================================
// TABLE ALIGNMENT PRESERVATION TESTS
// ============================================================================

func TestTableAlignmentPreservation(t *testing.T) {
	t.Parallel()

	t.Run("html table preserves justify alignment", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<td style="text-align:justify">This is justified text that should span the full width.</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Check for justify alignment in style attribute
		if !strings.Contains(result.Text, "text-align:justify") {
			t.Error("HTML table should preserve justify alignment")
		}
	})

	t.Run("html table preserves all alignment types", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<td style="text-align:left">Left</td>
						<td style="text-align:center">Center</td>
						<td style="text-align:right">Right</td>
						<td style="text-align:justify">Justify</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Check all alignment types
		if !strings.Contains(result.Text, "text-align:left") {
			t.Error("HTML table should preserve left alignment")
		}
		if !strings.Contains(result.Text, "text-align:center") {
			t.Error("HTML table should preserve center alignment")
		}
		if !strings.Contains(result.Text, "text-align:right") {
			t.Error("HTML table should preserve right alignment")
		}
		if !strings.Contains(result.Text, "text-align:justify") {
			t.Error("HTML table should preserve justify alignment")
		}
	})

	t.Run("html table preserves align attribute", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<td align="left">Left aligned</td>
						<td align="center">Center aligned</td>
						<td align="right">Right aligned</td>
						<td align="justify">Justify aligned</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// align attribute should be converted to style="text-align:..."
		if !strings.Contains(result.Text, "text-align:left") {
			t.Error("align='left' should convert to text-align:left")
		}
		if !strings.Contains(result.Text, "text-align:center") {
			t.Error("align='center' should convert to text-align:center")
		}
		if !strings.Contains(result.Text, "text-align:right") {
			t.Error("align='right' should convert to text-align:right")
		}
		if !strings.Contains(result.Text, "text-align:justify") {
			t.Error("align='justify' should convert to text-align:justify")
		}
	})

	t.Run("html table combines alignment with width", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<td style="width:50%; text-align:justify">Combined</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Should have both width and alignment in same style
		if !strings.Contains(result.Text, "width:50%") {
			t.Error("Should preserve width")
		}
		if !strings.Contains(result.Text, "text-align:justify") {
			t.Error("Should preserve justify alignment")
		}
		// Check they are in same style attribute with semicolon separator
		hasCombinedStyle := (strings.Contains(result.Text, "width:50%;text-align:justify") ||
			strings.Contains(result.Text, "text-align:justify;width:50%"))
		if !hasCombinedStyle {
			t.Error("Width and alignment should be in same style attribute")
		}
	})

	t.Run("markdown table with justify alignment shows warning", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<th style="text-align:left">Name</th>
						<th style="text-align:justify">Description</th>
					</tr>
					<tr>
						<td>Item</td>
						<td>Justified text content</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "markdown"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Should include metadata comment about justify alignment
		if !strings.Contains(result.Text, "<!-- Table metadata:") {
			t.Error("Markdown table should include metadata comment")
		}
		if !strings.Contains(result.Text, "align:justify") {
			t.Error("Metadata comment should indicate justify alignment")
		}
	})

	t.Run("table alignment priority: align over style", func(t *testing.T) {
		// When both align attribute and style attribute are present,
		// align attribute should take precedence
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<td align="center" style="text-align:left">Should be center</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Should use align attribute (center) not style (left)
		if !strings.Contains(result.Text, "text-align:center") {
			t.Error("align attribute should take precedence over style attribute")
		}
		if strings.Contains(result.Text, "text-align:left") {
			t.Error("Should not use style attribute when align is present")
		}
	})

	t.Run("extracts alignment from complex style attribute", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<td style="color:red; text-align:right; font-size:12px">Right aligned</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Should extract text-align from complex style
		if !strings.Contains(result.Text, "text-align:right") {
			t.Error("Should extract alignment from complex style attribute")
		}
	})

	t.Run("handles whitespace in style attribute", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<td style="text-align : center">Center with spaces</td>
						<td style="text-align:center">Center without spaces</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Both should be detected as center alignment
		count := strings.Count(result.Text, "text-align:center")
		if count < 2 {
			t.Errorf("Expected 2 center-aligned cells, got %d", count)
		}
	})

	t.Run("mixed case alignment attributes", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<table>
					<tr>
						<td ALIGN="Center">Mixed case align</td>
					</tr>
				</table>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		config := html.DefaultExtractConfig()
		config.TableFormat = "html"
		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Should handle case-insensitive align attribute
		if !strings.Contains(result.Text, "text-align:center") {
			t.Error("Should handle mixed case ALIGN attribute")
		}
	})
}

// ============================================================================
// CONVENIENCE API TESTS
// ============================================================================

func TestConvenienceAPIs(t *testing.T) {
	t.Parallel()

	t.Run("Extract", func(t *testing.T) {
		result, err := html.Extract(`<html><body><h1>Title</h1><p>Content</p></body></html>`)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}
		if result.Title != "Title" {
			t.Errorf("Title = %q, want 'Title'", result.Title)
		}
	})

	t.Run("ExtractText", func(t *testing.T) {
		text, err := html.ExtractText(`<html><body><p>Hello World</p></body></html>`)
		if err != nil {
			t.Fatalf("ExtractText() failed: %v", err)
		}
		if !strings.Contains(text, "Hello World") {
			t.Error("Text should contain 'Hello World'")
		}
	})

	t.Run("ExtractTitle", func(t *testing.T) {
		title, err := html.ExtractTitle(`<html><head><title>Test Title</title></head><body></body></html>`)
		if err != nil {
			t.Fatalf("ExtractTitle() failed: %v", err)
		}
		if title != "Test Title" {
			t.Errorf("Title = %q, want 'Test Title'", title)
		}
	})

	t.Run("ExtractImages", func(t *testing.T) {
		images, err := html.ExtractImages(`<html><body><img src="test.jpg" alt="Test"></body></html>`)
		if err != nil {
			t.Fatalf("ExtractImages() failed: %v", err)
		}
		if len(images) != 1 {
			t.Errorf("Got %d images, want 1", len(images))
		}
	})

	t.Run("ExtractVideos", func(t *testing.T) {
		videos, err := html.ExtractVideos(`<html><body><video src="test.mp4"></video></body></html>`)
		if err != nil {
			t.Fatalf("ExtractVideos() failed: %v", err)
		}
		if len(videos) == 0 {
			t.Error("Should extract videos")
		}
	})

	t.Run("ExtractAudios", func(t *testing.T) {
		audios, err := html.ExtractAudios(`<html><body><audio src="test.mp3"></audio></body></html>`)
		if err != nil {
			t.Fatalf("ExtractAudios() failed: %v", err)
		}
		if len(audios) == 0 {
			t.Error("Should extract audios")
		}
	})

	t.Run("ExtractLinks", func(t *testing.T) {
		links, err := html.ExtractLinks(`<html><body><a href="test.html">Link</a></body></html>`)
		if err != nil {
			t.Fatalf("ExtractLinks() failed: %v", err)
		}
		if len(links) == 0 {
			t.Error("Should extract links")
		}
	})

	t.Run("ExtractToMarkdown", func(t *testing.T) {
		markdown, err := html.ExtractToMarkdown(`<html><body><h1>Title</h1><p>Content</p></body></html>`)
		if err != nil {
			t.Fatalf("ExtractToMarkdown() failed: %v", err)
		}
		if markdown == "" {
			t.Error("Markdown should not be empty")
		}
	})

	t.Run("ExtractToJSON", func(t *testing.T) {
		jsonData, err := html.ExtractToJSON(`<html><head><title>Test</title></head><body><p>Content</p></body></html>`)
		if err != nil {
			t.Fatalf("ExtractToJSON() failed: %v", err)
		}
		if len(jsonData) == 0 {
			t.Error("JSON data should not be empty")
		}

		var result map[string]interface{}
		if err := json.Unmarshal(jsonData, &result); err != nil {
			t.Errorf("Invalid JSON: %v", err)
		}
		if _, ok := result["title"]; !ok {
			t.Error("JSON should have 'title' field")
		}
		if _, ok := result["text"]; !ok {
			t.Error("JSON should have 'text' field")
		}
	})

	t.Run("Summarize", func(t *testing.T) {
		htmlContent := `<html><body><p>Word1 word2 word3 word4 word5 word6</p></body></html>`
		summary, err := html.Summarize(htmlContent, 3)
		if err != nil {
			t.Fatalf("Summarize() failed: %v", err)
		}
		words := strings.Fields(summary)
		if len(words) > 3 {
			t.Errorf("Summary has %d words, want <= 3", len(words))
		}
	})

	t.Run("ExtractAndClean", func(t *testing.T) {
		cleaned, err := html.ExtractAndClean(`<html><body><p>Text</p><p>   </p><p>More</p></body></html>`)
		if err != nil {
			t.Fatalf("ExtractAndClean() failed: %v", err)
		}
		if cleaned == "" {
			t.Error("Cleaned text should not be empty")
		}
		// Empty paragraphs should be removed
		if strings.Contains(cleaned, "   ") {
			t.Error("Empty paragraphs should be cleaned")
		}
	})

	t.Run("GetWordCount", func(t *testing.T) {
		count, err := html.GetWordCount(`<html><body><p>Word1 word2 word3</p></body></html>`)
		if err != nil {
			t.Fatalf("GetWordCount() failed: %v", err)
		}
		if count != 3 {
			t.Errorf("WordCount = %d, want 3", count)
		}
	})

	t.Run("GetReadingTime", func(t *testing.T) {
		minutes, err := html.GetReadingTime(`<html><body><p>Word1 word2 word3</p></body></html>`)
		if err != nil {
			t.Fatalf("GetReadingTime() failed: %v", err)
		}
		if minutes <= 0 {
			t.Errorf("ReadingTime = %f, want > 0", minutes)
		}
	})
}

// ============================================================================
// COMPREHENSIVE LINK EXTRACTION TESTS
// ============================================================================

func TestExtractAllLinks(t *testing.T) {
	t.Parallel()

	t.Run("comprehensive link extraction", func(t *testing.T) {
		htmlContent := `
			<html>
			<head>
				<link rel="stylesheet" href="styles.css">
				<script src="app.js"></script>
			</head>
			<body>
				<a href="https://example.com">External Link</a>
				<a href="/internal">Internal Link</a>
				<img src="image.jpg" alt="Test Image">
				<video src="video.mp4"></video>
			</body>
			</html>
		`

		links, err := html.ExtractAllLinks(htmlContent)
		if err != nil {
			t.Fatalf("ExtractAllLinks() failed: %v", err)
		}

		if len(links) == 0 {
			t.Fatal("ExtractAllLinks() returned no links")
		}

		types := make(map[string]bool)
		for _, link := range links {
			types[link.Type] = true
		}

		expectedTypes := []string{"css", "js", "link", "image", "video"}
		for _, expectedType := range expectedTypes {
			if !types[expectedType] {
				t.Errorf("Expected link type %q not found", expectedType)
			}
		}
	})

	t.Run("link deduplication", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<a href="page.html">Link 1</a>
				<a href="page.html">Link 2</a>
				<img src="img.jpg">
				<img src="img.jpg" alt="Same">
			</body></html>
		`

		links, err := html.ExtractAllLinks(htmlContent)
		if err != nil {
			t.Fatalf("ExtractAllLinks() failed: %v", err)
		}

		urlCounts := make(map[string]int)
		for _, link := range links {
			urlCounts[link.URL]++
		}

		for url, count := range urlCounts {
			if count > 1 {
				t.Errorf("URL %q appears %d times (should be deduplicated)", url, count)
			}
		}
	})

	t.Run("relative URL resolution with manual base URL", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<a href="/about">About</a>
				<a href="contact.html">Contact</a>
				<img src="images/logo.jpg">
			</body></html>
		`

		config := html.DefaultLinkExtractionConfig()
		config.BaseURL = "https://mysite.com/"
		links, err := html.ExtractAllLinks(htmlContent, config)
		if err != nil {
			t.Fatalf("ExtractAllLinks() failed: %v", err)
		}

		expectedURLs := map[string]bool{
			"https://mysite.com/about":           false,
			"https://mysite.com/contact.html":    false,
			"https://mysite.com/images/logo.jpg": false,
		}

		for _, link := range links {
			if _, exists := expectedURLs[link.URL]; exists {
				expectedURLs[link.URL] = true
			}
		}

		for url, found := range expectedURLs {
			if !found {
				t.Errorf("Expected URL %q not found", url)
			}
		}
	})

	t.Run("selective extraction by type", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<img src="image.jpg">
				<video src="video.mp4"></video>
			</body></html>
		`

		config := html.DefaultLinkExtractionConfig()
		config.IncludeImages = false
		config.IncludeVideos = false

		links, err := html.ExtractAllLinks(htmlContent, config)
		if err != nil {
			t.Fatalf("ExtractAllLinks() failed: %v", err)
		}

		for _, link := range links {
			if link.Type == "image" {
				t.Error("Image links should not be included")
			}
			if link.Type == "video" {
				t.Error("Video links should not be included")
			}
		}
	})

	t.Run("GroupLinksByType", func(t *testing.T) {
		links := []html.LinkResource{
			{URL: "test.css", Type: "css"},
			{URL: "test.js", Type: "js"},
			{URL: "test2.css", Type: "css"},
		}

		grouped := html.GroupLinksByType(links)
		if len(grouped["css"]) != 2 {
			t.Errorf("Got %d CSS links, want 2", len(grouped["css"]))
		}
		if len(grouped["js"]) != 1 {
			t.Errorf("Got %d JS links, want 1", len(grouped["js"]))
		}
	})
}

// ============================================================================
// CACHE AND STATISTICS TESTS
// ============================================================================

func TestCache(t *testing.T) {
	t.Parallel()

	t.Run("cache hit on repeated extraction", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxCacheEntries = 100
		config.CacheTTL = time.Hour
		p, _ := html.New(config)
		defer p.Close()

		htmlContent := `<html><body><p>Test content</p></body></html>`

		// First extraction - cache miss
		result1, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		stats1 := p.GetStatistics()
		if stats1.CacheHits != 0 {
			t.Errorf("First extraction should have 0 cache hits, got %d", stats1.CacheHits)
		}

		// Second extraction - cache hit
		result2, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		stats2 := p.GetStatistics()
		if stats2.CacheHits != 1 {
			t.Errorf("Second extraction should have 1 cache hit, got %d", stats2.CacheHits)
		}

		// Results should be identical
		if result1.Text != result2.Text {
			t.Error("Cached result should match original")
		}
	})

	t.Run("cache cleared", func(t *testing.T) {
		p := html.NewWithDefaults()
		defer p.Close()

		htmlContent := `<html><body><p>Test</p></body></html>`

		p.Extract(htmlContent, html.DefaultExtractConfig())
		p.ClearCache()

		stats := p.GetStatistics()
		if stats.CacheHits != 0 || stats.CacheMisses != 0 {
			t.Error("Cache stats should be reset after ClearCache()")
		}
	})

	t.Run("cache disabled", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxCacheEntries = 0
		p, _ := html.New(config)
		defer p.Close()

		htmlContent := `<html><body><p>Test</p></body></html>`

		p.Extract(htmlContent, html.DefaultExtractConfig())
		p.Extract(htmlContent, html.DefaultExtractConfig())

		stats := p.GetStatistics()
		if stats.CacheHits != 0 {
			t.Errorf("With cache disabled, cache hits should be 0, got %d", stats.CacheHits)
		}
	})

	t.Run("statistics tracked", func(t *testing.T) {
		p := html.NewWithDefaults()
		defer p.Close()

		stats := p.GetStatistics()
		if stats.TotalProcessed != 0 {
			t.Errorf("Initial TotalProcessed = %d, want 0", stats.TotalProcessed)
		}

		htmlContent := `<html><body><p>Test</p></body></html>`
		p.Extract(htmlContent, html.DefaultExtractConfig())

		stats = p.GetStatistics()
		if stats.TotalProcessed != 1 {
			t.Errorf("TotalProcessed = %d, want 1", stats.TotalProcessed)
		}
	})
}

// ============================================================================
// CONCURRENCY TESTS
// ============================================================================

func TestConcurrency(t *testing.T) {
	t.Parallel()

	t.Run("concurrent extraction", func(t *testing.T) {
		p := html.NewWithDefaults()
		defer p.Close()

		htmlContent := `<html><body><article><h1>Title</h1><p>Content</p></article></body></html>`
		const goroutines = 50
		var wg sync.WaitGroup
		errors := make(chan error, goroutines)

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
				if err != nil {
					errors <- err
					return
				}
				if result.Title != "Title" {
					errors <- fmt.Errorf("wrong title: %q", result.Title)
				}
			}()
		}

		wg.Wait()
		close(errors)
		for err := range errors {
			t.Error(err)
		}
	})

	t.Run("concurrent cache operations", func(t *testing.T) {
		p := html.NewWithDefaults()
		defer p.Close()

		htmlContent := `<html><body><p>Test</p></body></html>`

		const goroutines = 50
		var wg sync.WaitGroup

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				p.Extract(htmlContent, html.DefaultExtractConfig())
			}()
		}

		wg.Wait()

		stats := p.GetStatistics()
		if stats.TotalProcessed != goroutines {
			t.Errorf("TotalProcessed = %d, want %d", stats.TotalProcessed, goroutines)
		}
	})

	t.Run("concurrent with cache clearing", func(t *testing.T) {
		p := html.NewWithDefaults()
		defer p.Close()

		const goroutines = 20
		var wg sync.WaitGroup

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				htmlContent := fmt.Sprintf(`<html><body><p>Content %d</p></body></html>`, id)
				p.Extract(htmlContent, html.DefaultExtractConfig())
				if id%5 == 0 {
					p.ClearCache()
				}
			}(i)
		}

		wg.Wait()

		stats := p.GetStatistics()
		if stats.TotalProcessed != goroutines {
			t.Errorf("TotalProcessed = %d, want %d", stats.TotalProcessed, goroutines)
		}
	})
}

// ============================================================================
// COMPATIBILITY TESTS
// ============================================================================

func TestCompatibilityWithStdHTML(t *testing.T) {
	t.Parallel()

	t.Run("Parse compatibility", func(t *testing.T) {
		htmlContent := `<html><head><title>Test</title></head><body><p>Content</p></body></html>`

		ourDoc, ourErr := html.Parse(strings.NewReader(htmlContent))
		stdDoc, stdErr := stdhtml.Parse(strings.NewReader(htmlContent))

		if (ourErr == nil) != (stdErr == nil) {
			t.Errorf("Parse error mismatch: our=%v, std=%v", ourErr, stdErr)
		}

		if ourDoc.Type != stdDoc.Type {
			t.Errorf("Parse document type mismatch: got %v, want %v", ourDoc.Type, stdDoc.Type)
		}
	})

	t.Run("EscapeString compatibility", func(t *testing.T) {
		tests := []string{
			"<html>",
			"a&b",
			`"quoted"`,
			"<script>alert('xss')</script>",
		}

		for _, input := range tests {
			our := html.EscapeString(input)
			std := stdhtml.EscapeString(input)
			if our != std {
				t.Errorf("EscapeString(%q) mismatch: our=%q, std=%q", input, our, std)
			}
		}
	})

	t.Run("UnescapeString compatibility", func(t *testing.T) {
		tests := []string{
			"&lt;html&gt;",
			"&amp;",
			"&nbsp;",
			"&copy;",
		}

		for _, input := range tests {
			our := html.UnescapeString(input)
			std := stdhtml.UnescapeString(input)
			if our != std {
				t.Errorf("UnescapeString(%q) mismatch: our=%q, std=%q", input, our, std)
			}
		}
	})

	t.Run("Tokenizer compatibility", func(t *testing.T) {
		htmlContent := "<p>Test</p><div>Content</div>"

		ourTokenizer := html.NewTokenizer(strings.NewReader(htmlContent))
		stdTokenizer := stdhtml.NewTokenizer(strings.NewReader(htmlContent))

		for {
			ourTT := ourTokenizer.Next()
			stdTT := stdTokenizer.Next()

			if ourTT != stdTT {
				t.Errorf("Token type mismatch: got %v, want %v", ourTT, stdTT)
				break
			}

			if ourTT == stdhtml.ErrorToken {
				break
			}
		}
	})
}

// ============================================================================
// EDGE CASES AND ERROR HANDLING TESTS
// ============================================================================

func TestEdgeCases(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name    string
		html    string
		wantErr bool
		check   func(*testing.T, *html.Result, error)
	}{
		{
			name:    "unicode characters",
			html:    `<html><body><p>Hello 世界 🌍</p></body></html>`,
			wantErr: false,
			check: func(t *testing.T, r *html.Result, err error) {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if !strings.Contains(r.Text, "世界") {
					t.Error("Text should contain Chinese characters")
				}
			},
		},
		{
			name:    "special HTML entities",
			html:    `<html><body><p>&lt;&gt;&amp;&quot;&#39;</p></body></html>`,
			wantErr: false,
			check: func(t *testing.T, r *html.Result, err error) {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if !strings.Contains(r.Text, "<") {
					t.Error("Text should contain decoded entities")
				}
			},
		},
		{
			name:    "mixed case tags",
			html:    `<HTML><BODY><P>Content</P></BODY></HTML>`,
			wantErr: false,
			check: func(t *testing.T, r *html.Result, err error) {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if !strings.Contains(r.Text, "Content") {
					t.Error("Text should contain 'Content'")
				}
			},
		},
		{
			name:    "self-closing tags",
			html:    `<html><body><br/><hr/><p>Content</p></body></html>`,
			wantErr: false,
			check: func(t *testing.T, r *html.Result, err error) {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			},
		},
		{
			name:    "CDATA sections",
			html:    `<html><body><![CDATA[Some data]]><p>Content</p></body></html>`,
			wantErr: false,
			check: func(t *testing.T, r *html.Result, err error) {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			},
		},
		{
			name:    "comments",
			html:    `<html><body><!-- Comment --><p>Content</p></body></html>`,
			wantErr: false,
			check: func(t *testing.T, r *html.Result, err error) {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if strings.Contains(r.Text, "Comment") {
					t.Error("Comments should be removed")
				}
			},
		},
		{
			name:    "very long text",
			html:    `<html><body><p>` + strings.Repeat("word ", 10000) + `</p></body></html>`,
			wantErr: false,
			check: func(t *testing.T, r *html.Result, err error) {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if r.WordCount == 0 {
					t.Error("WordCount should be > 0")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Extract(tt.html, html.DefaultExtractConfig())
			if (err != nil) != tt.wantErr {
				t.Errorf("Error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.check != nil {
				tt.check(t, result, err)
			}
		})
	}
}

// ============================================================================
// INTEGRATION TESTS
// ============================================================================

func TestIntegrationScenarios(t *testing.T) {
	t.Parallel()

	t.Run("blog post workflow", func(t *testing.T) {
		p := html.NewWithDefaults()
		defer p.Close()

		htmlContent := `<!DOCTYPE html>
<html>
<head><title>My Blog Post</title></head>
<body>
	<nav><a href="/">Home</a><a href="/about">About</a></nav>
	<article>
		<h1>Blog Post Title</h1>
		<img src="featured.jpg" alt="Featured Image">
		<p>First paragraph.</p>
		<p>Second paragraph with <a href="https://example.com">link</a>.</p>
	</article>
	<aside><h3>Related</h3></aside>
</body>
</html>`

		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if result.Title != "My Blog Post" {
			t.Errorf("Title = %q, want 'My Blog Post'", result.Title)
		}
		if !strings.Contains(result.Text, "First paragraph") {
			t.Error("Should extract main content")
		}
		if len(result.Images) == 0 {
			t.Error("Should extract images")
		}
		if len(result.Links) == 0 {
			t.Error("Should extract links")
		}
	})

	t.Run("news article with media", func(t *testing.T) {
		p := html.NewWithDefaults()
		defer p.Close()

		htmlContent := `
		<html><body>
			<article>
				<h1>Breaking News</h1>
				<video src="news.mp4" poster="thumb.jpg"></video>
				<p>News content.</p>
				<img src="photo.jpg">
				<audio src="interview.mp3"></audio>
			</article>
		</body></html>
		`

		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if len(result.Videos) == 0 {
			t.Error("Should extract videos")
		}
		if len(result.Audios) == 0 {
			t.Error("Should extract audios")
		}
		if len(result.Images) == 0 {
			t.Error("Should extract images")
		}
	})

	t.Run("documentation page with code and tables", func(t *testing.T) {
		p := html.NewWithDefaults()
		defer p.Close()

		htmlContent := `
		<html><body>
			<main>
				<h1>API Documentation</h1>
				<pre><code>npm install package</code></pre>
				<table>
					<tr><th>Method</th><th>Description</th></tr>
					<tr><td>init()</td><td>Initialize</td></tr>
				</table>
			</main>
		</body></html>
		`

		result, err := p.ExtractWithDefaults(htmlContent)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "npm install") {
			t.Error("Should extract code blocks")
		}
		if !strings.Contains(result.Text, "init()") {
			t.Error("Should extract table content")
		}
	})
}
