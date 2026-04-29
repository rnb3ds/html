package html

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Default configuration values.
const (
	DefaultMaxInputSize      = 50 * 1024 * 1024 // 50MB
	DefaultMaxCacheEntries   = 2000
	DefaultWorkerPoolSize    = 4
	DefaultCacheTTL          = time.Hour
	DefaultCacheCleanup      = 5 * time.Minute // Background cleanup interval for expired cache entries
	DefaultMaxDepth          = 500
	DefaultProcessingTimeout = 30 * time.Second
)

// Configuration limits - reference Default* constants for consistency
const (
	// maxConfigInputSize limits the maximum allowed MaxInputSize configuration.
	// This matches DefaultMaxInputSize (50MB) to prevent memory exhaustion.
	maxConfigInputSize = DefaultMaxInputSize

	// maxConfigWorkerSize limits the maximum worker pool size.
	// Value 256 prevents excessive goroutine creation while allowing
	// high-throughput batch processing on powerful machines.
	maxConfigWorkerSize = 256

	// maxConfigDepth limits the maximum HTML nesting depth.
	// This matches DefaultMaxDepth (500) to prevent stack overflow.
	maxConfigDepth = DefaultMaxDepth

	// maxConfigCacheEntries limits the maximum number of cache entries.
	// Value 100,000 entries ≈ 100MB assuming 1KB average entry size.
	maxConfigCacheEntries = 100000

	// Processing limits
	// maxHTMLForRegex limits HTML size for regex-based media URL detection.
	// Above 1MB, regex operations become slow and could cause ReDoS.
	maxHTMLForRegex = 1000000

	// maxRegexMatches limits the number of regex matches to prevent
	// excessive memory allocation on HTML with many media URLs.
	maxRegexMatches = 1000

	// maxCacheKeySize limits the content size used for cache key generation.
	// For content larger than 64KB, multi-point sampling is used instead.
	maxCacheKeySize = 64 * 1024

	// cacheKeySample is the sample size for cache key generation on large content.
	cacheKeySample = 4096

	// Buffer size estimates for pre-allocation
	initialTextSize   = 4096 // Initial capacity for text builder
	initialSliceCap   = 16   // Initial capacity for result slices
	initialMapCap     = 8    // Initial capacity for result maps
	imageHTMLBufExtra = 64   // Extra buffer for HTML image tag generation
	extractTagCap     = 16   // Initial capacity for tag attribute extraction
	linkMapCap        = 64   // Initial capacity for link deduplication map

	// Processing thresholds
	wordsPerMinute = 200 // Average reading speed for reading time estimation
)

// Pre-compiled regex patterns for media URL detection.
var (
	videoRegex = regexp.MustCompile(`(?i)https?://[^\s<>"',;)}\]]{1,500}\.(?:mp4|webm|ogg|mov|avi|wmv|flv|mkv|m4v|3gp)`)
	audioRegex = regexp.MustCompile(`(?i)https?://[^\s<>"',;)}\]]{1,500}\.(?:mp3|wav|ogg|m4a|aac|flac|wma|opus|oga)`)
)

// ============================================================================
// Flat Configuration Structure
// ============================================================================

// Config is the unified configuration for the HTML processor.
// All configuration options are flat for ease of use.
// Zero-value is not usable; start from DefaultConfig().
//
// Example:
//
//	cfg := html.DefaultConfig()
//	cfg.MaxInputSize = 10 * 1024 * 1024
//	cfg.InlineImageFormat = "markdown"
//	cfg.IncludeJS = false
//	processor, err := html.New(cfg)
type Config struct {
	// === Resource Management ===
	MaxInputSize      int           // Maximum HTML input size in bytes. Default: 50MB. Must be positive and <= 50MB.
	MaxCacheEntries   int           // Maximum number of cache entries. Set to 0 to disable caching. Default: 2000.
	CacheTTL          time.Duration // Time-to-live for cache entries. Default: 1 hour.
	CacheCleanup      time.Duration // Interval for background cleanup of expired cache entries. Set to 0 to disable. Default: 5 minutes.
	WorkerPoolSize    int           // Number of concurrent workers for batch processing. Default: 4. Must be positive and <= 256.
	ProcessingTimeout time.Duration // Maximum time allowed for processing a single document. Default: 30 seconds. Set to 0 for no timeout.

	// === Security ===
	EnableSanitization bool        // Controls whether HTML sanitization is applied. Default: true. Should only be disabled for trusted input.
	MaxDepth           int         // Maximum allowed nesting depth of HTML elements. Prevents stack overflow. Default: 500.
	Audit              AuditConfig // Security audit logging configuration.

	// === Content Extraction ===
	ExtractArticle bool // Enables article extraction mode. When true, identifies and extracts main content. Default: true.
	PreserveImages bool // Controls whether images are preserved in output. Default: true.
	PreserveLinks  bool // Controls whether links are preserved in output. Default: true.
	PreserveVideos bool // Controls whether video elements are extracted. Default: true.
	PreserveAudios bool // Controls whether audio elements are extracted. Default: true.

	// === Output Formats ===
	InlineImageFormat string // How images are formatted in text output. Options: "none", "markdown", "html", "placeholder". Default: "none".
	InlineLinkFormat  string // How links are formatted in text output. Options: "none", "markdown", "html". Default: "none".
	TableFormat       string // How tables are formatted in output. Options: "markdown", "html". Default: "markdown".
	Encoding          string // Character encoding of input HTML. Leave empty for auto-detection. Common: "utf-8", "windows-1252", "gbk".

	// === Link Extraction ===
	ResolveRelativeURLs  bool   // Controls whether relative URLs are resolved to absolute URLs. Requires BaseURL. Default: true.
	BaseURL              string // Base URL for resolving relative URLs. Example: "https://example.com"
	IncludeImages        bool   // Controls whether image URLs are included in link extraction. Default: true.
	IncludeVideos        bool   // Controls whether video URLs are included in link extraction. Default: true.
	IncludeAudios        bool   // Controls whether audio URLs are included in link extraction. Default: true.
	IncludeCSS           bool   // Controls whether CSS stylesheet URLs are included in link extraction. Default: true.
	IncludeJS            bool   // Controls whether JavaScript URLs are included in link extraction. Default: true.
	IncludeContentLinks  bool   // Controls whether content links (a[href]) are included. Default: true.
	IncludeExternalLinks bool   // Controls whether external links are included. Default: true.
	IncludeIcons         bool   // Controls whether favicon/icon URLs are included in link extraction. Default: true.

	// === Extension ===
	Scorer Scorer `json:"-"` // Optional custom scorer for content extraction. If nil, the default scorer is used.
}

// DefaultConfig returns a Config with all default values.
func DefaultConfig() Config {
	return Config{
		// Resource Management
		MaxInputSize:      DefaultMaxInputSize,
		MaxCacheEntries:   DefaultMaxCacheEntries,
		CacheTTL:          DefaultCacheTTL,
		CacheCleanup:      DefaultCacheCleanup,
		WorkerPoolSize:    DefaultWorkerPoolSize,
		ProcessingTimeout: DefaultProcessingTimeout,

		// Security
		EnableSanitization: true,
		MaxDepth:           DefaultMaxDepth,
		Audit:              DefaultAuditConfig(),

		// Content Extraction
		ExtractArticle: true,
		PreserveImages: true,
		PreserveLinks:  true,
		PreserveVideos: true,
		PreserveAudios: true,

		// Output Formats
		InlineImageFormat: "none",
		InlineLinkFormat:  "none",
		TableFormat:       "markdown",

		// Link Extraction
		ResolveRelativeURLs:  true,
		IncludeImages:        true,
		IncludeVideos:        true,
		IncludeAudios:        true,
		IncludeCSS:           true,
		IncludeJS:            true,
		IncludeContentLinks:  true,
		IncludeExternalLinks: true,
		IncludeIcons:         true,
	}
}

// Validate validates the configuration and returns an error if invalid.
func (c Config) Validate() error {
	switch {
	case c.MaxInputSize <= 0:
		return newConfigError("MaxInputSize", c.MaxInputSize, "must be positive")
	case c.MaxInputSize > maxConfigInputSize:
		return newConfigError("MaxInputSize", c.MaxInputSize, fmt.Sprintf("exceeds maximum %d", maxConfigInputSize))
	case c.MaxCacheEntries < 0:
		return newConfigError("MaxCacheEntries", c.MaxCacheEntries, "cannot be negative")
	case c.MaxCacheEntries > maxConfigCacheEntries:
		return newConfigError("MaxCacheEntries", c.MaxCacheEntries, fmt.Sprintf("exceeds maximum %d", maxConfigCacheEntries))
	case c.CacheTTL < 0:
		return newConfigError("CacheTTL", c.CacheTTL, "cannot be negative")
	case c.CacheCleanup < 0:
		return newConfigError("CacheCleanup", c.CacheCleanup, "cannot be negative")
	case c.WorkerPoolSize <= 0:
		return newConfigError("WorkerPoolSize", c.WorkerPoolSize, "must be positive")
	case c.WorkerPoolSize > maxConfigWorkerSize:
		return newConfigError("WorkerPoolSize", c.WorkerPoolSize, fmt.Sprintf("exceeds maximum %d", maxConfigWorkerSize))
	case c.MaxDepth <= 0:
		return newConfigError("MaxDepth", c.MaxDepth, "must be positive")
	case c.MaxDepth > maxConfigDepth:
		return newConfigError("MaxDepth", c.MaxDepth, fmt.Sprintf("exceeds maximum %d", maxConfigDepth))
	case c.ProcessingTimeout < 0:
		return newConfigError("ProcessingTimeout", c.ProcessingTimeout, "cannot be negative")
	}

	// Validate format strings
	if err := validateFormat("InlineImageFormat", c.InlineImageFormat, []string{"none", "markdown", "html", "placeholder"}); err != nil {
		return err
	}
	if err := validateFormat("InlineLinkFormat", c.InlineLinkFormat, []string{"none", "markdown", "html"}); err != nil {
		return err
	}
	if err := validateFormat("TableFormat", c.TableFormat, []string{"markdown", "html"}); err != nil {
		return err
	}

	return nil
}

// validateFormat validates that a format string is one of the allowed values.
// Empty string is allowed (will use default).
func validateFormat(field, value string, allowed []string) error {
	if value == "" {
		return nil // Empty means use default
	}
	lowerValue := strings.ToLower(value)
	for _, a := range allowed {
		if lowerValue == a {
			return nil
		}
	}
	return newConfigError(field, value, fmt.Sprintf("valid values: %s", strings.Join(allowed, ", ")))
}

// HighSecurityConfig returns a configuration optimized for high-security environments.
// This includes reduced limits, shorter timeouts, and comprehensive audit logging.
func HighSecurityConfig() Config {
	cfg := DefaultConfig()

	// Resource Management - reduced limits
	cfg.MaxInputSize = 10 * 1024 * 1024 // 10MB
	cfg.MaxCacheEntries = 500
	cfg.CacheTTL = 30 * time.Minute
	cfg.CacheCleanup = time.Minute
	cfg.WorkerPoolSize = 2
	cfg.ProcessingTimeout = 10 * time.Second

	// Security - strict settings
	cfg.MaxDepth = 100
	cfg.Audit = HighSecurityAuditConfig()

	return cfg
}

// TextOnlyConfig returns a configuration for extracting plain text only.
// This disables all media preservation for maximum performance.
func TextOnlyConfig() Config {
	cfg := DefaultConfig()

	// Content Extraction - disable all media
	cfg.PreserveImages = false
	cfg.PreserveLinks = false
	cfg.PreserveVideos = false
	cfg.PreserveAudios = false

	return cfg
}

// MarkdownConfig returns a configuration optimized for Markdown output.
// This enables markdown format for inline images and links.
func MarkdownConfig() Config {
	cfg := DefaultConfig()

	// Output Formats - enable markdown
	cfg.InlineImageFormat = "markdown"
	cfg.InlineLinkFormat = "markdown"

	return cfg
}

// ============================================================================
// Result Types
// ============================================================================

// Result holds the extraction result.
type Result struct {
	Text           string        `json:"text"`
	Title          string        `json:"title"`
	Images         []ImageInfo   `json:"images,omitempty"`
	Links          []LinkInfo    `json:"links,omitempty"`
	Videos         []VideoInfo   `json:"videos,omitempty"`
	Audios         []AudioInfo   `json:"audios,omitempty"`
	ProcessingTime time.Duration `json:"-"` // Serialized as processing_time_ms by MarshalJSON
	WordCount      int           `json:"word_count"`
	ReadingTime    time.Duration `json:"-"` // Serialized as reading_time_ms by MarshalJSON
}

// ImageInfo holds information about an extracted image.
type ImageInfo struct {
	URL          string `json:"url"`
	Alt          string `json:"alt"`
	Title        string `json:"title"`
	Width        string `json:"width"`
	Height       string `json:"height"`
	IsDecorative bool   `json:"is_decorative"`
	Position     int    `json:"position"`
}

// LinkInfo holds information about an extracted link.
type LinkInfo struct {
	URL        string `json:"url"`
	Text       string `json:"text"`
	Title      string `json:"title"`
	IsExternal bool   `json:"is_external"`
	IsNoFollow bool   `json:"is_nofollow"`
	Position   int    `json:"position"`
}

// VideoInfo holds information about an extracted video.
type VideoInfo struct {
	URL      string `json:"url"`
	Type     string `json:"type"`
	Poster   string `json:"poster"`
	Width    string `json:"width"`
	Height   string `json:"height"`
	Duration string `json:"duration"`
}

// AudioInfo holds information about an extracted audio.
type AudioInfo struct {
	URL      string `json:"url"`
	Type     string `json:"type"`
	Duration string `json:"duration"`
}

// LinkResource represents a link resource extracted from HTML.
type LinkResource struct {
	URL   string
	Title string
	Type  string
}

// Statistics holds processor statistics.
type Statistics struct {
	TotalProcessed     int64
	CacheHits          int64
	CacheMisses        int64
	ErrorCount         int64
	AverageProcessTime time.Duration
}
