package html

import (
	"fmt"
	"regexp"
	"sync"
	"time"
)

// Default configuration values.
const (
	DefaultMaxInputSize      = 50 * 1024 * 1024
	DefaultMaxCacheEntries   = 2000
	DefaultWorkerPoolSize    = 4
	DefaultCacheTTL          = time.Hour
	DefaultMaxDepth          = 500
	DefaultProcessingTimeout = 30 * time.Second

	// Configuration limits
	maxConfigInputSize    = 50 * 1024 * 1024
	maxConfigWorkerSize   = 256
	maxConfigDepth        = 500
	maxConfigCacheEntries = 100000 // Maximum 100K cache entries to prevent memory exhaustion

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
type Config struct {
	MaxInputSize       int
	MaxCacheEntries    int
	CacheTTL           time.Duration
	WorkerPoolSize     int
	EnableSanitization bool
	MaxDepth           int
	ProcessingTimeout  time.Duration
	Audit              AuditConfig
	Scorer             Scorer `json:"-"` // Optional custom scorer for content extraction
}

// DefaultConfig returns the default processor configuration.
func DefaultConfig() Config {
	return Config{
		MaxInputSize:       DefaultMaxInputSize,
		MaxCacheEntries:    DefaultMaxCacheEntries,
		CacheTTL:           DefaultCacheTTL,
		WorkerPoolSize:     DefaultWorkerPoolSize,
		EnableSanitization: true,
		MaxDepth:           DefaultMaxDepth,
		ProcessingTimeout:  DefaultProcessingTimeout,
		Audit:              DefaultAuditConfig(),
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
//   - Audit logging enabled for compliance requirements
func HighSecurityConfig() Config {
	return Config{
		MaxInputSize:       10 * 1024 * 1024, // 10MB - reduced for security
		MaxCacheEntries:    500,              // Reduced cache size
		CacheTTL:           30 * time.Minute, // Shorter TTL
		WorkerPoolSize:     2,                // Fewer workers for controlled resource usage
		EnableSanitization: true,             // Always enabled in high-security mode
		MaxDepth:           100,              // Reduced depth limit
		ProcessingTimeout:  10 * time.Second, // Shorter timeout
		Audit:              HighSecurityAuditConfig(),
	}
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
type ExtractConfig struct {
	ExtractArticle    bool
	PreserveImages    bool
	PreserveLinks     bool
	PreserveVideos    bool
	PreserveAudios    bool
	InlineImageFormat string
	TableFormat       string
	// Encoding specifies the character encoding of the input HTML.
	// If empty, the encoding will be auto-detected from meta tags or BOM.
	// Common values: "utf-8", "windows-1252", "iso-8859-1", "shift_jis", etc.
	Encoding string
}

// DefaultExtractConfig returns the default extraction configuration.
func DefaultExtractConfig() ExtractConfig {
	return ExtractConfig{
		ExtractArticle:    true,
		PreserveImages:    true,
		PreserveLinks:     true,
		PreserveVideos:    true,
		PreserveAudios:    true,
		InlineImageFormat: "none",
		TableFormat:       "markdown",
	}
}

// TextOnlyExtractConfig returns an ExtractConfig for extracting plain text only.
// This disables all media preservation and uses no inline image format.
func TextOnlyExtractConfig() ExtractConfig {
	return ExtractConfig{
		ExtractArticle:    true,
		PreserveImages:    false,
		PreserveLinks:     false,
		PreserveVideos:    false,
		PreserveAudios:    false,
		InlineImageFormat: "none",
		TableFormat:       "markdown",
	}
}

// LinkExtractionConfig holds the link extraction configuration.
type LinkExtractionConfig struct {
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

// DefaultLinkExtractionConfig returns the default link extraction configuration.
func DefaultLinkExtractionConfig() LinkExtractionConfig {
	return LinkExtractionConfig{
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

// resultPool is a sync.Pool for reusing Result structs in batch processing.
// This reduces memory allocations for high-throughput scenarios.
var resultPool = sync.Pool{
	New: func() any {
		return &Result{
			Images: make([]ImageInfo, 0, 16),
			Links:  make([]LinkInfo, 0, 16),
			Videos: make([]VideoInfo, 0, 4),
			Audios: make([]AudioInfo, 0, 4),
		}
	},
}

// getResult gets a Result from the pool.
// The returned Result has pre-allocated slices ready for use.
// Call putResult when done to return it to the pool.
func getResult() *Result {
	return resultPool.Get().(*Result)
}

// putResult returns a Result to the pool.
// The Result is reset before being returned to the pool.
// It is safe to call putResult with a nil pointer (no-op).
func putResult(r *Result) {
	if r == nil {
		return
	}
	// Reset fields
	r.Text = ""
	r.Title = ""
	r.Images = r.Images[:0]
	r.Links = r.Links[:0]
	r.Videos = r.Videos[:0]
	r.Audios = r.Audios[:0]
	r.WordCount = 0
	r.ReadingTime = 0
	r.ProcessingTime = 0
	resultPool.Put(r)
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
