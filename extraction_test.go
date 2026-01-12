package html_test

import (
	"strings"
	"testing"

	"github.com/cybergodev/html"
)

// TestArticleExtraction tests article content extraction
func TestArticleExtraction(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name        string
		html        string
		wantTitle   string
		wantContent string
	}{
		{
			name:        "article with h1 title",
			html:        `<html><body><article><h1>Article Title</h1><p>Article content here.</p></article></body></html>`,
			wantTitle:   "Article Title",
			wantContent: "Article content here",
		},
		{
			name:        "article with h2 title",
			html:        `<html><body><article><h2>Secondary Title</h2><p>Content text.</p></article></body></html>`,
			wantTitle:   "Secondary Title",
			wantContent: "Content text",
		},
		{
			name:        "multiple paragraphs",
			html:        `<html><body><article><h1>Title</h1><p>First paragraph.</p><p>Second paragraph.</p></article></body></html>`,
			wantTitle:   "Title",
			wantContent: "First paragraph",
		},
		{
			name:        "nested article",
			html:        `<html><body><div><article><h1>Nested</h1><p>Nested content.</p></article></div></body></html>`,
			wantTitle:   "Nested",
			wantContent: "Nested content",
		},
		{
			name:        "article with metadata",
			html:        `<html><head><title>Page Title</title></head><body><article><h1>Article</h1><p>Text</p></article></body></html>`,
			wantTitle:   "Page Title", // Title tag takes precedence
			wantContent: "Text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.ExtractWithDefaults(tt.html)
			if err != nil {
				t.Fatalf("ExtractWithDefaults() failed: %v", err)
			}

			if result.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", result.Title, tt.wantTitle)
			}

			if !strings.Contains(result.Text, tt.wantContent) {
				t.Errorf("Text should contain %q, got %q", tt.wantContent, result.Text)
			}
		})
	}
}

// TestImageExtraction tests image extraction
func TestImageExtraction(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name      string
		html      string
		wantCount int
		wantSrc   string
		wantAlt   string
	}{
		{
			name:      "single image",
			html:      `<html><body><img src="test.jpg" alt="Test Image"></body></html>`,
			wantCount: 1,
			wantSrc:   "test.jpg",
			wantAlt:   "Test Image",
		},
		{
			name:      "multiple images",
			html:      `<html><body><img src="img1.jpg"><img src="img2.jpg"></body></html>`,
			wantCount: 2,
		},
		{
			name:      "image without alt",
			html:      `<html><body><img src="noalt.jpg"></body></html>`,
			wantCount: 1,
			wantSrc:   "noalt.jpg",
		},
		{
			name:      "image in article",
			html:      `<html><body><article><img src="article.jpg" alt="Article Image"></article></body></html>`,
			wantCount: 1,
			wantSrc:   "article.jpg",
			wantAlt:   "Article Image",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.ExtractWithDefaults(tt.html)
			if err != nil {
				t.Fatalf("ExtractWithDefaults() failed: %v", err)
			}

			if len(result.Images) != tt.wantCount {
				t.Errorf("Images count = %d, want %d", len(result.Images), tt.wantCount)
			}

			if tt.wantCount > 0 && tt.wantSrc != "" {
				if result.Images[0].URL != tt.wantSrc {
					t.Errorf("Image[0].URL = %q, want %q", result.Images[0].URL, tt.wantSrc)
				}
			}

			if tt.wantAlt != "" && len(result.Images) > 0 {
				if result.Images[0].Alt != tt.wantAlt {
					t.Errorf("Image[0].Alt = %q, want %q", result.Images[0].Alt, tt.wantAlt)
				}
			}
		})
	}
}

// TestLinkExtraction tests link extraction
func TestLinkExtraction(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name      string
		html      string
		wantCount int
		wantHref  string
		wantText  string
	}{
		{
			name:      "single link",
			html:      `<html><body><a href="test.html">Test Link</a></body></html>`,
			wantCount: 1,
			wantHref:  "test.html",
			wantText:  "Test Link",
		},
		{
			name:      "multiple links",
			html:      `<html><body><a href="link1.html">Link 1</a><a href="link2.html">Link 2</a></body></html>`,
			wantCount: 2,
		},
		{
			name:      "link without text",
			html:      `<html><body><a href="empty.html"></a></body></html>`,
			wantCount: 1,
			wantHref:  "empty.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.ExtractWithDefaults(tt.html)
			if err != nil {
				t.Fatalf("ExtractWithDefaults() failed: %v", err)
			}

			if len(result.Links) != tt.wantCount {
				t.Errorf("Links count = %d, want %d", len(result.Links), tt.wantCount)
			}

			if tt.wantCount > 0 && tt.wantHref != "" {
				if result.Links[0].URL != tt.wantHref {
					t.Errorf("Link[0].URL = %q, want %q", result.Links[0].URL, tt.wantHref)
				}
			}

			if tt.wantText != "" && len(result.Links) > 0 {
				if result.Links[0].Text != tt.wantText {
					t.Errorf("Link[0].Text = %q, want %q", result.Links[0].Text, tt.wantText)
				}
			}
		})
	}
}

// TestVideoExtraction tests video extraction
func TestVideoExtraction(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name      string
		html      string
		wantCount int
		wantSrc   string
	}{
		{
			name:      "single video",
			html:      `<html><body><video src="test.mp4"></video></body></html>`,
			wantCount: 1,
			wantSrc:   "test.mp4",
		},
		{
			name:      "video with source tag",
			html:      `<html><body><video><source src="video.mp4" type="video/mp4"></video></body></html>`,
			wantCount: 1,
			wantSrc:   "video.mp4",
		},
		{
			name:      "multiple videos",
			html:      `<html><body><video src="v1.mp4"></video><video src="v2.mp4"></video></body></html>`,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.ExtractWithDefaults(tt.html)
			if err != nil {
				t.Fatalf("ExtractWithDefaults() failed: %v", err)
			}

			if len(result.Videos) != tt.wantCount {
				t.Errorf("Videos count = %d, want %d", len(result.Videos), tt.wantCount)
			}

			if tt.wantCount > 0 && tt.wantSrc != "" {
				if result.Videos[0].URL != tt.wantSrc {
					t.Errorf("Video[0].URL = %q, want %q", result.Videos[0].URL, tt.wantSrc)
				}
			}
		})
	}
}

// TestAudioExtraction tests audio extraction
func TestAudioExtraction(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name      string
		html      string
		wantCount int
		wantSrc   string
	}{
		{
			name:      "single audio",
			html:      `<html><body><audio src="test.mp3"></audio></body></html>`,
			wantCount: 1,
			wantSrc:   "test.mp3",
		},
		{
			name:      "audio with source tag",
			html:      `<html><body><audio><source src="audio.mp3" type="audio/mpeg"></audio></body></html>`,
			wantCount: 1,
			wantSrc:   "audio.mp3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.ExtractWithDefaults(tt.html)
			if err != nil {
				t.Fatalf("ExtractWithDefaults() failed: %v", err)
			}

			if len(result.Audios) != tt.wantCount {
				t.Errorf("Audios count = %d, want %d", len(result.Audios), tt.wantCount)
			}

			if tt.wantCount > 0 && tt.wantSrc != "" {
				if result.Audios[0].URL != tt.wantSrc {
					t.Errorf("Audio[0].URL = %q, want %q", result.Audios[0].URL, tt.wantSrc)
				}
			}
		})
	}
}

// TestMetadataExtraction tests metadata extraction
func TestMetadataExtraction(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name      string
		html      string
		wantTitle string
	}{
		{
			name: "basic title",
			html: `<html><head>
				<title>Page Title</title>
			</head><body></body></html>`,
			wantTitle: "Page Title",
		},
		{
			name: "og title",
			html: `<html><head>
				<meta property="og:title" content="OG Title">
			</head><body><h1>OG Title</h1></body></html>`,
			wantTitle: "OG Title",
		},
		{
			name:      "h1 title",
			html:      `<html><body><article><h1>Article Title</h1><p>Content</p></article></body></html>`,
			wantTitle: "Article Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.ExtractWithDefaults(tt.html)
			if err != nil {
				t.Fatalf("ExtractWithDefaults() failed: %v", err)
			}

			if tt.wantTitle != "" && result.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", result.Title, tt.wantTitle)
			}
		})
	}
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name    string
		html    string
		wantErr bool
	}{
		{
			name:    "empty string",
			html:    "",
			wantErr: false,
		},
		{
			name:    "whitespace only",
			html:    "   \n\t  ",
			wantErr: false,
		},
		{
			name:    "plain text",
			html:    "Just plain text",
			wantErr: false,
		},
		{
			name:    "unclosed tags",
			html:    "<html><body><p>Unclosed",
			wantErr: false,
		},
		{
			name:    "nested unclosed tags",
			html:    "<div><p><span>Text",
			wantErr: false,
		},
		{
			name:    "invalid HTML entities",
			html:    "<html><body>&invalid;</body></html>",
			wantErr: false,
		},
		{
			name:    "script tags",
			html:    `<html><body><script>alert('test');</script><p>Content</p></body></html>`,
			wantErr: false,
		},
		{
			name:    "style tags",
			html:    `<html><head><style>body{color:red;}</style></head><body><p>Content</p></body></html>`,
			wantErr: false,
		},
		{
			name:    "comments",
			html:    `<html><body><!-- Comment --><p>Content</p></body></html>`,
			wantErr: false,
		},
		{
			name:    "CDATA sections",
			html:    `<html><body><![CDATA[Some data]]><p>Content</p></body></html>`,
			wantErr: false,
		},
		{
			name:    "mixed case tags",
			html:    `<HTML><BODY><P>Content</P></BODY></HTML>`,
			wantErr: false,
		},
		{
			name:    "self-closing tags",
			html:    `<html><body><br/><hr/><p>Content</p></body></html>`,
			wantErr: false,
		},
		{
			name:    "attributes without values",
			html:    `<html><body><input disabled checked><p>Content</p></body></html>`,
			wantErr: false,
		},
		{
			name:    "special characters in text",
			html:    `<html><body><p>&lt;&gt;&amp;&quot;&#39;</p></body></html>`,
			wantErr: false,
		},
		{
			name:    "unicode characters",
			html:    `<html><body><p>Hello 世界 🌍</p></body></html>`,
			wantErr: false,
		},
		{
			name:    "very long text",
			html:    `<html><body><p>` + strings.Repeat("a", 10000) + `</p></body></html>`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.ExtractWithDefaults(tt.html)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractWithDefaults() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestScriptAndStyleRemoval tests that scripts and styles are removed
func TestScriptAndStyleRemoval(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `
		<html>
		<head>
			<style>body { color: red; }</style>
			<script>alert('test');</script>
		</head>
		<body>
			<script>console.log('inline');</script>
			<p>Visible content</p>
			<style>.hidden { display: none; }</style>
		</body>
		</html>
	`

	result, err := p.ExtractWithDefaults(htmlContent)
	if err != nil {
		t.Fatalf("ExtractWithDefaults() failed: %v", err)
	}

	if strings.Contains(result.Text, "alert") {
		t.Error("Text should not contain script content")
	}

	if strings.Contains(result.Text, "color: red") {
		t.Error("Text should not contain style content")
	}

	if !strings.Contains(result.Text, "Visible content") {
		t.Error("Text should contain visible content")
	}
}

// TestWhitespaceNormalization tests whitespace handling
func TestWhitespaceNormalization(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "multiple spaces",
			html: `<html><body><p>Multiple    spaces</p></body></html>`,
			want: "Multiple spaces",
		},
		{
			name: "newlines and tabs",
			html: "<html><body><p>Line1\n\t\tLine2</p></body></html>",
			want: "Line1",
		},
		{
			name: "leading and trailing whitespace",
			html: `<html><body><p>   Text   </p></body></html>`,
			want: "Text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.ExtractWithDefaults(tt.html)
			if err != nil {
				t.Fatalf("ExtractWithDefaults() failed: %v", err)
			}

			if !strings.Contains(result.Text, tt.want) {
				t.Errorf("Text should contain %q, got %q", tt.want, result.Text)
			}
		})
	}
}

// TestFormatOptions tests different format configurations
func TestFormatOptions(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `
		<html>
		<body>
			<article>
				<h1>Title</h1>
				<p>Paragraph</p>
				<img src="test.jpg" alt="Test">
				<a href="link.html">Link</a>
				<video src="video.mp4"></video>
				<audio src="audio.mp3"></audio>
			</article>
		</body>
		</html>
	`

	t.Run("preserve all", func(t *testing.T) {
		config := html.ExtractConfig{
			ExtractArticle: true,
			PreserveImages: true,
			PreserveLinks:  true,
			PreserveVideos: true,
			PreserveAudios: true,
		}

		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if len(result.Images) == 0 {
			t.Error("Should preserve images")
		}
		if len(result.Links) == 0 {
			t.Error("Should preserve links")
		}
		if len(result.Videos) == 0 {
			t.Error("Should preserve videos")
		}
		if len(result.Audios) == 0 {
			t.Error("Should preserve audios")
		}
	})

	t.Run("no images", func(t *testing.T) {
		config := html.ExtractConfig{
			ExtractArticle: true,
			PreserveImages: false,
			PreserveLinks:  true,
			PreserveVideos: true,
			PreserveAudios: true,
		}

		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if len(result.Images) != 0 {
			t.Error("Should not preserve images")
		}
	})

	t.Run("no links", func(t *testing.T) {
		config := html.ExtractConfig{
			ExtractArticle: true,
			PreserveImages: true,
			PreserveLinks:  false,
			PreserveVideos: true,
			PreserveAudios: true,
		}

		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if len(result.Links) != 0 {
			t.Error("Should not preserve links")
		}
	})
}
