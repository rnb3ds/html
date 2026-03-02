package html_test

// output_test.go - Tests for output format functions
// This file tests ExtractToMarkdown, ExtractToJSON, and Result serialization.

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cybergodev/html"
)

func TestExtractToMarkdownOutput(t *testing.T) {
	t.Parallel()

	t.Run("basic markdown extraction", func(t *testing.T) {
		htmlContent := `<html><body><h1>Title</h1><p>Content paragraph.</p></body></html>`
		markdown, err := html.ExtractToMarkdown([]byte(htmlContent))
		if err != nil {
			t.Fatalf("ExtractToMarkdown() failed: %v", err)
		}
		if markdown == "" {
			t.Error("Markdown should not be empty")
		}
		if !strings.Contains(markdown, "Title") {
			t.Error("Markdown should contain title")
		}
		if !strings.Contains(markdown, "Content") {
			t.Error("Markdown should contain content")
		}
	})

	t.Run("markdown with images", func(t *testing.T) {
		htmlContent := `<html><body><img src="test.jpg" alt="Test Image"><p>Text</p></body></html>`
		markdown, err := html.ExtractToMarkdown([]byte(htmlContent))
		if err != nil {
			t.Fatalf("ExtractToMarkdown() failed: %v", err)
		}
		if !strings.Contains(markdown, "![Test Image](test.jpg)") {
			t.Errorf("Markdown should contain image markdown syntax, got: %s", markdown)
		}
	})

	t.Run("markdown with links", func(t *testing.T) {
		htmlContent := `<html><body><a href="https://example.com">Link</a></body></html>`
		markdown, err := html.ExtractToMarkdown([]byte(htmlContent))
		if err != nil {
			t.Fatalf("ExtractToMarkdown() failed: %v", err)
		}
		// Links are preserved in the links array, not as inline markdown
		if !strings.Contains(markdown, "Link") {
			t.Errorf("Markdown should contain link text, got: %s", markdown)
		}
	})

	t.Run("markdown with table", func(t *testing.T) {
		htmlContent := `<html><body>
			<table>
				<tr><th>Name</th><th>Value</th></tr>
				<tr><td>Test</td><td>100</td></tr>
			</table>
		</body></html>`
		markdown, err := html.ExtractToMarkdown([]byte(htmlContent))
		if err != nil {
			t.Fatalf("ExtractToMarkdown() failed: %v", err)
		}
		if !strings.Contains(markdown, "|") {
			t.Error("Markdown should contain table separators")
		}
		if !strings.Contains(markdown, "Name") || !strings.Contains(markdown, "Value") {
			t.Error("Markdown should contain table headers")
		}
	})

	t.Run("empty HTML", func(t *testing.T) {
		markdown, err := html.ExtractToMarkdown([]byte(""))
		if err != nil {
			t.Fatalf("ExtractToMarkdown() failed: %v", err)
		}
		if markdown != "" {
			t.Errorf("Empty HTML should produce empty markdown, got: %s", markdown)
		}
	})

	t.Run("complex document", func(t *testing.T) {
		htmlContent := `<!DOCTYPE html>
		<html>
		<head><title>Test Article</title></head>
		<body>
			<article>
				<h1>Main Heading</h1>
				<p>First paragraph with <strong>bold</strong> text.</p>
				<h2>Subheading</h2>
				<p>Second paragraph with <em>italic</em> text.</p>
				<ul>
					<li>Item 1</li>
					<li>Item 2</li>
				</ul>
			</article>
		</body>
		</html>`
		markdown, err := html.ExtractToMarkdown([]byte(htmlContent))
		if err != nil {
			t.Fatalf("ExtractToMarkdown() failed: %v", err)
		}
		if markdown == "" {
			t.Error("Markdown should not be empty")
		}
	})
}

func TestExtractToJSON(t *testing.T) {
	t.Parallel()

	t.Run("basic JSON extraction", func(t *testing.T) {
		htmlContent := `<html><head><title>Test Title</title></head><body><p>Test content.</p></body></html>`
		jsonData, err := html.ExtractToJSON([]byte(htmlContent))
		if err != nil {
			t.Fatalf("ExtractToJSON() failed: %v", err)
		}
		if len(jsonData) == 0 {
			t.Error("JSON data should not be empty")
		}

		var result map[string]interface{}
		if err := json.Unmarshal(jsonData, &result); err != nil {
			t.Fatalf("Invalid JSON: %v", err)
		}

		if title, ok := result["title"].(string); !ok || title != "Test Title" {
			t.Errorf("title = %v, want 'Test Title'", result["title"])
		}
	})

	t.Run("JSON with all fields", func(t *testing.T) {
		htmlContent := `<html>
		<head><title>Title</title></head>
		<body>
			<img src="img.jpg" alt="Image">
			<a href="https://example.com">Link</a>
			<video src="video.mp4"></video>
			<audio src="audio.mp3"></audio>
		</body>
		</html>`
		jsonData, err := html.ExtractToJSON([]byte(htmlContent))
		if err != nil {
			t.Fatalf("ExtractToJSON() failed: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(jsonData, &result); err != nil {
			t.Fatalf("Invalid JSON: %v", err)
		}

		// Check that standard fields exist
		expectedFields := []string{"title", "text", "word_count"}
		for _, field := range expectedFields {
			if _, ok := result[field]; !ok {
				t.Errorf("JSON should have '%s' field", field)
			}
		}
	})

	t.Run("JSON structure validation", func(t *testing.T) {
		htmlContent := `<html><body><p>Test</p></body></html>`
		jsonData, err := html.ExtractToJSON([]byte(htmlContent))
		if err != nil {
			t.Fatalf("ExtractToJSON() failed: %v", err)
		}

		var result html.Result
		if err := json.Unmarshal(jsonData, &result); err != nil {
			t.Fatalf("Invalid JSON structure: %v", err)
		}
	})

	t.Run("empty HTML JSON", func(t *testing.T) {
		jsonData, err := html.ExtractToJSON([]byte(""))
		if err != nil {
			t.Fatalf("ExtractToJSON() failed: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(jsonData, &result); err != nil {
			t.Fatalf("Invalid JSON: %v", err)
		}
	})
}

func TestResultMarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("marshal result with all fields", func(t *testing.T) {
		result := &html.Result{
			Title:          "Test Title",
			Text:           "Test text content.",
			WordCount:      3,
			ReadingTime:    time.Millisecond * 500,
			ProcessingTime: time.Millisecond * 100,
			Images: []html.ImageInfo{
				{URL: "img.jpg", Alt: "Image", Width: "100", Height: "100"},
			},
			Links: []html.LinkInfo{
				{URL: "https://example.com", Text: "Link", IsExternal: true},
			},
			Videos: []html.VideoInfo{
				{URL: "video.mp4", Type: "video/mp4"},
			},
			Audios: []html.AudioInfo{
				{URL: "audio.mp3", Type: "audio/mp3"},
			},
		}

		jsonData, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("json.Marshal() failed: %v", err)
		}

		var unmarshaled html.Result
		if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
			t.Fatalf("json.Unmarshal() failed: %v", err)
		}

		if unmarshaled.Title != result.Title {
			t.Errorf("Title = %q, want %q", unmarshaled.Title, result.Title)
		}
		if unmarshaled.WordCount != result.WordCount {
			t.Errorf("WordCount = %d, want %d", unmarshaled.WordCount, result.WordCount)
		}
	})

	t.Run("marshal empty result", func(t *testing.T) {
		result := &html.Result{}

		jsonData, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("json.Marshal() failed: %v", err)
		}

		if string(jsonData) == "" {
			t.Error("JSON should not be empty")
		}
	})
}

func TestImageInfoJSON(t *testing.T) {
	t.Parallel()

	t.Run("image info serialization", func(t *testing.T) {
		img := html.ImageInfo{
			URL:          "https://example.com/image.jpg",
			Alt:          "Test image",
			Title:        "Image title",
			Width:        "800",
			Height:       "600",
			IsDecorative: false,
			Position:     1,
		}

		jsonData, err := json.Marshal(img)
		if err != nil {
			t.Fatalf("json.Marshal() failed: %v", err)
		}

		var unmarshaled html.ImageInfo
		if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
			t.Fatalf("json.Unmarshal() failed: %v", err)
		}

		if unmarshaled.URL != img.URL {
			t.Errorf("URL = %q, want %q", unmarshaled.URL, img.URL)
		}
		if unmarshaled.Alt != img.Alt {
			t.Errorf("Alt = %q, want %q", unmarshaled.Alt, img.Alt)
		}
	})
}

// TestMediaInfoJSONSerialization consolidates JSON serialization tests for all media info types.
func TestMediaInfoJSONSerialization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     interface{}
		expected map[string]interface{}
	}{
		{
			name: "image info full",
			data: html.ImageInfo{
				URL:          "https://example.com/image.jpg",
				Alt:          "Test image",
				Title:        "Image title",
				Width:        "800",
				Height:       "600",
				IsDecorative: false,
				Position:     1,
			},
			expected: map[string]interface{}{
				"url":   "https://example.com/image.jpg",
				"alt":   "Test image",
				"title": "Image title",
			},
		},
		{
			name: "link info full",
			data: html.LinkInfo{
				URL:        "https://example.com",
				Text:       "Example",
				Title:      "Example site",
				IsExternal: true,
				IsNoFollow: true,
			},
			expected: map[string]interface{}{
				"url":         "https://example.com",
				"text":        "Example",
				"is_external": true,
			},
		},
		{
			name: "video info full",
			data: html.VideoInfo{
				URL:    "https://example.com/video.mp4",
				Type:   "video/mp4",
				Poster: "poster.jpg",
				Width:  "640",
				Height: "480",
			},
			expected: map[string]interface{}{
				"url":  "https://example.com/video.mp4",
				"type": "video/mp4",
			},
		},
		{
			name: "audio info full",
			data: html.AudioInfo{
				URL:  "https://example.com/audio.mp3",
				Type: "audio/mp3",
			},
			expected: map[string]interface{}{
				"url":  "https://example.com/audio.mp3",
				"type": "audio/mp3",
			},
		},
		{
			name: "image info minimal",
			data: html.ImageInfo{
				URL: "https://example.com/minimal.jpg",
			},
			expected: map[string]interface{}{
				"url": "https://example.com/minimal.jpg",
			},
		},
		{
			name: "link info minimal",
			data: html.LinkInfo{
				URL: "https://example.com/minimal",
			},
			expected: map[string]interface{}{
				"url": "https://example.com/minimal",
			},
		},
		{
			name: "video info minimal",
			data: html.VideoInfo{
				URL: "https://example.com/minimal.mp4",
			},
			expected: map[string]interface{}{
				"url": "https://example.com/minimal.mp4",
			},
		},
		{
			name: "audio info minimal",
			data: html.AudioInfo{
				URL: "https://example.com/minimal.mp3",
			},
			expected: map[string]interface{}{
				"url": "https://example.com/minimal.mp3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.data)
			if err != nil {
				t.Fatalf("json.Marshal() failed: %v", err)
			}

			// Verify it's valid JSON
			var result map[string]interface{}
			if err := json.Unmarshal(jsonData, &result); err != nil {
				t.Fatalf("json.Unmarshal() failed: %v", err)
			}

			// Verify expected fields
			for key, expectedValue := range tt.expected {
				actualValue, ok := result[key]
				if !ok {
					t.Errorf("Missing expected field: %s", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("Field %s = %v, want %v", key, actualValue, expectedValue)
				}
			}

			// Verify round-trip works
			jsonData2, err := json.Marshal(result)
			if err != nil {
				t.Errorf("Round-trip json.Marshal() failed: %v", err)
			}

			if len(jsonData2) == 0 {
				t.Error("Round-trip produced empty JSON")
			}
		})
	}
}

func TestOutputWithProcessor(t *testing.T) {
	t.Parallel()

	t.Run("processor ExtractToMarkdown", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		htmlContent := `<html><body><h1>Title</h1><p>Content</p></body></html>`
		markdown, err := p.ExtractToMarkdown([]byte(htmlContent))
		if err != nil {
			t.Fatalf("ExtractToMarkdown() failed: %v", err)
		}
		if markdown == "" {
			t.Error("Markdown should not be empty")
		}
	})

	t.Run("processor ExtractToJSON", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		htmlContent := `<html><body><h1>Title</h1><p>Content</p></body></html>`
		jsonData, err := p.ExtractToJSON([]byte(htmlContent))
		if err != nil {
			t.Fatalf("ExtractToJSON() failed: %v", err)
		}
		if len(jsonData) == 0 {
			t.Error("JSON should not be empty")
		}
	})
}

// Helper function to write file (cross-platform)
func writeFile(path string, content []byte) error {
	return os.WriteFile(path, content, 0644)
}
