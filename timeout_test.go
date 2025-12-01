package html_test

import (
	"strings"
	"testing"
	"time"

	"github.com/cybergodev/html"
)

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

func TestDefaultConfigHasTimeout(t *testing.T) {
	t.Parallel()

	config := html.DefaultConfig()
	if config.ProcessingTimeout <= 0 {
		t.Errorf("DefaultConfig() ProcessingTimeout = %v, want positive value", config.ProcessingTimeout)
	}

	// Verify default is 30 seconds
	if config.ProcessingTimeout != 30*time.Second {
		t.Errorf("DefaultConfig() ProcessingTimeout = %v, want %v", config.ProcessingTimeout, 30*time.Second)
	}
}
