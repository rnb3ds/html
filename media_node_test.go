package html_test

import (
	"strings"
	"testing"

	"github.com/cybergodev/html"
)

// parseIframeNode and parseEmbedNode are only reachable when iframe/embed tags
// survive sanitization (currently they are in tagsToRemoveMap). These tests verify
// extraction works through the regex-based fallback path.

func TestIframeNodeVideoExtraction(t *testing.T) {
	t.Parallel()

	t.Run("iframe with video URL extracted", func(t *testing.T) {
		p, err := html.New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		htmlContent := buildPaddedHTML(
			`<iframe src="https://www.youtube.com/embed/dQw4w9WgXcQ" width="640" height="480"></iframe>`,
		)
		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		found := false
		for _, v := range result.Videos {
			if v.URL == "https://www.youtube.com/embed/dQw4w9WgXcQ" {
				found = true
				if v.Type == "" {
					t.Error("video type should not be empty")
				}
				break
			}
		}
		if !found {
			t.Error("youtube embed URL not found in videos")
		}
	})

	t.Run("iframe with non-video URL not extracted as video", func(t *testing.T) {
		p, err := html.New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		htmlContent := buildPaddedHTML(
			`<iframe src="https://example.com/page" width="800" height="600"></iframe>`,
		)
		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		for _, v := range result.Videos {
			if v.URL == "https://example.com/page" {
				t.Error("non-video iframe URL should not appear in videos")
			}
		}
	})

	t.Run("iframe without src produces no video", func(t *testing.T) {
		p, err := html.New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		htmlContent := `<html><body><iframe width="640" height="480"></iframe></body></html>`
		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if len(result.Videos) > 0 {
			for _, v := range result.Videos {
				if v.URL == "" {
					t.Error("iframe without src should not produce a video entry")
				}
			}
		}
	})
}

func TestEmbedNodeVideoExtraction(t *testing.T) {
	t.Parallel()

	t.Run("embed with video src URL extracted", func(t *testing.T) {
		p, err := html.New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		htmlContent := buildPaddedHTML(
			`<embed src="https://www.youtube.com/embed/test123" type="video/mp4" width="800" height="600">`,
		)
		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if len(result.Videos) == 0 {
			t.Fatal("expected at least one video from embed tag")
		}
	})

	t.Run("embed with data attribute extracted", func(t *testing.T) {
		p, err := html.New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		htmlContent := buildPaddedHTML(
			`<embed data="https://player.vimeo.com/video/12345" type="application/x-shockwave-flash" width="400" height="300">`,
		)
		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if len(result.Videos) == 0 {
			t.Fatal("expected at least one video from embed data attribute")
		}
	})

	t.Run("embed with non-video URL ignored", func(t *testing.T) {
		p, err := html.New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		htmlContent := buildPaddedHTML(
			`<embed src="https://example.com/file.swf" type="application/octet-stream" width="100" height="100">`,
		)
		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		for _, v := range result.Videos {
			if v.URL == "https://example.com/file.swf" {
				t.Error("non-video embed URL should not appear in videos")
			}
		}
	})
}

// buildPaddedHTML creates an HTML document with padding to ensure regex-based extraction
// is triggered and the content is large enough for full processing.
func buildPaddedHTML(inner string) string {
	var sb strings.Builder
	sb.WriteString("<html><head><title>Test</title></head><body>")
	for range 50 {
		sb.WriteString("<p>This is paragraph content to pad the HTML for testing purposes.</p>")
	}
	sb.WriteString(inner)
	for range 50 {
		sb.WriteString("<p>This is paragraph content to pad the HTML for testing purposes.</p>")
	}
	sb.WriteString("</body></html>")
	return sb.String()
}
