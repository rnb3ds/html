package html_test

import (
	"testing"
	"time"

	"github.com/cybergodev/html"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	config := html.DefaultConfig()

	if config.MaxInputSize <= 0 {
		t.Error("DefaultConfig() MaxInputSize should be positive")
	}
	if config.MaxCacheEntries < 0 {
		t.Error("DefaultConfig() MaxCacheEntries should be non-negative")
	}
	if config.CacheTTL < 0 {
		t.Error("DefaultConfig() CacheTTL should be non-negative")
	}
	if config.WorkerPoolSize <= 0 {
		t.Error("DefaultConfig() WorkerPoolSize should be positive")
	}
	if !config.EnableSanitization {
		t.Error("DefaultConfig() should enable sanitization by default")
	}
	if config.MaxDepth <= 0 {
		t.Error("DefaultConfig() MaxDepth should be positive")
	}
}

func TestDefaultExtractConfig(t *testing.T) {
	t.Parallel()

	config := html.DefaultExtractConfig()

	if !config.ExtractArticle {
		t.Error("DefaultExtractConfig() should enable article extraction")
	}
	if !config.PreserveImages {
		t.Error("DefaultExtractConfig() should preserve images")
	}
	if !config.PreserveLinks {
		t.Error("DefaultExtractConfig() should preserve links")
	}
	if !config.PreserveVideos {
		t.Error("DefaultExtractConfig() should preserve videos")
	}
	if !config.PreserveAudios {
		t.Error("DefaultExtractConfig() should preserve audios")
	}
	if config.InlineImageFormat != "none" {
		t.Errorf("DefaultExtractConfig() InlineImageFormat = %q, want %q", config.InlineImageFormat, "none")
	}
}

func TestConfigValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  html.Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  html.DefaultConfig(),
			wantErr: false,
		},
		{
			name: "zero MaxInputSize",
			config: html.Config{
				MaxInputSize:       0,
				MaxCacheEntries:    100,
				CacheTTL:           time.Hour,
				WorkerPoolSize:     4,
				EnableSanitization: true,
				MaxDepth:           100,
			},
			wantErr: true,
		},
		{
			name: "negative MaxInputSize",
			config: html.Config{
				MaxInputSize:       -1,
				MaxCacheEntries:    100,
				CacheTTL:           time.Hour,
				WorkerPoolSize:     4,
				EnableSanitization: true,
				MaxDepth:           100,
			},
			wantErr: true,
		},
		{
			name: "negative MaxCacheEntries",
			config: html.Config{
				MaxInputSize:       1024,
				MaxCacheEntries:    -1,
				CacheTTL:           time.Hour,
				WorkerPoolSize:     4,
				EnableSanitization: true,
				MaxDepth:           100,
			},
			wantErr: true,
		},
		{
			name: "zero MaxCacheEntries (disabled cache)",
			config: html.Config{
				MaxInputSize:       1024,
				MaxCacheEntries:    0,
				CacheTTL:           time.Hour,
				WorkerPoolSize:     4,
				EnableSanitization: true,
				MaxDepth:           100,
			},
			wantErr: false,
		},
		{
			name: "negative CacheTTL",
			config: html.Config{
				MaxInputSize:       1024,
				MaxCacheEntries:    100,
				CacheTTL:           -1,
				WorkerPoolSize:     4,
				EnableSanitization: true,
				MaxDepth:           100,
			},
			wantErr: true,
		},
		{
			name: "zero CacheTTL (no expiration)",
			config: html.Config{
				MaxInputSize:       1024,
				MaxCacheEntries:    100,
				CacheTTL:           0,
				WorkerPoolSize:     4,
				EnableSanitization: true,
				MaxDepth:           100,
			},
			wantErr: false,
		},
		{
			name: "zero WorkerPoolSize",
			config: html.Config{
				MaxInputSize:       1024,
				MaxCacheEntries:    100,
				CacheTTL:           time.Hour,
				WorkerPoolSize:     0,
				EnableSanitization: true,
				MaxDepth:           100,
			},
			wantErr: true,
		},
		{
			name: "negative WorkerPoolSize",
			config: html.Config{
				MaxInputSize:       1024,
				MaxCacheEntries:    100,
				CacheTTL:           time.Hour,
				WorkerPoolSize:     -1,
				EnableSanitization: true,
				MaxDepth:           100,
			},
			wantErr: true,
		},
		{
			name: "zero MaxDepth",
			config: html.Config{
				MaxInputSize:       1024,
				MaxCacheEntries:    100,
				CacheTTL:           time.Hour,
				WorkerPoolSize:     4,
				EnableSanitization: true,
				MaxDepth:           0,
			},
			wantErr: true,
		},
		{
			name: "negative MaxDepth",
			config: html.Config{
				MaxInputSize:       1024,
				MaxCacheEntries:    100,
				CacheTTL:           time.Hour,
				WorkerPoolSize:     4,
				EnableSanitization: true,
				MaxDepth:           -1,
			},
			wantErr: true,
		},
		{
			name: "sanitization disabled",
			config: html.Config{
				MaxInputSize:       1024,
				MaxCacheEntries:    100,
				CacheTTL:           time.Hour,
				WorkerPoolSize:     4,
				EnableSanitization: false,
				MaxDepth:           100,
			},
			wantErr: false,
		},
		{
			name: "negative ProcessingTimeout",
			config: html.Config{
				MaxInputSize:       1024,
				MaxCacheEntries:    100,
				CacheTTL:           time.Hour,
				WorkerPoolSize:     4,
				EnableSanitization: true,
				MaxDepth:           100,
				ProcessingTimeout:  -1,
			},
			wantErr: true,
		},
		{
			name: "zero ProcessingTimeout (no timeout)",
			config: html.Config{
				MaxInputSize:       1024,
				MaxCacheEntries:    100,
				CacheTTL:           time.Hour,
				WorkerPoolSize:     4,
				EnableSanitization: true,
				MaxDepth:           100,
				ProcessingTimeout:  0,
			},
			wantErr: false,
		},
		{
			name: "valid ProcessingTimeout",
			config: html.Config{
				MaxInputSize:       1024,
				MaxCacheEntries:    100,
				CacheTTL:           time.Hour,
				WorkerPoolSize:     4,
				EnableSanitization: true,
				MaxDepth:           100,
				ProcessingTimeout:  30 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := html.New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCustomConfig(t *testing.T) {
	t.Parallel()

	config := html.Config{
		MaxInputSize:       1024 * 1024,
		MaxCacheEntries:    50,
		CacheTTL:           30 * time.Minute,
		WorkerPoolSize:     8,
		EnableSanitization: false,
		MaxDepth:           50,
	}

	p, err := html.New(config)
	if err != nil {
		t.Fatalf("New() with custom config failed: %v", err)
	}
	defer p.Close()

	htmlContent := `<html><body><p>Test</p></body></html>`
	_, err = p.Extract(htmlContent, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Extract() with custom config failed: %v", err)
	}
}

func TestExtractConfigVariations(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

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
	}{
		{
			name: "all enabled",
			config: html.ExtractConfig{
				ExtractArticle:    true,
				PreserveImages:    true,
				PreserveLinks:     true,
				PreserveVideos:    true,
				PreserveAudios:    true,
				InlineImageFormat: "none",
			},
		},
		{
			name: "all disabled",
			config: html.ExtractConfig{
				ExtractArticle:    false,
				PreserveImages:    false,
				PreserveLinks:     false,
				PreserveVideos:    false,
				PreserveAudios:    false,
				InlineImageFormat: "none",
			},
		},
		{
			name: "only images",
			config: html.ExtractConfig{
				ExtractArticle:    false,
				PreserveImages:    true,
				PreserveLinks:     false,
				PreserveVideos:    false,
				PreserveAudios:    false,
				InlineImageFormat: "none",
			},
		},
		{
			name: "markdown images",
			config: html.ExtractConfig{
				ExtractArticle:    true,
				PreserveImages:    true,
				PreserveLinks:     true,
				PreserveVideos:    true,
				PreserveAudios:    true,
				InlineImageFormat: "markdown",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Extract(htmlContent, tt.config)
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if tt.config.PreserveImages && len(result.Images) == 0 {
				t.Error("Expected images to be extracted")
			}
			if !tt.config.PreserveImages && len(result.Images) > 0 {
				t.Error("Expected no images to be extracted")
			}

			if tt.config.PreserveLinks && len(result.Links) == 0 {
				t.Error("Expected links to be extracted")
			}
			if !tt.config.PreserveLinks && len(result.Links) > 0 {
				t.Error("Expected no links to be extracted")
			}

			if tt.config.PreserveVideos && len(result.Videos) == 0 {
				t.Error("Expected videos to be extracted")
			}
			if !tt.config.PreserveVideos && len(result.Videos) > 0 {
				t.Error("Expected no videos to be extracted")
			}

			if tt.config.PreserveAudios && len(result.Audios) == 0 {
				t.Error("Expected audios to be extracted")
			}
			if !tt.config.PreserveAudios && len(result.Audios) > 0 {
				t.Error("Expected no audios to be extracted")
			}
		})
	}
}
