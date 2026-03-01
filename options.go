package html

import (
	"fmt"
	"time"
)

// Option is a functional option for configuring Processor.
type Option func(*Config) error

// WithMaxInputSize sets the maximum input size in bytes.
func WithMaxInputSize(size int) Option {
	return func(c *Config) error {
		if size <= 0 {
			return NewConfigError("MaxInputSize", size, "must be positive")
		}
		if size > maxConfigInputSize {
			return NewConfigError("MaxInputSize", size, fmt.Sprintf("exceeds maximum %d", maxConfigInputSize))
		}
		c.MaxInputSize = size
		return nil
	}
}

// WithMaxCacheEntries sets the maximum number of cache entries.
func WithMaxCacheEntries(entries int) Option {
	return func(c *Config) error {
		if entries < 0 {
			return NewConfigError("MaxCacheEntries", entries, "cannot be negative")
		}
		c.MaxCacheEntries = entries
		return nil
	}
}

// WithCacheTTL sets the cache time-to-live duration.
func WithCacheTTL(ttl time.Duration) Option {
	return func(c *Config) error {
		if ttl < 0 {
			return NewConfigError("CacheTTL", ttl, "cannot be negative")
		}
		c.CacheTTL = ttl
		return nil
	}
}

// WithWorkerPoolSize sets the worker pool size for concurrent processing.
func WithWorkerPoolSize(size int) Option {
	return func(c *Config) error {
		if size <= 0 {
			return NewConfigError("WorkerPoolSize", size, "must be positive")
		}
		if size > maxConfigWorkerSize {
			return NewConfigError("WorkerPoolSize", size, fmt.Sprintf("exceeds maximum %d", maxConfigWorkerSize))
		}
		c.WorkerPoolSize = size
		return nil
	}
}

// WithSanitization enables or disables HTML sanitization.
func WithSanitization(enabled bool) Option {
	return func(c *Config) error {
		c.EnableSanitization = enabled
		return nil
	}
}

// WithMaxDepth sets the maximum DOM tree depth.
func WithMaxDepth(depth int) Option {
	return func(c *Config) error {
		if depth <= 0 {
			return NewConfigError("MaxDepth", depth, "must be positive")
		}
		if depth > maxConfigDepth {
			return NewConfigError("MaxDepth", depth, fmt.Sprintf("exceeds maximum %d", maxConfigDepth))
		}
		c.MaxDepth = depth
		return nil
	}
}

// WithProcessingTimeout sets the processing timeout duration.
func WithProcessingTimeout(timeout time.Duration) Option {
	return func(c *Config) error {
		if timeout < 0 {
			return NewConfigError("ProcessingTimeout", timeout, "cannot be negative")
		}
		c.ProcessingTimeout = timeout
		return nil
	}
}

// WithCache is a convenience option that sets both cache entries and TTL.
func WithCache(entries int, ttl time.Duration) Option {
	return func(c *Config) error {
		if entries < 0 {
			return NewConfigError("MaxCacheEntries", entries, "cannot be negative")
		}
		if ttl < 0 {
			return NewConfigError("CacheTTL", ttl, "cannot be negative")
		}
		c.MaxCacheEntries = entries
		c.CacheTTL = ttl
		return nil
	}
}

// ExtractOption is a functional option for extraction configuration.
type ExtractOption func(*ExtractConfig)

// WithArticleExtraction enables or disables article extraction.
func WithArticleExtraction(enabled bool) ExtractOption {
	return func(c *ExtractConfig) {
		c.ExtractArticle = enabled
	}
}

// WithImagePreservation enables or disables image preservation.
func WithImagePreservation(enabled bool) ExtractOption {
	return func(c *ExtractConfig) {
		c.PreserveImages = enabled
	}
}

// WithLinkPreservation enables or disables link preservation.
func WithLinkPreservation(enabled bool) ExtractOption {
	return func(c *ExtractConfig) {
		c.PreserveLinks = enabled
	}
}

// WithVideoPreservation enables or disables video preservation.
func WithVideoPreservation(enabled bool) ExtractOption {
	return func(c *ExtractConfig) {
		c.PreserveVideos = enabled
	}
}

// WithAudioPreservation enables or disables audio preservation.
func WithAudioPreservation(enabled bool) ExtractOption {
	return func(c *ExtractConfig) {
		c.PreserveAudios = enabled
	}
}

// WithInlineImageFormat sets the format for inline images ("markdown", "html", "placeholder", "none").
func WithInlineImageFormat(format string) ExtractOption {
	return func(c *ExtractConfig) {
		c.InlineImageFormat = format
	}
}

// WithTableFormat sets the format for table output ("markdown", "html").
func WithTableFormat(format string) ExtractOption {
	return func(c *ExtractConfig) {
		c.TableFormat = format
	}
}

// WithEncoding specifies the character encoding of the input HTML.
func WithEncoding(encoding string) ExtractOption {
	return func(c *ExtractConfig) {
		c.Encoding = encoding
	}
}

// TextOnly returns a preset ExtractOption slice for extracting plain text only.
// Disables all media preservation and uses no inline image format.
func TextOnly() []ExtractOption {
	return []ExtractOption{
		WithArticleExtraction(true),
		WithImagePreservation(false),
		WithLinkPreservation(false),
		WithVideoPreservation(false),
		WithAudioPreservation(false),
		WithInlineImageFormat("none"),
	}
}

// FullContent returns a preset ExtractOption slice for extracting all content.
// Enables all media preservation and uses markdown format for inline images.
func FullContent() []ExtractOption {
	return []ExtractOption{
		WithArticleExtraction(true),
		WithImagePreservation(true),
		WithLinkPreservation(true),
		WithVideoPreservation(true),
		WithAudioPreservation(true),
		WithInlineImageFormat("markdown"),
	}
}

// applyExtractOptions applies multiple ExtractOption functions to an ExtractConfig.
func applyExtractOptions(config *ExtractConfig, opts ...ExtractOption) {
	for _, opt := range opts {
		opt(config)
	}
}
