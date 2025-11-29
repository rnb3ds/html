package html_test

import (
	"strings"
	"testing"
	"time"

	"github.com/cybergodev/html"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("valid config", func(t *testing.T) {
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

	t.Run("invalid config - negative max input size", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxInputSize = -1
		_, err := html.New(config)
		if err == nil {
			t.Fatal("New() should fail with negative MaxInputSize")
		}
	})

	t.Run("invalid config - zero timeout", func(t *testing.T) {
		config := html.DefaultConfig()
		config.ProcessingTimeout = 0
		_, err := html.New(config)
		if err == nil {
			t.Fatal("New() should fail with zero ProcessingTimeout")
		}
	})

	t.Run("invalid config - negative cache entries", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxCacheEntries = -1
		_, err := html.New(config)
		if err == nil {
			t.Fatal("New() should fail with negative MaxCacheEntries")
		}
	})

	t.Run("invalid config - negative cache TTL", func(t *testing.T) {
		config := html.DefaultConfig()
		config.CacheTTL = -1
		_, err := html.New(config)
		if err == nil {
			t.Fatal("New() should fail with negative CacheTTL")
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
}

func TestNewWithDefaults(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	if p == nil {
		t.Fatal("NewWithDefaults() returned nil")
	}
	defer p.Close()
}

func TestExtract(t *testing.T) {
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

	t.Run("input too large", func(t *testing.T) {
		config := html.Config{
			MaxInputSize:       100,
			ProcessingTimeout:  30 * time.Second,
			MaxCacheEntries:    10,
			CacheTTL:           time.Hour,
			WorkerPoolSize:     4,
			EnableSanitization: true,
			MaxDepth:           100,
		}
		p2, _ := html.New(config)
		defer p2.Close()

		largeHTML := strings.Repeat("a", 200)
		_, err := p2.Extract(largeHTML, html.DefaultExtractConfig())
		if err == nil {
			t.Fatal("Extract() should fail with input too large")
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

func TestClose(t *testing.T) {
	t.Parallel()

	t.Run("close once", func(t *testing.T) {
		p := html.NewWithDefaults()
		err := p.Close()
		if err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
	})

	t.Run("close twice", func(t *testing.T) {
		p := html.NewWithDefaults()
		p.Close()
		err := p.Close()
		if err != nil {
			t.Fatalf("Close() should be idempotent: %v", err)
		}
	})

	t.Run("extract after close", func(t *testing.T) {
		p := html.NewWithDefaults()
		p.Close()

		_, err := p.ExtractWithDefaults("<html><body>Test</body></html>")
		if err == nil {
			t.Fatal("Extract() should fail after Close()")
		}
	})
}

func TestGetStatistics(t *testing.T) {
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

func TestClearCache(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><p>Test</p></body></html>`
	p.Extract(htmlContent, html.DefaultExtractConfig())

	stats := p.GetStatistics()
	if stats.CacheMisses != 1 {
		t.Errorf("GetStatistics() CacheMisses = %d, want 1", stats.CacheMisses)
	}

	p.ClearCache()

	stats = p.GetStatistics()
	if stats.CacheHits != 0 || stats.CacheMisses != 0 {
		t.Errorf("ClearCache() should reset cache stats")
	}

	// Extract again after cache clear
	p.Extract(htmlContent, html.DefaultExtractConfig())

	stats = p.GetStatistics()
	if stats.CacheMisses != 1 {
		t.Errorf("GetStatistics() CacheMisses = %d, want 1 after cache clear", stats.CacheMisses)
	}
}

func TestMaxDepthExceeded(t *testing.T) {
	t.Parallel()

	config := html.Config{
		MaxInputSize:       1024 * 1024,
		ProcessingTimeout:  30 * time.Second,
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

func TestConcurrentExtract(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><p>Concurrent test</p></body></html>`

	const goroutines = 10
	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
			if err != nil {
				t.Errorf("Concurrent Extract() failed: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}

	stats := p.GetStatistics()
	if stats.TotalProcessed != goroutines {
		t.Errorf("Concurrent processing: TotalProcessed = %d, want %d", stats.TotalProcessed, goroutines)
	}
}
