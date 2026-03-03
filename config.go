package html

import (
	"fmt"
	"regexp"
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
	maxConfigInputSize    = DefaultMaxInputSize // Match DefaultMaxInputSize
	maxConfigWorkerSize   = 256
	maxConfigDepth        = DefaultMaxDepth // Match DefaultMaxDepth
	maxConfigCacheEntries = 100000          // Maximum 100K cache entries

	// Processing limits
	maxHTMLForRegex = 1000000
	maxRegexMatches = 1000
	maxCacheKeySize = 64 * 1024
	cacheKeySample  = 4096

	// Buffer size estimates
	initialTextSize   = 4096
	initialSliceCap   = 16
	initialMapCap     = 8
	imageHTMLBufExtra = 64
	extractTagCap     = 16
	linkMapCap        = 64

	// Processing thresholds
	wordsPerMinute       = 200
	maxShortStringLength = 32
)

// Pre-compiled regex patterns for media URL detection.
var (
	videoRegex = regexp.MustCompile(`(?i)https?://[^\s<>"',;)}\]]{1,500}\.(?:mp4|webm|ogg|mov|avi|wmv|flv|mkv|m4v|3gp)`)
	audioRegex = regexp.MustCompile(`(?i)https?://[^\s<>"',;)}\]]{1,500}\.(?:mp3|wav|ogg|m4a|aac|flac|wma|opus|oga)`)
)

// Config holds the processor configuration.
// It unifies processor settings, extraction options, and link extraction settings
// into a single configuration struct for simpler API usage.
type Config struct {
	// Processor settings
	MaxInputSize       int
	MaxCacheEntries    int
	CacheTTL           time.Duration
	CacheCleanup       time.Duration // Interval for background cleanup of expired cache entries (0 = disabled)
	WorkerPoolSize     int
	EnableSanitization bool
	MaxDepth           int
	ProcessingTimeout  time.Duration
	Audit              AuditConfig
	Scorer             Scorer `json:"-"` // Optional custom scorer for content extraction

	// Extraction settings
	ExtractArticle bool
	PreserveImages bool
	PreserveLinks  bool
	PreserveVideos bool
	PreserveAudios bool
	ImageFormat    string // Format for inline images: "none", "markdown", "html", "placeholder"
	LinkFormat     string // Format for inline links: "none", "markdown", "html"
	TableFormat    string // Format for tables: "markdown", "html"
	Encoding       string // Character encoding of input HTML (empty for auto-detection)

	// Link extraction settings
	LinkExtraction LinkExtractionOptions
}

// LinkExtractionOptions holds the link extraction configuration.
type LinkExtractionOptions struct {
	ResolveRelativeURLs  bool
	BaseURL              string
	IncludeImages        bool
	IncludeVideos        bool
	IncludeAudios        bool
	IncludeCSS           bool
	IncludeJS            bool
	IncludeContentLinks  bool
	IncludeExternalLinks bool
	IncludeIcons         bool
}

// DefaultConfig returns the default processor configuration.
func DefaultConfig() Config {
	return Config{
		// Processor settings
		MaxInputSize:       DefaultMaxInputSize,
		MaxCacheEntries:    DefaultMaxCacheEntries,
		CacheTTL:           DefaultCacheTTL,
		CacheCleanup:       DefaultCacheCleanup,
		WorkerPoolSize:     DefaultWorkerPoolSize,
		EnableSanitization: true,
		MaxDepth:           DefaultMaxDepth,
		ProcessingTimeout:  DefaultProcessingTimeout,
		Audit:              DefaultAuditConfig(),

		// Extraction settings
		ExtractArticle: true,
		PreserveImages: true,
		PreserveLinks:  true,
		PreserveVideos: true,
		PreserveAudios: true,
		ImageFormat:    "none",
		LinkFormat:     "none",
		TableFormat:    "markdown",

		// Link extraction settings
		LinkExtraction: LinkExtractionOptions{
			ResolveRelativeURLs:  true,
			IncludeImages:        true,
			IncludeVideos:        true,
			IncludeAudios:        true,
			IncludeCSS:           true,
			IncludeJS:            true,
			IncludeContentLinks:  true,
			IncludeExternalLinks: true,
			IncludeIcons:         true,
		},
	}
}

// HighSecurityConfig returns a configuration optimized for high-security environments.
// This configuration uses stricter limits to mitigate potential DoS attacks and
// is recommended for financial, healthcare, and government applications.
//
// Security enhancements over DefaultConfig:
//   - Smaller MaxInputSize (10MB vs 50MB) to limit memory exposure
//   - Lower MaxDepth (100 vs 500) to prevent deep nesting attacks
//   - Shorter ProcessingTimeout (10s vs 30s) for faster attack detection
//   - Fewer cache entries to reduce memory footprint
//   - Shorter cache TTL and more frequent cleanup
//   - Audit logging enabled for compliance requirements
func HighSecurityConfig() Config {
	return Config{
		// Processor settings
		MaxInputSize:       10 * 1024 * 1024, // 10MB - reduced for security
		MaxCacheEntries:    500,              // Reduced cache size
		CacheTTL:           30 * time.Minute, // Shorter TTL
		CacheCleanup:       time.Minute,      // More frequent cleanup
		WorkerPoolSize:     2,                // Fewer workers for controlled resource usage
		EnableSanitization: true,             // Always enabled in high-security mode
		MaxDepth:           100,              // Reduced depth limit
		ProcessingTimeout:  10 * time.Second, // Shorter timeout
		Audit:              HighSecurityAuditConfig(),

		// Extraction settings (same as DefaultConfig)
		ExtractArticle: true,
		PreserveImages: true,
		PreserveLinks:  true,
		PreserveVideos: true,
		PreserveAudios: true,
		ImageFormat:    "none",
		LinkFormat:     "none",
		TableFormat:    "markdown",

		// Link extraction settings (same as DefaultConfig)
		LinkExtraction: LinkExtractionOptions{
			ResolveRelativeURLs:  true,
			IncludeImages:        true,
			IncludeVideos:        true,
			IncludeAudios:        true,
			IncludeCSS:           true,
			IncludeJS:            true,
			IncludeContentLinks:  true,
			IncludeExternalLinks: true,
			IncludeIcons:         true,
		},
	}
}

// This disables all media preservation (images, links, videos, audios).
func TextOnlyConfig() Config {
	cfg := DefaultConfig()
	cfg.PreserveImages = false
	cfg.PreserveLinks = false
	cfg.PreserveVideos = false
	cfg.PreserveAudios = false
	return cfg
}

// MarkdownConfig returns a configuration optimized for Markdown output.
// This enables markdown format for inline images.
func MarkdownConfig() Config {
	cfg := DefaultConfig()
	cfg.ImageFormat = "markdown"
	return cfg
}

// Validate validates the configuration and returns an error if invalid.
func (c Config) Validate() error {
	switch {
	case c.MaxInputSize <= 0:
		return fmt.Errorf("%w: MaxInputSize must be positive, got %d", ErrInvalidConfig, c.MaxInputSize)
	case c.MaxInputSize > maxConfigInputSize:
		return fmt.Errorf("%w: MaxInputSize too large (max %d), got %d", ErrInvalidConfig, maxConfigInputSize, c.MaxInputSize)
	case c.MaxCacheEntries < 0:
		return fmt.Errorf("%w: MaxCacheEntries cannot be negative, got %d", ErrInvalidConfig, c.MaxCacheEntries)
	case c.MaxCacheEntries > maxConfigCacheEntries:
		return fmt.Errorf("%w: MaxCacheEntries too large (max %d), got %d", ErrInvalidConfig, maxConfigCacheEntries, c.MaxCacheEntries)
	case c.CacheTTL < 0:
		return fmt.Errorf("%w: CacheTTL cannot be negative, got %v", ErrInvalidConfig, c.CacheTTL)
	case c.WorkerPoolSize <= 0:
		return fmt.Errorf("%w: WorkerPoolSize must be positive, got %d", ErrInvalidConfig, c.WorkerPoolSize)
	case c.WorkerPoolSize > maxConfigWorkerSize:
		return fmt.Errorf("%w: WorkerPoolSize too large (max %d), got %d", ErrInvalidConfig, maxConfigWorkerSize, c.WorkerPoolSize)
	case c.MaxDepth <= 0:
		return fmt.Errorf("%w: MaxDepth must be positive, got %d", ErrInvalidConfig, c.MaxDepth)
	case c.MaxDepth > maxConfigDepth:
		return fmt.Errorf("%w: MaxDepth too large (max %d), got %d", ErrInvalidConfig, maxConfigDepth, c.MaxDepth)
	case c.ProcessingTimeout < 0:
		return fmt.Errorf("%w: ProcessingTimeout cannot be negative, got %v", ErrInvalidConfig, c.ProcessingTimeout)
	}
	return nil
}

// ExtractConfig holds the extraction configuration.
// This is an internal type used to pass extraction options to processing functions.
// For public API usage, prefer Config with DefaultConfig().
type ExtractConfig struct {
	ExtractArticle    bool
	PreserveImages    bool
	PreserveLinks     bool
	PreserveVideos    bool
	PreserveAudios    bool
	InlineImageFormat string
	InlineLinkFormat  string
	TableFormat       string
	// Encoding specifies the character encoding of the input HTML.
	// If empty, the encoding will be auto-detected from meta tags or BOM.
	// Common values: "utf-8", "windows-1252", "iso-8859-1", "shift_jis", etc.
	Encoding string
}

// defaultExtractConfig caches the default extraction configuration to avoid repeated allocations.
var defaultExtractConfig = ExtractConfig{
	ExtractArticle:    true,
	PreserveImages:    true,
	PreserveLinks:     true,
	PreserveVideos:    true,
	PreserveAudios:    true,
	InlineImageFormat: "none",
	InlineLinkFormat:  "none",
	TableFormat:       "markdown",
}

// DefaultExtractConfig returns the default extraction configuration.
// For new code, prefer using Config with DefaultConfig().
func DefaultExtractConfig() ExtractConfig {
	return defaultExtractConfig
}

// TextOnlyExtractConfig returns an ExtractConfig for extracting plain text only.
// This disables all media preservation and uses no inline image format.
// For new code, prefer TextOnlyConfig().
func TextOnlyExtractConfig() ExtractConfig {
	return ExtractConfig{
		ExtractArticle:    true,
		PreserveImages:    false,
		PreserveLinks:     false,
		PreserveVideos:    false,
		PreserveAudios:    false,
		InlineImageFormat: "none",
		InlineLinkFormat:  "none",
		TableFormat:       "markdown",
	}
}

// LinkExtractionConfig holds the link extraction configuration.
// This is an alias for LinkExtractionOptions, used internally for link extraction.
// For public API usage, prefer Config.LinkExtraction.
type LinkExtractionConfig = LinkExtractionOptions

// defaultLinkExtractionConfig caches the default link extraction configuration.
var defaultLinkExtractionConfig = LinkExtractionConfig{
	ResolveRelativeURLs:  true,
	BaseURL:              "",
	IncludeImages:        true,
	IncludeVideos:        true,
	IncludeAudios:        true,
	IncludeCSS:           true,
	IncludeJS:            true,
	IncludeContentLinks:  true,
	IncludeExternalLinks: true,
	IncludeIcons:         true,
}

// DefaultLinkExtractionConfig returns the default link extraction configuration.
// For new code, prefer using DefaultConfig().LinkExtraction.
func DefaultLinkExtractionConfig() LinkExtractionConfig {
	return defaultLinkExtractionConfig
}

// Result holds the extraction result.
type Result struct {
	Text           string        `json:"text"`
	Title          string        `json:"title"`
	Images         []ImageInfo   `json:"images,omitempty"`
	Links          []LinkInfo    `json:"links,omitempty"`
	Videos         []VideoInfo   `json:"videos,omitempty"`
	Audios         []AudioInfo   `json:"audios,omitempty"`
	ProcessingTime time.Duration `json:"processing_time_ms"`
	WordCount      int           `json:"word_count"`
	ReadingTime    time.Duration `json:"reading_time_ms"`
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
