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

// TestParseIframeNodeDOMPath exercises parseIframeNode via the DOM walk path
// by disabling sanitization so iframe tags survive into the parsed tree.
func TestParseIframeNodeDOMPath(t *testing.T) {
	t.Parallel()

	t.Run("iframe with video src via DOM", func(t *testing.T) {
		cfg := html.DefaultConfig()
		cfg.EnableSanitization = false
		p, err := html.New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		htmlContent := `<html><head><title>Iframe DOM Test</title></head><body>
			<article>
				<p>Main article content for extraction.</p>
				<iframe src="https://www.youtube.com/embed/dQw4w9WgXcQ" width="640" height="480"></iframe>
			</article>
		</body></html>`
		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		found := false
		for _, v := range result.Videos {
			if v.URL == "https://www.youtube.com/embed/dQw4w9WgXcQ" {
				found = true
				if v.Type != "embed" {
					t.Errorf("expected type 'embed', got %q", v.Type)
				}
				break
			}
		}
		if !found {
			t.Error("iframe video not found via DOM path")
		}
	})

	t.Run("iframe with non-video src ignored via DOM", func(t *testing.T) {
		cfg := html.DefaultConfig()
		cfg.EnableSanitization = false
		p, err := html.New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		htmlContent := `<html><head><title>Non-video Iframe</title></head><body>
			<article>
				<p>Article content.</p>
				<iframe src="https://example.com/page" width="800" height="600"></iframe>
			</article>
		</body></html>`
		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		for _, v := range result.Videos {
			if v.URL == "https://example.com/page" {
				t.Error("non-video iframe should not appear in videos via DOM path")
			}
		}
	})

	t.Run("iframe without src produces empty video via DOM", func(t *testing.T) {
		cfg := html.DefaultConfig()
		cfg.EnableSanitization = false
		p, err := html.New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		htmlContent := `<html><body><article>
			<p>Content.</p>
			<iframe width="640" height="480"></iframe>
		</article></body></html>`
		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		for _, v := range result.Videos {
			if v.URL == "" {
				t.Error("iframe without src should not produce a video entry")
			}
		}
	})
}

// TestParseEmbedNodeDOMPath exercises parseEmbedNode via the DOM walk path
// by disabling sanitization so embed/object tags survive into the parsed tree.
func TestParseEmbedNodeDOMPath(t *testing.T) {
	t.Parallel()

	t.Run("embed with video src via DOM", func(t *testing.T) {
		cfg := html.DefaultConfig()
		cfg.EnableSanitization = false
		p, err := html.New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		htmlContent := `<html><head><title>Embed DOM Test</title></head><body>
			<article>
				<p>Article content.</p>
				<embed src="https://www.youtube.com/embed/test123" type="video/mp4" width="800" height="600">
			</article>
		</body></html>`
		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		found := false
		for _, v := range result.Videos {
			if v.URL == "https://www.youtube.com/embed/test123" {
				found = true
				break
			}
		}
		if !found {
			t.Error("embed video not found via DOM path")
		}
	})

	t.Run("embed with data attribute via DOM", func(t *testing.T) {
		cfg := html.DefaultConfig()
		cfg.EnableSanitization = false
		p, err := html.New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		htmlContent := `<html><body><article>
			<p>Content.</p>
			<embed data="https://player.vimeo.com/video/12345" type="application/x-shockwave-flash" width="400" height="300">
		</article></body></html>`
		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		found := false
		for _, v := range result.Videos {
			if v.URL == "https://player.vimeo.com/video/12345" {
				found = true
				break
			}
		}
		if !found {
			t.Error("embed data attribute video not found via DOM path")
		}
	})

	t.Run("embed with non-video URL ignored via DOM", func(t *testing.T) {
		cfg := html.DefaultConfig()
		cfg.EnableSanitization = false
		p, err := html.New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		htmlContent := `<html><body><article>
			<p>Content.</p>
			<embed src="https://example.com/file.swf" type="application/octet-stream">
		</article></body></html>`
		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		for _, v := range result.Videos {
			if v.URL == "https://example.com/file.swf" {
				t.Error("non-video embed should not appear in videos via DOM path")
			}
		}
	})

	t.Run("object tag with video data via DOM", func(t *testing.T) {
		cfg := html.DefaultConfig()
		cfg.EnableSanitization = false
		p, err := html.New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		htmlContent := `<html><body><article>
			<p>Content.</p>
			<object data="https://www.youtube.com/embed/obj123" type="video/mp4" width="320" height="240"></object>
		</article></body></html>`
		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		found := false
		for _, v := range result.Videos {
			if v.URL == "https://www.youtube.com/embed/obj123" {
				found = true
				break
			}
		}
		if !found {
			t.Error("object video not found via DOM path")
		}
	})
}
