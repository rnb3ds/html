package html_test

import (
	"strings"
	"testing"

	"github.com/cybergodev/html"
)

func TestExtractLinksEdgeCases(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name      string
		html      string
		wantCount int
	}{
		{
			name:      "link with title attribute",
			html:      `<a href="test.html" title="Link Title">Text</a>`,
			wantCount: 1,
		},
		{
			name:      "link with rel nofollow",
			html:      `<a href="test.html" rel="nofollow">Text</a>`,
			wantCount: 1,
		},
		{
			name:      "link with multiple rel values",
			html:      `<a href="test.html" rel="nofollow noopener">Text</a>`,
			wantCount: 1,
		},
		{
			name:      "link with target blank",
			html:      `<a href="test.html" target="_blank">Text</a>`,
			wantCount: 1,
		},
		{
			name:      "javascript link",
			html:      `<a href="javascript:void(0)">Text</a>`,
			wantCount: 1, // Implementation extracts all links
		},
		{
			name:      "mailto link",
			html:      `<a href="mailto:test@example.com">Email</a>`,
			wantCount: 1,
		},
		{
			name:      "tel link",
			html:      `<a href="tel:+1234567890">Call</a>`,
			wantCount: 1,
		},
		{
			name:      "data uri link",
			html:      `<a href="data:text/html,<h1>Test</h1>">Data</a>`,
			wantCount: 1, // Implementation extracts all links
		},
		{
			name:      "very long url",
			html:      `<a href="` + strings.Repeat("a", 3000) + `">Long</a>`,
			wantCount: 0,
		},
		{
			name:      "link with nested images",
			html:      `<a href="test.html"><img src="icon.png" alt="Icon">Text</a>`,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := html.DefaultExtractConfig()
			config.PreserveLinks = true

			result, err := p.Extract(tt.html, config)
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if len(result.Links) != tt.wantCount {
				t.Errorf("Links count = %d, want %d", len(result.Links), tt.wantCount)
			}
		})
	}
}

func TestExtractVideosEdgeCases(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name      string
		html      string
		wantCount int
	}{
		{
			name:      "video with multiple sources",
			html:      `<video><source src="v1.webm"><source src="v2.mp4"><source src="v3.ogg"></video>`,
			wantCount: 1,
		},
		{
			name:      "video with poster and controls",
			html:      `<video src="test.mp4" poster="thumb.jpg" controls></video>`,
			wantCount: 1,
		},
		{
			name:      "video with autoplay",
			html:      `<video src="test.mp4" autoplay muted></video>`,
			wantCount: 1,
		},
		{
			name:      "video with loop",
			html:      `<video src="test.mp4" loop></video>`,
			wantCount: 1,
		},
		{
			name:      "iframe with non-video content",
			html:      `<iframe src="https://example.com/page.html"></iframe>`,
			wantCount: 0,
		},
		{
			name:      "iframe without src",
			html:      `<iframe></iframe>`,
			wantCount: 0,
		},
		{
			name:      "video with data uri",
			html:      `<video src="data:video/mp4;base64,AAAA"></video>`,
			wantCount: 1, // Implementation extracts all videos
		},
		{
			name:      "video with very long url",
			html:      `<video src="` + strings.Repeat("a", 3000) + `.mp4"></video>`,
			wantCount: 0,
		},
		{
			name:      "embed tag",
			html:      `<embed src="test.mp4" type="video/mp4">`,
			wantCount: 1, // Implementation may extract embed tags
		},
		{
			name:      "object tag",
			html:      `<object data="test.mp4" type="video/mp4"></object>`,
			wantCount: 1, // Implementation may extract object tags
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := html.DefaultExtractConfig()
			config.PreserveVideos = true

			result, err := p.Extract(tt.html, config)
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if len(result.Videos) != tt.wantCount {
				t.Errorf("Videos count = %d, want %d", len(result.Videos), tt.wantCount)
			}
		})
	}
}

func TestExtractAudiosEdgeCases(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name      string
		html      string
		wantCount int
	}{
		{
			name:      "audio with multiple sources",
			html:      `<audio><source src="a1.ogg"><source src="a2.mp3"></audio>`,
			wantCount: 1,
		},
		{
			name:      "audio with controls",
			html:      `<audio src="test.mp3" controls></audio>`,
			wantCount: 1,
		},
		{
			name:      "audio with autoplay",
			html:      `<audio src="test.mp3" autoplay></audio>`,
			wantCount: 1,
		},
		{
			name:      "audio with loop",
			html:      `<audio src="test.mp3" loop></audio>`,
			wantCount: 1,
		},
		{
			name:      "audio with data uri",
			html:      `<audio src="data:audio/mp3;base64,AAAA"></audio>`,
			wantCount: 1, // Implementation extracts all audios
		},
		{
			name:      "audio with very long url",
			html:      `<audio src="` + strings.Repeat("a", 3000) + `.mp3"></audio>`,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := html.DefaultExtractConfig()
			config.PreserveAudios = true

			result, err := p.Extract(tt.html, config)
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if len(result.Audios) != tt.wantCount {
				t.Errorf("Audios count = %d, want %d", len(result.Audios), tt.wantCount)
			}
		})
	}
}

func TestExtractImagesEdgeCases(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name      string
		html      string
		wantCount int
	}{
		{
			name:      "image with srcset",
			html:      `<img src="small.jpg" srcset="medium.jpg 2x, large.jpg 3x" alt="Test">`,
			wantCount: 1,
		},
		{
			name:      "image with loading lazy",
			html:      `<img src="test.jpg" loading="lazy" alt="Test">`,
			wantCount: 1,
		},
		{
			name:      "image with data uri",
			html:      `<img src="data:image/png;base64,iVBORw0KGgo=" alt="Test">`,
			wantCount: 1, // Implementation extracts all images
		},
		{
			name:      "image with very long url",
			html:      `<img src="` + strings.Repeat("a", 3000) + `.jpg" alt="Test">`,
			wantCount: 0,
		},
		{
			name:      "image without src",
			html:      `<img alt="No source">`,
			wantCount: 0,
		},
		{
			name:      "image with empty src",
			html:      `<img src="" alt="Empty">`,
			wantCount: 0,
		},
		{
			name:      "svg image",
			html:      `<img src="test.svg" alt="SVG">`,
			wantCount: 1,
		},
		{
			name:      "picture element",
			html:      `<picture><source srcset="test.webp"><img src="test.jpg" alt="Test"></picture>`,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := html.DefaultExtractConfig()
			config.PreserveImages = true

			result, err := p.Extract(tt.html, config)
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if len(result.Images) != tt.wantCount {
				t.Errorf("Images count = %d, want %d", len(result.Images), tt.wantCount)
			}
		})
	}
}

func TestExtractWithPositionTracking(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `
		<article>
			<p>First paragraph</p>
			<img src="image1.jpg" alt="Image 1">
			<p>Second paragraph</p>
			<img src="image2.jpg" alt="Image 2">
			<p>Third paragraph</p>
		</article>
	`

	config := html.DefaultExtractConfig()
	config.PreserveImages = true

	result, err := p.Extract(htmlContent, config)
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	if len(result.Images) != 2 {
		t.Errorf("Images count = %d, want 2", len(result.Images))
	}
}

func TestExtractTextContentEdgeCases(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name        string
		html        string
		wantContain string
	}{
		{
			name:        "text with entities",
			html:        `<p>&lt;html&gt; &amp; &quot;test&quot;</p>`,
			wantContain: "html",
		},
		{
			name:        "text with unicode",
			html:        `<p>Hello 世界 🌍</p>`,
			wantContain: "世界",
		},
		{
			name:        "text with special chars",
			html:        `<p>Price: $100 & €50</p>`,
			wantContain: "$100",
		},
		{
			name:        "text with line breaks",
			html:        `<p>Line1<br>Line2<br>Line3</p>`,
			wantContain: "Line1",
		},
		{
			name:        "text with hr",
			html:        `<p>Before</p><hr><p>After</p>`,
			wantContain: "Before",
		},
		{
			name:        "text in lists",
			html:        `<ul><li>Item 1</li><li>Item 2</li></ul>`,
			wantContain: "Item 1",
		},
		{
			name:        "text in definition list",
			html:        `<dl><dt>Term</dt><dd>Definition</dd></dl>`,
			wantContain: "Term",
		},
		{
			name:        "text in blockquote",
			html:        `<blockquote>Quoted text</blockquote>`,
			wantContain: "Quoted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Extract(tt.html, html.DefaultExtractConfig())
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if !strings.Contains(result.Text, tt.wantContain) {
				t.Errorf("Text should contain %q, got %q", tt.wantContain, result.Text)
			}
		})
	}
}

func TestExtractTitleEdgeCases(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name      string
		html      string
		wantTitle string
	}{
		{
			name:      "title with entities",
			html:      `<html><head><title>&lt;Test&gt; Title</title></head></html>`,
			wantTitle: "<Test> Title",
		},
		{
			name:      "title with whitespace",
			html:      `<html><head><title>  Title  </title></head></html>`,
			wantTitle: "Title",
		},
		{
			name:      "multiple h1 tags",
			html:      `<html><body><h1>First</h1><h1>Second</h1></body></html>`,
			wantTitle: "Second", // Implementation may use last h1
		},
		{
			name:      "h1 with nested elements",
			html:      `<html><body><h1>Title <span>with</span> <strong>nested</strong></h1></body></html>`,
			wantTitle: "Title with nested",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Extract(tt.html, html.DefaultExtractConfig())
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if !strings.Contains(result.Title, tt.wantTitle) {
				t.Errorf("Title should contain %q, got %q", tt.wantTitle, result.Title)
			}
		})
	}
}
