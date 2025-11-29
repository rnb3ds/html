package html_test

import (
	"strings"
	"testing"

	"github.com/cybergodev/html"
)

func TestExtractTitle(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name     string
		html     string
		wantText string
	}{
		{
			name:     "title tag",
			html:     `<html><head><title>Page Title</title></head><body></body></html>`,
			wantText: "Page Title",
		},
		{
			name:     "h1 tag",
			html:     `<html><body><h1>Main Heading</h1></body></html>`,
			wantText: "Main Heading",
		},
		{
			name:     "h2 fallback",
			html:     `<html><body><h2>Secondary Heading</h2></body></html>`,
			wantText: "Secondary Heading",
		},
		{
			name:     "no title",
			html:     `<html><body><p>No title</p></body></html>`,
			wantText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Extract(tt.html, html.DefaultExtractConfig())
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}
			if result.Title != tt.wantText {
				t.Errorf("Title = %q, want %q", result.Title, tt.wantText)
			}
		})
	}
}

func TestExtractText(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name     string
		html     string
		wantText string
	}{
		{
			name:     "simple paragraph",
			html:     `<html><body><p>Hello World</p></body></html>`,
			wantText: "Hello World",
		},
		{
			name:     "multiple paragraphs",
			html:     `<html><body><p>First</p><p>Second</p></body></html>`,
			wantText: "First",
		},
		{
			name:     "nested elements",
			html:     `<html><body><div><p>Nested <strong>text</strong></p></div></body></html>`,
			wantText: "Nested text",
		},
		{
			name:     "script tags removed",
			html:     `<html><body><p>Visible</p><script>alert('hidden')</script></body></html>`,
			wantText: "Visible",
		},
		{
			name:     "style tags removed",
			html:     `<html><body><p>Visible</p><style>body{color:red}</style></body></html>`,
			wantText: "Visible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Extract(tt.html, html.DefaultExtractConfig())
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}
			if !strings.Contains(result.Text, tt.wantText) {
				t.Errorf("Text = %q, want to contain %q", result.Text, tt.wantText)
			}
		})
	}
}

func TestExtractImages(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name           string
		html           string
		wantCount      int
		wantURL        string
		wantAlt        string
		preserveImages bool
	}{
		{
			name:           "single image",
			html:           `<html><body><img src="test.jpg" alt="Test Image"></body></html>`,
			wantCount:      1,
			wantURL:        "test.jpg",
			wantAlt:        "Test Image",
			preserveImages: true,
		},
		{
			name:           "multiple images",
			html:           `<html><body><img src="1.jpg"><img src="2.jpg"></body></html>`,
			wantCount:      2,
			preserveImages: true,
		},
		{
			name:           "image without alt",
			html:           `<html><body><img src="test.jpg"></body></html>`,
			wantCount:      1,
			wantURL:        "test.jpg",
			wantAlt:        "",
			preserveImages: true,
		},
		{
			name:           "no images",
			html:           `<html><body><p>No images</p></body></html>`,
			wantCount:      0,
			preserveImages: true,
		},
		{
			name:           "images disabled",
			html:           `<html><body><img src="test.jpg"></body></html>`,
			wantCount:      0,
			preserveImages: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := html.DefaultExtractConfig()
			config.PreserveImages = tt.preserveImages

			result, err := p.Extract(tt.html, config)
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if len(result.Images) != tt.wantCount {
				t.Errorf("Images count = %d, want %d", len(result.Images), tt.wantCount)
			}

			if tt.wantCount > 0 && tt.wantURL != "" {
				if result.Images[0].URL != tt.wantURL {
					t.Errorf("Image URL = %q, want %q", result.Images[0].URL, tt.wantURL)
				}
				if result.Images[0].Alt != tt.wantAlt {
					t.Errorf("Image Alt = %q, want %q", result.Images[0].Alt, tt.wantAlt)
				}
			}
		})
	}
}

func TestExtractLinks(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name          string
		html          string
		wantCount     int
		wantURL       string
		wantText      string
		preserveLinks bool
	}{
		{
			name:          "single link",
			html:          `<html><body><a href="https://example.com">Example</a></body></html>`,
			wantCount:     1,
			wantURL:       "https://example.com",
			wantText:      "Example",
			preserveLinks: true,
		},
		{
			name:          "multiple links",
			html:          `<html><body><a href="link1.html">Link 1</a><a href="link2.html">Link 2</a></body></html>`,
			wantCount:     2,
			preserveLinks: true,
		},
		{
			name:          "no links",
			html:          `<html><body><p>No links</p></body></html>`,
			wantCount:     0,
			preserveLinks: true,
		},
		{
			name:          "links disabled",
			html:          `<html><body><a href="test.html">Test</a></body></html>`,
			wantCount:     0,
			preserveLinks: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := html.DefaultExtractConfig()
			config.PreserveLinks = tt.preserveLinks

			result, err := p.Extract(tt.html, config)
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if len(result.Links) != tt.wantCount {
				t.Errorf("Links count = %d, want %d", len(result.Links), tt.wantCount)
			}

			if tt.wantCount > 0 && tt.wantURL != "" {
				if result.Links[0].URL != tt.wantURL {
					t.Errorf("Link URL = %q, want %q", result.Links[0].URL, tt.wantURL)
				}
				if result.Links[0].Text != tt.wantText {
					t.Errorf("Link Text = %q, want %q", result.Links[0].Text, tt.wantText)
				}
			}
		})
	}
}

func TestExtractVideos(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name           string
		html           string
		wantCount      int
		wantURL        string
		preserveVideos bool
	}{
		{
			name:           "video tag with src",
			html:           `<html><body><video src="test.mp4"></video></body></html>`,
			wantCount:      1,
			wantURL:        "test.mp4",
			preserveVideos: true,
		},
		{
			name:           "video with source webm",
			html:           `<html><body><video><source src="test.webm" type="video/webm"></video></body></html>`,
			wantCount:      1,
			wantURL:        "test.webm",
			preserveVideos: true,
		},
		{
			name:           "video with source ogg",
			html:           `<html><body><video><source src="test.ogg"></video></body></html>`,
			wantCount:      1,
			wantURL:        "test.ogg",
			preserveVideos: true,
		},
		{
			name:           "video mov",
			html:           `<html><body><video src="test.mov"></video></body></html>`,
			wantCount:      1,
			wantURL:        "test.mov",
			preserveVideos: true,
		},
		{
			name:           "video avi",
			html:           `<html><body><video src="test.avi"></video></body></html>`,
			wantCount:      1,
			wantURL:        "test.avi",
			preserveVideos: true,
		},
		{
			name:           "video mkv",
			html:           `<html><body><video src="test.mkv"></video></body></html>`,
			wantCount:      1,
			wantURL:        "test.mkv",
			preserveVideos: true,
		},
		{
			name:           "youtube embed",
			html:           `<html><body><iframe src="https://youtube.com/embed/abc123"></iframe></body></html>`,
			wantCount:      1,
			wantURL:        "https://youtube.com/embed/abc123",
			preserveVideos: true,
		},
		{
			name:           "vimeo embed",
			html:           `<html><body><iframe src="https://player.vimeo.com/video/123456"></iframe></body></html>`,
			wantCount:      1,
			wantURL:        "https://player.vimeo.com/video/123456",
			preserveVideos: true,
		},
		{
			name:           "video with poster",
			html:           `<html><body><video src="test.mp4" poster="thumb.jpg"></video></body></html>`,
			wantCount:      1,
			wantURL:        "test.mp4",
			preserveVideos: true,
		},
		{
			name:           "video with dimensions",
			html:           `<html><body><video src="test.mp4" width="640" height="480"></video></body></html>`,
			wantCount:      1,
			wantURL:        "test.mp4",
			preserveVideos: true,
		},
		{
			name:           "multiple source tags",
			html:           `<html><body><video><source src="test.webm"><source src="test.mp4"></video></body></html>`,
			wantCount:      1,
			wantURL:        "test.webm",
			preserveVideos: true,
		},
		{
			name:           "video without src",
			html:           `<html><body><video></video></body></html>`,
			wantCount:      0,
			preserveVideos: true,
		},
		{
			name:           "iframe without video",
			html:           `<html><body><iframe src="https://example.com/page.html"></iframe></body></html>`,
			wantCount:      0,
			preserveVideos: true,
		},
		{
			name:           "multiple videos",
			html:           `<html><body><video src="1.mp4"></video><video src="2.mp4"></video></body></html>`,
			wantCount:      2,
			preserveVideos: true,
		},
		{
			name:           "videos disabled",
			html:           `<html><body><video src="test.mp4"></video></body></html>`,
			wantCount:      0,
			preserveVideos: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := html.DefaultExtractConfig()
			config.PreserveVideos = tt.preserveVideos

			result, err := p.Extract(tt.html, config)
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if len(result.Videos) != tt.wantCount {
				t.Errorf("Videos count = %d, want %d", len(result.Videos), tt.wantCount)
			}

			if tt.wantCount > 0 && tt.wantURL != "" {
				if result.Videos[0].URL != tt.wantURL {
					t.Errorf("Video URL = %q, want %q", result.Videos[0].URL, tt.wantURL)
				}
			}
		})
	}
}

func TestExtractAudios(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name           string
		html           string
		wantCount      int
		wantURL        string
		preserveAudios bool
	}{
		{
			name:           "audio with src mp3",
			html:           `<html><body><audio src="test.mp3"></audio></body></html>`,
			wantCount:      1,
			wantURL:        "test.mp3",
			preserveAudios: true,
		},
		{
			name:           "audio with source ogg",
			html:           `<html><body><audio><source src="test.ogg" type="audio/ogg"></audio></body></html>`,
			wantCount:      1,
			wantURL:        "test.ogg",
			preserveAudios: true,
		},
		{
			name:           "audio wav",
			html:           `<html><body><audio src="test.wav"></audio></body></html>`,
			wantCount:      1,
			wantURL:        "test.wav",
			preserveAudios: true,
		},
		{
			name:           "audio m4a",
			html:           `<html><body><audio src="test.m4a"></audio></body></html>`,
			wantCount:      1,
			wantURL:        "test.m4a",
			preserveAudios: true,
		},
		{
			name:           "audio flac",
			html:           `<html><body><audio src="test.flac"></audio></body></html>`,
			wantCount:      1,
			wantURL:        "test.flac",
			preserveAudios: true,
		},
		{
			name:           "audio without src",
			html:           `<html><body><audio></audio></body></html>`,
			wantCount:      0,
			preserveAudios: true,
		},
		{
			name:           "multiple source tags",
			html:           `<html><body><audio><source src="test.ogg"><source src="test.mp3"></audio></body></html>`,
			wantCount:      1,
			wantURL:        "test.ogg",
			preserveAudios: true,
		},
		{
			name:           "multiple audios",
			html:           `<html><body><audio src="1.mp3"></audio><audio src="2.mp3"></audio></body></html>`,
			wantCount:      2,
			preserveAudios: true,
		},
		{
			name:           "audios disabled",
			html:           `<html><body><audio src="test.mp3"></audio></body></html>`,
			wantCount:      0,
			preserveAudios: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := html.DefaultExtractConfig()
			config.PreserveAudios = tt.preserveAudios

			result, err := p.Extract(tt.html, config)
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if len(result.Audios) != tt.wantCount {
				t.Errorf("Audios count = %d, want %d", len(result.Audios), tt.wantCount)
			}

			if tt.wantCount > 0 && tt.wantURL != "" {
				if result.Audios[0].URL != tt.wantURL {
					t.Errorf("Audio URL = %q, want %q", result.Audios[0].URL, tt.wantURL)
				}
			}
		})
	}
}

func TestWordCountAndReadingTime(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name          string
		html          string
		wantWordCount int
	}{
		{
			name:          "single word",
			html:          `<html><body><p>Hello</p></body></html>`,
			wantWordCount: 1,
		},
		{
			name:          "multiple words",
			html:          `<html><body><p>Hello World Test</p></body></html>`,
			wantWordCount: 3,
		},
		{
			name:          "empty",
			html:          `<html><body></body></html>`,
			wantWordCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Extract(tt.html, html.DefaultExtractConfig())
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if result.WordCount != tt.wantWordCount {
				t.Errorf("WordCount = %d, want %d", result.WordCount, tt.wantWordCount)
			}

			if tt.wantWordCount > 0 && result.ReadingTime == 0 {
				t.Error("ReadingTime should be > 0 when WordCount > 0")
			}
		})
	}
}

func TestArticleExtraction(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `
		<html>
		<body>
			<nav>Navigation menu</nav>
			<article>
				<h1>Article Title</h1>
				<p>This is the main article content.</p>
				<p>It has multiple paragraphs.</p>
			</article>
			<aside>Sidebar content</aside>
			<footer>Footer content</footer>
		</body>
		</html>
	`

	config := html.DefaultExtractConfig()
	config.ExtractArticle = true

	result, err := p.Extract(htmlContent, config)
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	if !strings.Contains(result.Text, "main article content") {
		t.Error("Article extraction should include main content")
	}

	if strings.Contains(result.Text, "Navigation menu") {
		t.Error("Article extraction should exclude navigation")
	}
}

func TestInlineImageFormats(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><p>Before</p><img src="test.jpg" alt="Test"><p>After</p></body></html>`

	tests := []struct {
		name        string
		format      string
		wantContain string
	}{
		{
			name:        "none format",
			format:      "none",
			wantContain: "Before",
		},
		{
			name:        "placeholder format",
			format:      "placeholder",
			wantContain: "[IMAGE:1]",
		},
		{
			name:        "markdown format",
			format:      "markdown",
			wantContain: "![Test](test.jpg)",
		},
		{
			name:        "html format",
			format:      "html",
			wantContain: `<img src="test.jpg"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := html.DefaultExtractConfig()
			config.InlineImageFormat = tt.format

			result, err := p.Extract(htmlContent, config)
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if !strings.Contains(result.Text, tt.wantContain) {
				t.Errorf("Text = %q, want to contain %q", result.Text, tt.wantContain)
			}
		})
	}
}
