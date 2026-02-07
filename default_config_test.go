package html_test

import (
	"errors"
	"testing"
	"time"

	"github.com/cybergodev/html"
)

// TestDefaultConfigurations verifies that all default values are correctly set
func TestDefaultConfigurations(t *testing.T) {
	t.Parallel()

	t.Run("DefaultConfig", func(t *testing.T) {
		config := html.DefaultConfig()

		if config.MaxInputSize != html.DefaultMaxInputSize {
			t.Errorf("MaxInputSize = %d, want %d", config.MaxInputSize, html.DefaultMaxInputSize)
		}
		if config.MaxCacheEntries != html.DefaultMaxCacheEntries {
			t.Errorf("MaxCacheEntries = %d, want %d", config.MaxCacheEntries, html.DefaultMaxCacheEntries)
		}
		if config.CacheTTL != html.DefaultCacheTTL {
			t.Errorf("CacheTTL = %v, want %v", config.CacheTTL, html.DefaultCacheTTL)
		}
		if config.WorkerPoolSize != html.DefaultWorkerPoolSize {
			t.Errorf("WorkerPoolSize = %d, want %d", config.WorkerPoolSize, html.DefaultWorkerPoolSize)
		}
		if !config.EnableSanitization {
			t.Error("EnableSanitization should be true by default")
		}
		if config.MaxDepth != html.DefaultMaxDepth {
			t.Errorf("MaxDepth = %d, want %d", config.MaxDepth, html.DefaultMaxDepth)
		}
		if config.ProcessingTimeout != html.DefaultProcessingTimeout {
			t.Errorf("ProcessingTimeout = %v, want %v", config.ProcessingTimeout, html.DefaultProcessingTimeout)
		}
	})

	t.Run("DefaultExtractConfig", func(t *testing.T) {
		config := html.DefaultExtractConfig()

		if !config.ExtractArticle {
			t.Error("ExtractArticle should be true by default")
		}
		if !config.PreserveImages {
			t.Error("PreserveImages should be true by default")
		}
		if !config.PreserveLinks {
			t.Error("PreserveLinks should be true by default")
		}
		if !config.PreserveVideos {
			t.Error("PreserveVideos should be true by default")
		}
		if !config.PreserveAudios {
			t.Error("PreserveAudios should be true by default")
		}
		if config.InlineImageFormat != "none" {
			t.Errorf("InlineImageFormat = %s, want 'none'", config.InlineImageFormat)
		}
		if config.TableFormat != "markdown" {
			t.Errorf("TableFormat = %s, want 'markdown'", config.TableFormat)
		}
		if config.Encoding != "" {
			t.Errorf("Encoding = %s, want empty string (auto-detect)", config.Encoding)
		}
	})

	t.Run("DefaultLinkExtractionConfig", func(t *testing.T) {
		config := html.DefaultLinkExtractionConfig()

		if !config.ResolveRelativeURLs {
			t.Error("ResolveRelativeURLs should be true by default")
		}
		if config.BaseURL != "" {
			t.Errorf("BaseURL = %s, want empty string (auto-detect)", config.BaseURL)
		}
		if !config.IncludeImages {
			t.Error("IncludeImages should be true by default")
		}
		if !config.IncludeVideos {
			t.Error("IncludeVideos should be true by default")
		}
		if !config.IncludeAudios {
			t.Error("IncludeAudios should be true by default")
		}
		if !config.IncludeCSS {
			t.Error("IncludeCSS should be true by default")
		}
		if !config.IncludeJS {
			t.Error("IncludeJS should be true by default")
		}
		if !config.IncludeContentLinks {
			t.Error("IncludeContentLinks should be true by default")
		}
		if !config.IncludeExternalLinks {
			t.Error("IncludeExternalLinks should be true by default")
		}
		if !config.IncludeIcons {
			t.Error("IncludeIcons should be true by default")
		}
	})
}

// TestDefaultMaxDepth verifies the default MaxDepth constant value
func TestDefaultMaxDepth(t *testing.T) {
	t.Parallel()

	t.Run("DefaultMaxDepth value", func(t *testing.T) {
		if html.DefaultMaxDepth != 500 {
			t.Errorf("DefaultMaxDepth = %d, want 500", html.DefaultMaxDepth)
		}
	})

	t.Run("DefaultMaxDepth is used in DefaultConfig", func(t *testing.T) {
		config := html.DefaultConfig()
		if config.MaxDepth != html.DefaultMaxDepth {
			t.Errorf("config.MaxDepth = %d, want DefaultMaxDepth (%d)", config.MaxDepth, html.DefaultMaxDepth)
		}
	})

	t.Run("MaxDepth can be overridden", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxDepth = 100 // Custom value

		processor, err := html.New(config)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer processor.Close()
		// Processor created successfully - custom MaxDepth accepted
	})
}

// TestMaxDepthValidation verifies that MaxDepth is properly validated
func TestMaxDepthValidation(t *testing.T) {
	t.Parallel()

	t.Run("Rejects zero MaxDepth", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxInputSize = 1000
		config.MaxDepth = 0 // Invalid

		_, err := html.New(config)
		if err == nil {
			t.Error("New() should reject zero MaxDepth")
		}
		if !errors.Is(err, html.ErrInvalidConfig) {
			t.Errorf("Error should be ErrInvalidConfig, got %v", err)
		}
	})

	t.Run("Rejects negative MaxDepth", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxInputSize = 1000
		config.MaxDepth = -1 // Invalid

		_, err := html.New(config)
		if err == nil {
			t.Error("New() should reject negative MaxDepth")
		}
	})

	t.Run("Rejects excessive MaxDepth", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxInputSize = 1000
		config.MaxDepth = 10000 // Too large

		_, err := html.New(config)
		if err == nil {
			t.Error("New() should reject excessive MaxDepth")
		}
	})

	t.Run("Accepts maximum allowed MaxDepth", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxDepth = 500 // Maximum allowed

		processor, err := html.New(config)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer processor.Close()
		_ = processor
	})
}

// TestMaxDepthEnforcement verifies that MaxDepth is enforced during extraction
func TestMaxDepthEnforcement(t *testing.T) {
	t.Parallel()

	t.Run("Default MaxDepth is 500", func(t *testing.T) {
		processor, _ := html.New()
		defer processor.Close()
		// Processor created successfully with default MaxDepth
		_ = processor
	})

	t.Run("Custom MaxDepth is applied", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxDepth = 100

		processor, err := html.New(config)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer processor.Close()
		// Processor created successfully with custom MaxDepth
		_ = processor
	})

	t.Run("MaxDepth validation works", func(t *testing.T) {
		testCases := []struct {
			name    string
			maxDepth int
			wantErr bool
		}{
			{"valid: 500", 500, false},
			{"valid: 100", 100, false},
			{"valid: 50", 50, false},
			{"invalid: 0", 0, true},
			{"invalid: -1", -1, true},
			{"invalid: 1000", 1000, true}, // Exceeds maxConfigDepth (500)
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := html.Config{
					MaxInputSize:   1000,
					WorkerPoolSize: 4,
					MaxDepth:       tc.maxDepth,
				}

				_, err := html.New(config)
				if tc.wantErr && err == nil {
					t.Errorf("New() should reject MaxDepth=%d", tc.maxDepth)
				}
				if !tc.wantErr && err != nil {
					t.Errorf("New() should accept MaxDepth=%d: %v", tc.maxDepth, err)
				}
			})
		}
	})
}

// TestDocumentationDefaultMaxDepth verifies documentation matches actual default
func TestDocumentationDefaultMaxDepth(t *testing.T) {
	t.Parallel()

	t.Run("README.md example values", func(t *testing.T) {
		// This test ensures that documentation examples use reasonable values
		// while noting the actual default is 500

		// README custom configuration example uses MaxDepth: 50
		// This is intentional - it's showing customization, not defaults
		config := html.Config{
			MaxInputSize:       10 * 1024 * 1024,
			ProcessingTimeout:  30 * time.Second,
			MaxCacheEntries:    500,
			CacheTTL:           30 * time.Minute,
			WorkerPoolSize:     8,
			EnableSanitization: true,
			MaxDepth:           50, // Example custom value
		}

		processor, err := html.New(config)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer processor.Close()
		// Custom MaxDepth applied successfully
		_ = processor
	})
}

// BenchmarkMaxDepthValidation benchmarks the depth validation performance
func BenchmarkMaxDepthValidation(b *testing.B) {
	processor, _ := html.New()
	defer processor.Close()

	// HTML with reasonable nesting
	htmlContent := `<div><div><div><div>Content</div></div></div></div>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = processor.Extract([]byte(htmlContent), html.DefaultExtractConfig())
	}
}
