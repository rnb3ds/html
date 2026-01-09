package html_test

import (
	"strings"
	"testing"
	"time"

	"github.com/cybergodev/html"
)

// TestProcessorLifecycle tests processor creation, usage, and cleanup
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

	t.Run("Close once", func(t *testing.T) {
		p := html.NewWithDefaults()
		err := p.Close()
		if err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
	})

	t.Run("Close twice (idempotent)", func(t *testing.T) {
		p := html.NewWithDefaults()
		p.Close()
		err := p.Close()
		if err != nil {
			t.Fatalf("Close() should be idempotent: %v", err)
		}
	})

	t.Run("Extract after close", func(t *testing.T) {
		p := html.NewWithDefaults()
		p.Close()

		_, err := p.ExtractWithDefaults("<html><body>Test</body></html>")
		if err == nil {
			t.Fatal("Extract() should fail after Close()")
		}
	})
}

// TestBasicExtraction tests basic extraction functionality
func TestBasicExtraction(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	t.Run("simple HTML", func(t *testing.T) {
		htmlContent := `<html><body><p>Hello World</p></body></html>`
		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "Hello World") {
			t.Errorf("Extract() text = %q, want to contain %q", result.Text, "Hello World")
		}
	})

	t.Run("empty HTML", func(t *testing.T) {
		result, err := p.Extract("", html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed on empty input: %v", err)
		}
		if result.Text != "" {
			t.Errorf("Extract() text = %q, want empty", result.Text)
		}
	})

	t.Run("whitespace only", func(t *testing.T) {
		result, err := p.Extract("   \n\t  ", html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed on whitespace: %v", err)
		}
		if result.Text != "" {
			t.Errorf("Extract() text = %q, want empty", result.Text)
		}
	})

	t.Run("malformed HTML", func(t *testing.T) {
		htmlContent := `<html><body><p>Unclosed paragraph`
		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() should handle malformed HTML: %v", err)
		}
		if !strings.Contains(result.Text, "Unclosed paragraph") {
			t.Errorf("Extract() should extract text from malformed HTML")
		}
	})
}

// TestExtractWithDefaults tests the convenience method
func TestExtractWithDefaults(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><article><h1>Title</h1><p>Content</p></article></body></html>`
	result, err := p.ExtractWithDefaults(htmlContent)
	if err != nil {
		t.Fatalf("ExtractWithDefaults() failed: %v", err)
	}

	if result.Title != "Title" {
		t.Errorf("ExtractWithDefaults() title = %q, want %q", result.Title, "Title")
	}
	if !strings.Contains(result.Text, "Content") {
		t.Errorf("ExtractWithDefaults() text should contain %q", "Content")
	}
}

// TestExtractFromFile tests file-based extraction
func TestExtractFromFile(t *testing.T) {
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
}

// TestConfiguration tests configuration validation and defaults
func TestConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("DefaultConfig", func(t *testing.T) {
		config := html.DefaultConfig()

		if config.MaxInputSize <= 0 {
			t.Error("DefaultConfig() MaxInputSize should be positive")
		}
		if config.MaxCacheEntries < 0 {
			t.Error("DefaultConfig() MaxCacheEntries should be non-negative")
		}
		if config.CacheTTL < 0 {
			t.Error("DefaultConfig() CacheTTL should be non-negative")
		}
		if config.WorkerPoolSize <= 0 {
			t.Error("DefaultConfig() WorkerPoolSize should be positive")
		}
		if !config.EnableSanitization {
			t.Error("DefaultConfig() should enable sanitization by default")
		}
		if config.MaxDepth <= 0 {
			t.Error("DefaultConfig() MaxDepth should be positive")
		}
		if config.ProcessingTimeout != 30*time.Second {
			t.Errorf("DefaultConfig() ProcessingTimeout = %v, want %v", config.ProcessingTimeout, 30*time.Second)
		}
	})

	t.Run("DefaultExtractConfig", func(t *testing.T) {
		config := html.DefaultExtractConfig()

		if !config.ExtractArticle {
			t.Error("DefaultExtractConfig() should enable article extraction")
		}
		if !config.PreserveImages {
			t.Error("DefaultExtractConfig() should preserve images")
		}
		if !config.PreserveLinks {
			t.Error("DefaultExtractConfig() should preserve links")
		}
		if !config.PreserveVideos {
			t.Error("DefaultExtractConfig() should preserve videos")
		}
		if !config.PreserveAudios {
			t.Error("DefaultExtractConfig() should preserve audios")
		}
		if config.InlineImageFormat != "none" {
			t.Errorf("DefaultExtractConfig() InlineImageFormat = %q, want %q", config.InlineImageFormat, "none")
		}
	})

	t.Run("invalid config - negative max input size", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxInputSize = -1
		_, err := html.New(config)
		if err == nil {
			t.Fatal("New() should fail with negative MaxInputSize")
		}
	})

	t.Run("invalid config - zero worker pool size", func(t *testing.T) {
		config := html.DefaultConfig()
		config.WorkerPoolSize = 0
		_, err := html.New(config)
		if err == nil {
			t.Fatal("New() should fail with zero WorkerPoolSize")
		}
	})

	t.Run("invalid config - zero max depth", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxDepth = 0
		_, err := html.New(config)
		if err == nil {
			t.Fatal("New() should fail with zero MaxDepth")
		}
	})

	t.Run("custom config", func(t *testing.T) {
		config := html.Config{
			MaxInputSize:       1024 * 1024,
			MaxCacheEntries:    50,
			CacheTTL:           30 * time.Minute,
			WorkerPoolSize:     8,
			EnableSanitization: false,
			MaxDepth:           50,
		}

		p, err := html.New(config)
		if err != nil {
			t.Fatalf("New() with custom config failed: %v", err)
		}
		defer p.Close()

		htmlContent := `<html><body><p>Test</p></body></html>`
		_, err = p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() with custom config failed: %v", err)
		}
	})
}

// TestInputSizeLimit tests input size validation
func TestInputSizeLimit(t *testing.T) {
	t.Parallel()

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
}

// TestMaxDepthExceeded tests depth limit validation
func TestMaxDepthExceeded(t *testing.T) {
	t.Parallel()

	config := html.Config{
		MaxInputSize:       1024 * 1024,
		MaxCacheEntries:    100,
		CacheTTL:           time.Hour,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           5,
	}
	p, _ := html.New(config)
	defer p.Close()

	// Create deeply nested HTML
	deepHTML := "<div>" + strings.Repeat("<div>", 10) + "content" + strings.Repeat("</div>", 10) + "</div>"

	_, err := p.Extract(deepHTML, html.DefaultExtractConfig())
	if err == nil {
		t.Fatal("Extract() should fail with max depth exceeded")
	}
}

// TestProcessingTimeout tests timeout functionality
func TestProcessingTimeout(t *testing.T) {
	t.Parallel()

	// Create deeply nested HTML that takes time to process
	var sb strings.Builder
	depth := 500
	for i := 0; i < depth; i++ {
		sb.WriteString("<div>")
	}
	sb.WriteString("Content")
	for i := 0; i < depth; i++ {
		sb.WriteString("</div>")
	}

	htmlContent := sb.String()

	// Test with very short timeout
	config := html.Config{
		MaxInputSize:       10 * 1024 * 1024,
		MaxCacheEntries:    100,
		CacheTTL:           time.Hour,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           1000,
		ProcessingTimeout:  1 * time.Nanosecond, // Extremely short timeout
	}

	p, err := html.New(config)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()

	_, err = p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != html.ErrProcessingTimeout {
		t.Errorf("Expected ErrProcessingTimeout, got: %v", err)
	}
}

// TestProcessingWithoutTimeout tests disabled timeout
func TestProcessingWithoutTimeout(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html>
		<body>
			<article>
				<h1>Test Article</h1>
				<p>This is a test article with some content.</p>
			</article>
		</body>
		</html>
	`

	// Test with no timeout (0 means disabled)
	config := html.Config{
		MaxInputSize:       10 * 1024 * 1024,
		MaxCacheEntries:    100,
		CacheTTL:           time.Hour,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
		ProcessingTimeout:  0, // No timeout
	}

	p, err := html.New(config)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()

	result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	if result.Title != "Test Article" {
		t.Errorf("Title = %q, want %q", result.Title, "Test Article")
	}
}

// TestProcessingWithReasonableTimeout tests normal timeout
func TestProcessingWithReasonableTimeout(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html>
		<body>
			<article>
				<h1>Test Article</h1>
				<p>This is a test article with some content.</p>
				<img src="test.jpg" alt="Test Image">
				<a href="link.html">Test Link</a>
			</article>
		</body>
		</html>
	`

	// Test with reasonable timeout
	config := html.Config{
		MaxInputSize:       10 * 1024 * 1024,
		MaxCacheEntries:    100,
		CacheTTL:           time.Hour,
		WorkerPoolSize:     4,
		EnableSanitization: true,
		MaxDepth:           100,
		ProcessingTimeout:  5 * time.Second, // Reasonable timeout
	}

	p, err := html.New(config)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()

	result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	if result.Title != "Test Article" {
		t.Errorf("Title = %q, want %q", result.Title, "Test Article")
	}

	if len(result.Images) != 1 {
		t.Errorf("Images count = %d, want 1", len(result.Images))
	}

	if len(result.Links) != 1 {
		t.Errorf("Links count = %d, want 1", len(result.Links))
	}
}

// TestStatistics tests statistics tracking
func TestStatistics(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	stats := p.GetStatistics()
	if stats.TotalProcessed != 0 {
		t.Errorf("GetStatistics() initial TotalProcessed = %d, want 0", stats.TotalProcessed)
	}

	htmlContent := `<html><body><p>Test</p></body></html>`
	p.Extract(htmlContent, html.DefaultExtractConfig())

	stats = p.GetStatistics()
	if stats.TotalProcessed != 1 {
		t.Errorf("GetStatistics() TotalProcessed = %d, want 1", stats.TotalProcessed)
	}

	// Extract same content again (should hit cache)
	p.Extract(htmlContent, html.DefaultExtractConfig())

	stats = p.GetStatistics()
	if stats.TotalProcessed != 2 {
		t.Errorf("GetStatistics() TotalProcessed = %d, want 2", stats.TotalProcessed)
	}
	if stats.CacheHits != 1 {
		t.Errorf("GetStatistics() CacheHits = %d, want 1", stats.CacheHits)
	}
}

