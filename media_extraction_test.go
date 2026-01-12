package html_test

import (
	"testing"

	"github.com/cybergodev/html"
)

// TestIframeExtraction tests iframe video extraction
func TestIframeExtraction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		html       string
		wantCount  int
		wantType   string
	}{
		{
			name: "YouTube iframe",
			html: `
				<html><body>
					<iframe src="https://www.youtube.com/embed/test123" width="640" height="480"></iframe>
				</body></html>
			`,
			wantCount: 1,
			wantType:  "embed",
		},
		{
			name: "Vimeo iframe",
			html: `
				<html><body>
					<iframe src="https://player.vimeo.com/video/456789" width="640" height="360"></iframe>
				</body></html>
			`,
			wantCount: 1,
			wantType:  "embed",
		},
		{
			name: "Dailymotion iframe",
			html: `
				<html><body>
					<iframe src="https://www.dailymotion.com/embed/videoxyz" width="560" height="315"></iframe>
				</body></html>
			`,
			wantCount: 1,
			wantType:  "embed",
		},
		{
			name: "non-video iframe",
			html: `
				<html><body>
					<iframe src="https://example.com/page.html"></iframe>
				</body></html>
			`,
			wantCount: 0,
			wantType:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := html.NewWithDefaults()
			defer p.Close()

			result, err := p.Extract(tt.html, html.DefaultExtractConfig())
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if len(result.Videos) != tt.wantCount {
				t.Errorf("Extract() returned %d videos, want %d", len(result.Videos), tt.wantCount)
			}

			if tt.wantCount > 0 && len(result.Videos) > 0 && result.Videos[0].Type != tt.wantType {
				t.Errorf("Extract() video type = %q, want %q", result.Videos[0].Type, tt.wantType)
			}
		})
	}
}

// TestEmbedExtraction tests embed and object video extraction
func TestEmbedExtraction(t *testing.T) {
	t.Parallel()

	t.Run("embed tag with video URL", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<embed src="https://www.youtube.com/embed/test123" type="video/mp4" width="640" height="480">
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if len(result.Videos) == 0 {
			t.Error("Extract() returned no videos, want at least 1")
		}
	})

	t.Run("object tag with video URL", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<object data="https://www.youtube.com/embed/test123" type="video/mp4" width="640" height="480"></object>
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if len(result.Videos) == 0 {
			t.Error("Extract() returned no videos, want at least 1")
		}
	})

	t.Run("non-video embed", func(t *testing.T) {
		htmlContent := `
			<html><body>
				<embed src="https://example.com/content.pdf" type="application/pdf">
			</body></html>
		`

		p := html.NewWithDefaults()
		defer p.Close()

		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if len(result.Videos) != 0 {
			t.Errorf("Extract() returned %d videos, want 0", len(result.Videos))
		}
	})
}

// TestVideoSourceExtraction tests video with source elements
func TestVideoSourceExtraction(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html><body>
			<video width="640" height="480">
				<source src="https://example.com/video.mp4" type="video/mp4">
				<source src="https://example.com/video.webm" type="video/webm">
			</video>
		</body></html>
	`

	p := html.NewWithDefaults()
	defer p.Close()

	result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	if len(result.Videos) == 0 {
		t.Error("Extract() returned no videos, want at least 1")
	}

	if result.Videos[0].URL != "https://example.com/video.mp4" {
		t.Errorf("Extract() video URL = %q, want 'https://example.com/video.mp4'", result.Videos[0].URL)
	}
}

// TestAudioSourceExtraction tests audio with source elements
func TestAudioSourceExtraction(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html><body>
			<audio>
				<source src="https://example.com/audio.mp3" type="audio/mpeg">
				<source src="https://example.com/audio.ogg" type="audio/ogg">
			</audio>
		</body></html>
	`

	p := html.NewWithDefaults()
	defer p.Close()

	result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	if len(result.Audios) == 0 {
		t.Error("Extract() returned no audio files, want at least 1")
	}

	if result.Audios[0].URL != "https://example.com/audio.mp3" {
		t.Errorf("Extract() audio URL = %q, want 'https://example.com/audio.mp3'", result.Audios[0].URL)
	}
}

// TestImageWithAttributes tests image extraction with various attributes
func TestImageWithAttributes(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html><body>
			<img src="https://example.com/image.jpg" alt="Test Image" title="Image Title" width="800" height="600">
		</body></html>
	`

	p := html.NewWithDefaults()
	defer p.Close()

	result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	if len(result.Images) == 0 {
		t.Fatal("Extract() returned no images, want at least 1")
	}

	img := result.Images[0]
	if img.URL != "https://example.com/image.jpg" {
		t.Errorf("Extract() image URL = %q, want 'https://example.com/image.jpg'", img.URL)
	}
	if img.Alt != "Test Image" {
		t.Errorf("Extract() image Alt = %q, want 'Test Image'", img.Alt)
	}
	if img.Title != "Image Title" {
		t.Errorf("Extract() image Title = %q, want 'Image Title'", img.Title)
	}
	if img.Width != "800" {
		t.Errorf("Extract() image Width = %q, want '800'", img.Width)
	}
	if img.Height != "600" {
		t.Errorf("Extract() image Height = %q, want '600'", img.Height)
	}
}

// TestLinkWithNofollow tests link extraction with rel="nofollow"
func TestLinkWithNofollow(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html><body>
			<a href="https://example.com" rel="nofollow">Example Link</a>
		</body></html>
	`

	p := html.NewWithDefaults()
	defer p.Close()

	result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	if len(result.Links) == 0 {
		t.Fatal("Extract() returned no links, want at least 1")
	}

	if !result.Links[0].IsNoFollow {
		t.Error("Extract() link IsNoFollow = false, want true")
	}
}

// TestExternalLinkDetection tests external link detection
func TestExternalLinkDetection(t *testing.T) {
	t.Parallel()

	htmlContent := `
		<html><body>
			<a href="https://example.com">External Link</a>
			<a href="/internal">Internal Link</a>
		</body></html>
	`

	p := html.NewWithDefaults()
	defer p.Close()

	result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	if len(result.Links) < 2 {
		t.Fatalf("Extract() returned %d links, want at least 2", len(result.Links))
	}

	// First link should be external
	if !result.Links[0].IsExternal {
		t.Error("Extract() first link IsExternal = false, want true")
	}

	// Second link should be internal
	if result.Links[1].IsExternal {
		t.Error("Extract() second link IsExternal = true, want false")
	}
}

// BenchmarkIframeExtraction benchmarks iframe extraction
func BenchmarkIframeExtraction(b *testing.B) {
	htmlContent := `
		<html><body>
			<iframe src="https://www.youtube.com/embed/test123" width="640" height="480"></iframe>
		</body></html>
	`

	p := html.NewWithDefaults()
	defer p.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.Extract(htmlContent, html.DefaultExtractConfig())
	}
}

// BenchmarkEmbedExtraction benchmarks embed extraction
func BenchmarkEmbedExtraction(b *testing.B) {
	htmlContent := `
		<html><body>
			<embed src="https://www.youtube.com/embed/test123" type="video/mp4" width="640" height="480">
		</body></html>
	`

	p := html.NewWithDefaults()
	defer p.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.Extract(htmlContent, html.DefaultExtractConfig())
	}
}
