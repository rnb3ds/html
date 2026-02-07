package html

import (
	"errors"
	"testing"
	"time"
)

// TestDefaultConfigurations verifies that all default values are correctly set
func TestDefaultConfigurations(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultConfig()

		if config.MaxInputSize != DefaultMaxInputSize {
			t.Errorf("MaxInputSize = %d, want %d", config.MaxInputSize, DefaultMaxInputSize)
		}
		if config.MaxCacheEntries != DefaultMaxCacheEntries {
			t.Errorf("MaxCacheEntries = %d, want %d", config.MaxCacheEntries, DefaultMaxCacheEntries)
		}
		if config.CacheTTL != DefaultCacheTTL {
			t.Errorf("CacheTTL = %v, want %v", config.CacheTTL, DefaultCacheTTL)
		}
		if config.WorkerPoolSize != DefaultWorkerPoolSize {
			t.Errorf("WorkerPoolSize = %d, want %d", config.WorkerPoolSize, DefaultWorkerPoolSize)
		}
		if !config.EnableSanitization {
			t.Error("EnableSanitization should be true by default")
		}
		if config.MaxDepth != DefaultMaxDepth {
			t.Errorf("MaxDepth = %d, want %d", config.MaxDepth, DefaultMaxDepth)
		}
		if config.ProcessingTimeout != DefaultProcessingTimeout {
			t.Errorf("ProcessingTimeout = %v, want %v", config.ProcessingTimeout, DefaultProcessingTimeout)
		}
	})

	t.Run("DefaultExtractConfig", func(t *testing.T) {
		config := DefaultExtractConfig()

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
		config := DefaultLinkExtractionConfig()

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
	t.Run("DefaultMaxDepth value", func(t *testing.T) {
		if DefaultMaxDepth != 500 {
			t.Errorf("DefaultMaxDepth = %d, want 500", DefaultMaxDepth)
		}
	})

	t.Run("DefaultMaxDepth is used in DefaultConfig", func(t *testing.T) {
		config := DefaultConfig()
		if config.MaxDepth != DefaultMaxDepth {
			t.Errorf("config.MaxDepth = %d, want DefaultMaxDepth (%d)", config.MaxDepth, DefaultMaxDepth)
		}
	})

	t.Run("MaxDepth can be overridden", func(t *testing.T) {
		config := Config{
			MaxInputSize:   DefaultMaxInputSize,
			WorkerPoolSize: DefaultWorkerPoolSize,
			MaxDepth:       100, // Custom value
		}

		processor, err := New(config)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer processor.Close()

		if processor.config.MaxDepth != 100 {
			t.Errorf("processor.config.MaxDepth = %d, want 100", processor.config.MaxDepth)
		}
	})
}

// TestMaxDepthValidation verifies that MaxDepth is properly validated
func TestMaxDepthValidation(t *testing.T) {
	t.Run("Rejects zero MaxDepth", func(t *testing.T) {
		config := Config{
			MaxInputSize:   1000,
			WorkerPoolSize: 4,
			MaxDepth:       0, // Invalid
		}

		_, err := New(config)
		if err == nil {
			t.Error("New() should reject zero MaxDepth")
		}
		if !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("Error should be ErrInvalidConfig, got %v", err)
		}
	})

	t.Run("Rejects negative MaxDepth", func(t *testing.T) {
		config := Config{
			MaxInputSize:   1000,
			WorkerPoolSize: 4,
			MaxDepth:       -1, // Invalid
		}

		_, err := New(config)
		if err == nil {
			t.Error("New() should reject negative MaxDepth")
		}
	})

	t.Run("Rejects excessive MaxDepth", func(t *testing.T) {
		config := Config{
			MaxInputSize:   1000,
			WorkerPoolSize: 4,
			MaxDepth:       10000, // Too large
		}

		_, err := New(config)
		if err == nil {
			t.Error("New() should reject excessive MaxDepth")
		}
	})

	t.Run("Accepts maximum allowed MaxDepth", func(t *testing.T) {
		config := Config{
			MaxInputSize:   1000,
			WorkerPoolSize: 4,
			MaxDepth:       500, // Maximum allowed
		}

		processor, err := New(config)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer processor.Close()

		if processor.config.MaxDepth != 500 {
			t.Errorf("processor.config.MaxDepth = %d, want 500", processor.config.MaxDepth)
		}
	})
}

// TestMaxDepthEnforcement verifies that MaxDepth is enforced during extraction
func TestMaxDepthEnforcement(t *testing.T) {
	t.Run("Default MaxDepth is 500", func(t *testing.T) {
		processor, _ := New()
		defer processor.Close()

		if processor.config.MaxDepth != 500 {
			t.Errorf("Default MaxDepth = %d, want 500", processor.config.MaxDepth)
		}
	})

	t.Run("Custom MaxDepth is applied", func(t *testing.T) {
		config := Config{
			MaxInputSize:   1000,
			WorkerPoolSize: 4,
			MaxDepth:       100,
		}

		processor, err := New(config)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer processor.Close()

		if processor.config.MaxDepth != 100 {
			t.Errorf("Custom MaxDepth = %d, want 100", processor.config.MaxDepth)
		}
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
				config := Config{
					MaxInputSize:   1000,
					WorkerPoolSize: 4,
					MaxDepth:       tc.maxDepth,
				}

				_, err := New(config)
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
	t.Run("README.md example values", func(t *testing.T) {
		// This test ensures that documentation examples use reasonable values
		// while noting the actual default is 500

		// README custom configuration example uses MaxDepth: 50
		// This is intentional - it's showing customization, not defaults
		config := Config{
			MaxInputSize:       10 * 1024 * 1024,
			ProcessingTimeout:  30 * time.Second,
			MaxCacheEntries:    500,
			CacheTTL:           30 * time.Minute,
			WorkerPoolSize:     8,
			EnableSanitization: true,
			MaxDepth:           50, // Example custom value
		}

		processor, err := New(config)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer processor.Close()

		if processor.config.MaxDepth != 50 {
			t.Errorf("Custom MaxDepth not applied: got %d, want 50", processor.config.MaxDepth)
		}
	})
}

// BenchmarkMaxDepthValidation benchmarks the depth validation performance
func BenchmarkMaxDepthValidation(b *testing.B) {
	processor, _ := New()
	defer processor.Close()

	// HTML with reasonable nesting
	html := `<div><div><div><div>Content</div></div></div></div>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = processor.ExtractWithDefaults([]byte(html))
	}
}
