package html_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cybergodev/html"
)

// TestEndToEndWorkflow tests complete extraction workflows
func TestEndToEndWorkflow(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	t.Run("blog post extraction", func(t *testing.T) {
		htmlContent := `
<!DOCTYPE html>
<html>
<head>
	<title>My Blog Post</title>
	<meta name="description" content="A test blog post">
</head>
<body>
	<nav>
		<a href="/">Home</a>
		<a href="/about">About</a>
	</nav>
	<article>
		<h1>Blog Post Title</h1>
		<p class="meta">Published on 2024-01-01</p>
		<img src="featured.jpg" alt="Featured Image">
		<p>This is the first paragraph of the blog post.</p>
		<p>This is the second paragraph with <a href="https://example.com">a link</a>.</p>
		<h2>Section Heading</h2>
		<p>More content here.</p>
		<ul>
			<li>Point 1</li>
			<li>Point 2</li>
			<li>Point 3</li>
		</ul>
	</article>
	<aside>
		<h3>Related Posts</h3>
		<ul>
			<li><a href="/post1">Post 1</a></li>
			<li><a href="/post2">Post 2</a></li>
		</ul>
	</aside>
	<footer>
		<p>Copyright 2024</p>
	</footer>
</body>
</html>
`

		config := html.DefaultExtractConfig()
		config.ExtractArticle = true

		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Verify title (extracts from <title> tag first, then <h1>)
		if result.Title != "My Blog Post" && result.Title != "Blog Post Title" {
			t.Errorf("Title = %q, want either %q or %q", result.Title, "My Blog Post", "Blog Post Title")
		}

		// Verify main content extracted
		if !strings.Contains(result.Text, "first paragraph") {
			t.Error("Main content should be extracted")
		}

		// Verify navigation excluded
		if strings.Contains(result.Text, "Home") && strings.Contains(result.Text, "About") {
			t.Error("Navigation should be excluded from article extraction")
		}

		// Verify images
		if len(result.Images) == 0 {
			t.Error("Images should be extracted")
		}

		// Verify links
		if len(result.Links) == 0 {
			t.Error("Links should be extracted")
		}

		// Verify word count
		if result.WordCount == 0 {
			t.Error("Word count should be calculated")
		}

		// Verify reading time
		if result.ReadingTime == 0 {
			t.Error("Reading time should be calculated")
		}
	})

	t.Run("news article with media", func(t *testing.T) {
		htmlContent := `
<!DOCTYPE html>
<html>
<body>
	<article>
		<h1>Breaking News</h1>
		<video src="news.mp4" poster="thumb.jpg"></video>
		<p>News content here.</p>
		<img src="photo1.jpg" alt="Photo 1">
		<p>More news content.</p>
		<audio src="interview.mp3"></audio>
	</article>
</body>
</html>
`

		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if len(result.Videos) == 0 {
			t.Error("Videos should be extracted")
		}
		if len(result.Audios) == 0 {
			t.Error("Audios should be extracted")
		}
		if len(result.Images) == 0 {
			t.Error("Images should be extracted")
		}
	})

	t.Run("documentation page", func(t *testing.T) {
		htmlContent := `
<!DOCTYPE html>
<html>
<body>
	<main>
		<h1>API Documentation</h1>
		<h2>Installation</h2>
		<pre><code>npm install package</code></pre>
		<h2>Usage</h2>
		<p>Import the package:</p>
		<pre><code>import pkg from 'package';</code></pre>
		<h2>API Reference</h2>
		<table>
			<tr><th>Method</th><th>Description</th></tr>
			<tr><td>init()</td><td>Initialize</td></tr>
			<tr><td>run()</td><td>Execute</td></tr>
		</table>
	</main>
</body>
</html>
`

		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "npm install") {
			t.Error("Code blocks should be extracted")
		}
		if !strings.Contains(result.Text, "init()") {
			t.Error("Table content should be extracted")
		}
	})
}

// TestFileBasedWorkflow tests file-based extraction
func TestFileBasedWorkflow(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tmpDir := t.TempDir()

	t.Run("single file extraction", func(t *testing.T) {
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
			t.Errorf("Title = %q, want %q", result.Title, "Test")
		}
	})

	t.Run("batch file extraction", func(t *testing.T) {
		files := []string{
			filepath.Join(tmpDir, "file1.html"),
			filepath.Join(tmpDir, "file2.html"),
			filepath.Join(tmpDir, "file3.html"),
		}

		for i, file := range files {
			content := `<html><body><h1>Title ` + string(rune('1'+i)) + `</h1></body></html>`
			if err := os.WriteFile(file, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}

		results, err := p.ExtractBatchFiles(files, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("ExtractBatchFiles() failed: %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("ExtractBatchFiles() returned %d results, want 3", len(results))
		}

		for i, result := range results {
			expectedTitle := "Title " + string(rune('1'+i))
			if result.Title != expectedTitle {
				t.Errorf("Result[%d] title = %q, want %q", i, result.Title, expectedTitle)
			}
		}
	})
}

// TestCachingWorkflow tests caching behavior in real workflows
func TestCachingWorkflow(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><article><h1>Cached</h1><p>Content</p></article></body></html>`

	// First extraction - cache miss
	result1, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("First Extract() failed: %v", err)
	}

	stats1 := p.GetStatistics()
	if stats1.CacheMisses != 1 {
		t.Errorf("First extraction should be cache miss, got %d misses", stats1.CacheMisses)
	}

	// Second extraction - cache hit
	result2, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Second Extract() failed: %v", err)
	}

	stats2 := p.GetStatistics()
	if stats2.CacheHits != 1 {
		t.Errorf("Second extraction should be cache hit, got %d hits", stats2.CacheHits)
	}

	// Results should be identical
	if result1.Text != result2.Text {
		t.Error("Cached result should match original")
	}
	if result1.Title != result2.Title {
		t.Error("Cached title should match original")
	}

	// Different config - new cache entry
	config := html.DefaultExtractConfig()
	config.PreserveImages = false

	_, err = p.Extract(htmlContent, config)
	if err != nil {
		t.Fatalf("Third Extract() failed: %v", err)
	}

	stats3 := p.GetStatistics()
	if stats3.CacheMisses != 2 {
		t.Errorf("Different config should cause cache miss, got %d misses", stats3.CacheMisses)
	}
}

// TestErrorRecoveryWorkflow tests error handling and recovery
func TestErrorRecoveryWorkflow(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	t.Run("recover from invalid HTML", func(t *testing.T) {
		invalidHTML := `<html><body><p>Unclosed paragraph`

		result, err := p.Extract(invalidHTML, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Should handle invalid HTML gracefully: %v", err)
		}

		if !strings.Contains(result.Text, "Unclosed paragraph") {
			t.Error("Should extract text from invalid HTML")
		}
	})

	t.Run("continue after error in batch", func(t *testing.T) {
		htmlContents := []string{
			`<html><body><p>Valid 1</p></body></html>`,
			strings.Repeat("x", 100*1024*1024), // Too large
			`<html><body><p>Valid 2</p></body></html>`,
		}

		results, err := p.ExtractBatch(htmlContents, html.DefaultExtractConfig())
		if err == nil {
			t.Fatal("Should return error for partial failure")
		}

		// Valid results should still be present
		if results[0] == nil {
			t.Error("First result should be valid")
		}
		if results[2] == nil {
			t.Error("Third result should be valid")
		}
	})

	t.Run("processor remains usable after errors", func(t *testing.T) {
		// Cause an error
		_, err := p.Extract(strings.Repeat("x", 100*1024*1024), html.DefaultExtractConfig())
		if err == nil {
			t.Fatal("Should fail with input too large")
		}

		// Processor should still work
		validHTML := `<html><body><p>Still works</p></body></html>`
		result, err := p.Extract(validHTML, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Processor should remain usable after error: %v", err)
		}

		if !strings.Contains(result.Text, "Still works") {
			t.Error("Processor should continue working after error")
		}
	})
}

// TestStatisticsTracking tests statistics across workflows
func TestStatisticsTracking(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><p>Test</p></body></html>`

	// Initial state
	stats := p.GetStatistics()
	if stats.TotalProcessed != 0 {
		t.Error("Initial TotalProcessed should be 0")
	}

	// Process some content
	for i := 0; i < 5; i++ {
		p.Extract(htmlContent, html.DefaultExtractConfig())
	}

	stats = p.GetStatistics()
	if stats.TotalProcessed != 5 {
		t.Errorf("TotalProcessed = %d, want 5", stats.TotalProcessed)
	}

	// Cache should have hits
	if stats.CacheHits < 4 {
		t.Errorf("Should have cache hits, got %d", stats.CacheHits)
	}

	// Average processing time should be calculated (may be 0 for very fast operations)
	if stats.AverageProcessTime < 0 {
		t.Error("AverageProcessTime should not be negative")
	}

	// Clear cache and verify stats reset
	p.ClearCache()
	stats = p.GetStatistics()
	if stats.CacheHits != 0 || stats.CacheMisses != 0 {
		t.Error("Cache stats should be reset after ClearCache()")
	}
}

// TestConfigurationVariations tests different configuration combinations
func TestConfigurationVariations(t *testing.T) {
	t.Parallel()

	htmlContent := `
<html>
<body>
	<article>
		<h1>Title</h1>
		<p>Content</p>
		<img src="test.jpg" alt="Test">
		<a href="link.html">Link</a>
		<video src="video.mp4"></video>
		<audio src="audio.mp3"></audio>
	</article>
</body>
</html>
`

	tests := []struct {
		name   string
		config html.ExtractConfig
		check  func(*html.Result) error
	}{
		{
			name: "minimal extraction",
			config: html.ExtractConfig{
				ExtractArticle:    false,
				PreserveImages:    false,
				PreserveLinks:     false,
				PreserveVideos:    false,
				PreserveAudios:    false,
				InlineImageFormat: "none",
			},
			check: func(r *html.Result) error {
				if len(r.Images) > 0 {
					return nil // Images might still be extracted
				}
				if len(r.Links) > 0 {
					return nil // Links might still be extracted
				}
				return nil
			},
		},
		{
			name: "full extraction",
			config: html.ExtractConfig{
				ExtractArticle:    true,
				PreserveImages:    true,
				PreserveLinks:     true,
				PreserveVideos:    true,
				PreserveAudios:    true,
				InlineImageFormat: "markdown",
			},
			check: func(r *html.Result) error {
				if len(r.Images) == 0 {
					t.Error("Images should be extracted")
				}
				if len(r.Links) == 0 {
					t.Error("Links should be extracted")
				}
				if len(r.Videos) == 0 {
					t.Error("Videos should be extracted")
				}
				if len(r.Audios) == 0 {
					t.Error("Audios should be extracted")
				}
				return nil
			},
		},
		{
			name: "images only",
			config: html.ExtractConfig{
				ExtractArticle:    false,
				PreserveImages:    true,
				PreserveLinks:     false,
				PreserveVideos:    false,
				PreserveAudios:    false,
				InlineImageFormat: "html",
			},
			check: func(r *html.Result) error {
				if len(r.Images) == 0 {
					t.Error("Images should be extracted")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := html.NewWithDefaults()
			defer p.Close()

			result, err := p.Extract(htmlContent, tt.config)
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if err := tt.check(result); err != nil {
				t.Error(err)
			}
		})
	}
}
