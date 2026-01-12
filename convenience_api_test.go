package html_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cybergodev/html"
)

// TestExtract tests package-level Extract function
func TestExtract(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html>
			<head><title>Test Page</title></head>
			<body>
				<article>
					<h1>Main Content</h1>
					<p>This is the main content of the page.</p>
				</article>
			</body>
		</html>
	`

	result, err := html.Extract(htmlContent)
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	if result.Text == "" {
		t.Error("Extract() returned empty text")
	}

	if result.Title != "Test Page" {
		t.Errorf("Extract() title = %q, want %q", result.Title, "Test Page")
	}
}

// TestExtractText tests package-level ExtractText function
func TestExtractText(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html>
			<body>
				<article>
					<h1>Hello World</h1>
					<p>This is a test.</p>
				</article>
			</body>
		</html>
	`

	text, err := html.ExtractText(htmlContent)
	if err != nil {
		t.Fatalf("ExtractText() failed: %v", err)
	}

	if text == "" {
		t.Error("ExtractText() returned empty string")
	}

	if !strings.Contains(text, "Hello World") {
		t.Errorf("ExtractText() = %q, want to contain %q", text, "Hello World")
	}
}

// TestExtractWithTitle tests package-level ExtractWithTitle function
func TestExtractWithTitle(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html>
			<head><title>Test Title</title></head>
			<body>
				<p>Content here</p>
			</body>
		</html>
	`

	title, text, err := html.ExtractWithTitle(htmlContent)
	if err != nil {
		t.Fatalf("ExtractWithTitle() failed: %v", err)
	}

	if title != "Test Title" {
		t.Errorf("ExtractWithTitle() title = %q, want %q", title, "Test Title")
	}

	if text == "" {
		t.Error("ExtractWithTitle() returned empty text")
	}
}

// TestExtractTitle tests package-level ExtractTitle function
func TestExtractTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		html     string
		wantTitle string
	}{
		{
			name: "title tag present",
			html: `<html><head><title>My Title</title></head><body>Content</body></html>`,
			wantTitle: "My Title",
		},
		{
			name: "h1 tag as fallback",
			html: `<html><body><h1>H1 Title</h1><p>Content</p></body></html>`,
			wantTitle: "H1 Title",
		},
		{
			name: "h2 tag as fallback",
			html: `<html><body><h2>H2 Title</h2><p>Content</p></body></html>`,
			wantTitle: "H2 Title",
		},
		{
			name: "no title found",
			html: `<html><body><p>Content</p></body></html>`,
			wantTitle: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, err := html.ExtractTitle(tt.html)
			if err != nil {
				t.Fatalf("ExtractTitle() failed: %v", err)
			}
			if title != tt.wantTitle {
				t.Errorf("ExtractTitle() = %q, want %q", title, tt.wantTitle)
			}
		})
	}
}

// TestExtractImages tests package-level ExtractImages function
func TestExtractImages(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html>
			<body>
				<img src="https://example.com/image1.jpg" alt="Image 1">
				<img src="https://example.com/image2.png" alt="Image 2">
			</body>
		</html>
	`

	images, err := html.ExtractImages(htmlContent)
	if err != nil {
		t.Fatalf("ExtractImages() failed: %v", err)
	}

	if len(images) != 2 {
		t.Errorf("ExtractImages() returned %d images, want 2", len(images))
	}

	if images[0].URL != "https://example.com/image1.jpg" {
		t.Errorf("ExtractImages()[0].URL = %q, want %q", images[0].URL, "https://example.com/image1.jpg")
	}
}

// TestExtractVideos tests package-level ExtractVideos function
func TestExtractVideos(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html>
			<body>
				<video src="https://example.com/video.mp4" width="640" height="480"></video>
				<iframe src="https://www.youtube.com/embed/test123"></iframe>
			</body>
		</html>
	`

	videos, err := html.ExtractVideos(htmlContent)
	if err != nil {
		t.Fatalf("ExtractVideos() failed: %v", err)
	}

	if len(videos) == 0 {
		t.Error("ExtractVideos() returned no videos")
	}
}

// TestExtractAudios tests package-level ExtractAudios function
func TestExtractAudios(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html>
			<body>
				<audio src="https://example.com/audio.mp3"></audio>
			</body>
		</html>
	`

	audios, err := html.ExtractAudios(htmlContent)
	if err != nil {
		t.Fatalf("ExtractAudios() failed: %v", err)
	}

	if len(audios) == 0 {
		t.Error("ExtractAudios() returned no audio files")
	}

	if audios[0].URL != "https://example.com/audio.mp3" {
		t.Errorf("ExtractAudios()[0].URL = %q, want %q", audios[0].URL, "https://example.com/audio.mp3")
	}
}

// TestExtractLinks tests package-level ExtractLinks function
func TestExtractLinks(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html>
			<body>
				<a href="https://example.com">Example Site</a>
				<a href="/internal">Internal Link</a>
			</body>
		</html>
	`

	links, err := html.ExtractLinks(htmlContent)
	if err != nil {
		t.Fatalf("ExtractLinks() failed: %v", err)
	}

	if len(links) == 0 {
		t.Error("ExtractLinks() returned no links")
	}
}

// TestGetWordCount tests package-level GetWordCount function
func TestGetWordCount(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html>
			<body>
				<p>This is a test with several words.</p>
			</body>
		</html>
	`

	count, err := html.GetWordCount(htmlContent)
	if err != nil {
		t.Fatalf("GetWordCount() failed: %v", err)
	}

	if count == 0 {
		t.Error("GetWordCount() returned 0, want > 0")
	}
}

// TestGetReadingTime tests package-level GetReadingTime function
func TestGetReadingTime(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html>
			<body>
				<p>This is a test content with some words to calculate reading time.</p>
			</body>
		</html>
	`

	minutes, err := html.GetReadingTime(htmlContent)
	if err != nil {
		t.Fatalf("GetReadingTime() failed: %v", err)
	}

	if minutes <= 0 {
		t.Errorf("GetReadingTime() = %f, want > 0", minutes)
	}
}

// TestSummarize tests package-level Summarize function
func TestSummarize(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html>
			<body>
				<p>Word1 word2 word3 word4 word5 word6 word7 word8 word9 word10 word11 word12.</p>
			</body>
		</html>
	`

	t.Run("Summarize with word limit", func(t *testing.T) {
		summary, err := html.Summarize(htmlContent, 5)
		if err != nil {
			t.Fatalf("Summarize() failed: %v", err)
		}

		words := strings.Fields(summary)
		if len(words) > 5 {
			t.Errorf("Summarize() returned %d words, want <= 5", len(words))
		}
	})

	t.Run("Summarize with zero limit returns full text", func(t *testing.T) {
		summary, err := html.Summarize(htmlContent, 0)
		if err != nil {
			t.Fatalf("Summarize() failed: %v", err)
		}

		if summary == "" {
			t.Error("Summarize() returned empty string")
		}
	})
}

// TestExtractAndClean tests package-level ExtractAndClean function
func TestExtractAndClean(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html>
			<body>
				<p>  This   has   extra   spaces.  </p>
				<p>Another line.</p>
			</body>
		</html>
	`

	cleaned, err := html.ExtractAndClean(htmlContent)
	if err != nil {
		t.Fatalf("ExtractAndClean() failed: %v", err)
	}

	if cleaned == "" {
		t.Error("ExtractAndClean() returned empty string")
	}

	// Check that excessive whitespace is removed
	if strings.Contains(cleaned, "   ") {
		t.Error("ExtractAndClean() should remove excessive whitespace")
	}
}

// TestExtractToMarkdown tests package-level ExtractToMarkdown function
func TestExtractToMarkdown(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html>
			<body>
				<h1>Title</h1>
				<p>Content here</p>
				<img src="https://example.com/image.jpg" alt="Test Image">
			</body>
		</html>
	`

	markdown, err := html.ExtractToMarkdown(htmlContent)
	if err != nil {
		t.Fatalf("ExtractToMarkdown() failed: %v", err)
	}

	if markdown == "" {
		t.Error("ExtractToMarkdown() returned empty string")
	}
}

// TestExtractToJSON tests package-level ExtractToJSON function
func TestExtractToJSON(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html>
			<head><title>Test</title></head>
			<body>
				<p>Content here</p>
			</body>
		</html>
	`

	jsonData, err := html.ExtractToJSON(htmlContent)
	if err != nil {
		t.Fatalf("ExtractToJSON() failed: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("ExtractToJSON() returned empty data")
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Errorf("ExtractToJSON() returned invalid JSON: %v", err)
	}

	// Check for expected fields
	if _, ok := result["title"]; !ok {
		t.Error("ExtractToJSON() missing 'title' field")
	}
	if _, ok := result["text"]; !ok {
		t.Error("ExtractToJSON() missing 'text' field")
	}
}

// TestConfigPresets tests configuration preset functions
func TestConfigPresets(t *testing.T) {
	t.Parallel()

	t.Run("ConfigForRSS", func(t *testing.T) {
		config := html.ConfigForRSS()
		if config.ExtractArticle {
			t.Error("ConfigForRSS() should disable article detection")
		}
		if !config.PreserveImages {
			t.Error("ConfigForRSS() should preserve images")
		}
		if !config.PreserveLinks {
			t.Error("ConfigForRSS() should preserve links")
		}
	})

	t.Run("ConfigForSearchIndex", func(t *testing.T) {
		config := html.ConfigForSearchIndex()
		if !config.ExtractArticle {
			t.Error("ConfigForSearchIndex() should enable article detection")
		}
		if !config.PreserveImages {
			t.Error("ConfigForSearchIndex() should preserve images")
		}
		if !config.PreserveVideos {
			t.Error("ConfigForSearchIndex() should preserve videos")
		}
	})

	t.Run("ConfigForSummary", func(t *testing.T) {
		config := html.ConfigForSummary()
		if !config.ExtractArticle {
			t.Error("ConfigForSummary() should enable article detection")
		}
		if config.PreserveImages {
			t.Error("ConfigForSummary() should not preserve images")
		}
		if config.PreserveLinks {
			t.Error("ConfigForSummary() should not preserve links")
		}
	})

	t.Run("ConfigForMarkdown", func(t *testing.T) {
		config := html.ConfigForMarkdown()
		if config.InlineImageFormat != "markdown" {
			t.Errorf("ConfigForMarkdown() InlineImageFormat = %q, want 'markdown'", config.InlineImageFormat)
		}
		if !config.PreserveImages {
			t.Error("ConfigForMarkdown() should preserve images")
		}
	})
}

// BenchmarkExtractText benchmarks package-level ExtractText function
func BenchmarkExtractText(b *testing.B) {
	htmlContent := `<html><body><p>Test content</p></body></html>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = html.ExtractText(htmlContent)
	}
}

// BenchmarkExtractToJSON benchmarks package-level ExtractToJSON function
func BenchmarkExtractToJSON(b *testing.B) {
	htmlContent := `<html><body><p>Test content</p></body></html>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = html.ExtractToJSON(htmlContent)
	}
}

// BenchmarkExtractToMarkdown benchmarks package-level ExtractToMarkdown function
func BenchmarkExtractToMarkdown(b *testing.B) {
	htmlContent := `<html><body><p>Test content</p></body></html>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = html.ExtractToMarkdown(htmlContent)
	}
}
