package html

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	htmlstd "html"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cybergodev/html/internal"
	stdxhtml "golang.org/x/net/html"
)

// Type aliases for commonly used types from golang.org/x/net/html
type (
	Node        = stdxhtml.Node
	NodeType    = stdxhtml.NodeType
	Token       = stdxhtml.Token
	Attribute   = stdxhtml.Attribute
	Tokenizer   = stdxhtml.Tokenizer
	ParseOption = stdxhtml.ParseOption
)

const (
	ErrorNode    = stdxhtml.ErrorNode
	TextNode     = stdxhtml.TextNode
	DocumentNode = stdxhtml.DocumentNode
	ElementNode  = stdxhtml.ElementNode
	CommentNode  = stdxhtml.CommentNode
	DoctypeNode  = stdxhtml.DoctypeNode
	RawNode      = stdxhtml.RawNode
)

const (
	ErrorToken          = stdxhtml.ErrorToken
	TextToken           = stdxhtml.TextToken
	StartTagToken       = stdxhtml.StartTagToken
	EndTagToken         = stdxhtml.EndTagToken
	SelfClosingTagToken = stdxhtml.SelfClosingTagToken
	CommentToken        = stdxhtml.CommentToken
	DoctypeToken        = stdxhtml.DoctypeToken
)

// Extract extracts content from HTML bytes with automatic encoding detection.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
//
// This is the primary method for HTML content extraction when the source encoding
// may not be UTF-8, such as content from HTTP responses, databases, or files.
//
// Parameters:
//
//	htmlBytes - Raw HTML bytes (auto-detects encoding)
//	configs - Optional extraction configurations
//
// Returns:
//
//	*Result - Extracted content with UTF-8 encoded text
//	error - Error if extraction fails
//
// Example:
//
//	// HTTP response
//	resp, _ := http.Get(url)
//	bytes, _ := io.ReadAll(resp.Body)
//	result, _ := html.Extract(bytes)
//
//	// File
//	bytes, _ := os.ReadFile("document.html")
//	result, _ := html.Extract(bytes)
func Extract(htmlBytes []byte, configs ...ExtractConfig) (*Result, error) {
	processor, _ := New()
	defer processor.Close()
	return processor.Extract(htmlBytes, configs...)
}

// ExtractFromFile extracts content from an HTML file with automatic encoding detection.
// Use this when you have a file path instead of raw bytes.
//
// Parameters:
//
//	filePath - Path to the HTML file
//	configs - Optional extraction configurations
//
// Returns:
//
//	*Result - Extracted content with UTF-8 encoded text
//	error - Error if file reading or extraction fails
//
// Example:
//
//	result, _ := html.ExtractFromFile("document.html", html.ExtractConfig{
//	    InlineImageFormat: "markdown",
//	})
func ExtractFromFile(filePath string, configs ...ExtractConfig) (*Result, error) {
	processor, _ := New()
	defer processor.Close()
	return processor.ExtractFromFile(filePath, configs...)
}

// ExtractText extracts plain text from HTML bytes with automatic encoding detection.
// The method automatically detects character encoding and converts to UTF-8.
//
// Parameters:
//
//	htmlBytes - Raw HTML bytes (auto-detects encoding)
//
// Returns:
//
//	string - Extracted plain text in UTF-8
//	error - Error if extraction fails
//
// Example:
//
//	bytes, _ := os.ReadFile("document.html")
//	text, _ := html.ExtractText(bytes)
func ExtractText(htmlBytes []byte) (string, error) {
	result, err := Extract(htmlBytes)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// ExtractAllLinks extracts all links from HTML bytes with automatic encoding detection.
// The method automatically detects character encoding and converts to UTF-8.
//
// Parameters:
//
//	htmlBytes - Raw HTML bytes (auto-detects encoding)
//	configs - Optional link extraction configurations
//
// Returns:
//
//	[]LinkResource - List of extracted links with UTF-8 encoded titles
//	error - Error if extraction fails
//
// Example:
//
//	bytes, _ := os.ReadFile("document.html")
//	links, _ := html.ExtractAllLinks(bytes)
func ExtractAllLinks(htmlBytes []byte, configs ...LinkExtractionConfig) ([]LinkResource, error) {
	processor, _ := New()
	defer processor.Close()
	return processor.ExtractAllLinks(htmlBytes, configs...)
}

func GroupLinksByType(links []LinkResource) map[string][]LinkResource {
	if len(links) == 0 {
		return make(map[string][]LinkResource)
	}

	grouped := make(map[string][]LinkResource, 8)
	for _, link := range links {
		if link.Type != "" {
			grouped[link.Type] = append(grouped[link.Type], link)
		} else {
			grouped["unknown"] = append(grouped["unknown"], link)
		}
	}
	return grouped
}

const (
	DefaultMaxInputSize      = 50 * 1024 * 1024
	DefaultMaxCacheEntries   = 2000
	DefaultWorkerPoolSize    = 4
	DefaultCacheTTL          = time.Hour
	DefaultMaxDepth          = 500
	DefaultProcessingTimeout = 30 * time.Second

	// Configuration limits
	maxConfigInputSize  = 50 * 1024 * 1024
	maxConfigWorkerSize = 256
	maxConfigDepth      = 500

	// Processing limits
	maxURLLength     = 2000
	maxHTMLForRegex  = 1000000
	maxRegexMatches  = 1000
	maxDataURILength = 100000
	maxCacheKeySize  = 64 * 1024
	cacheKeySample   = 4096

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

var (
	ErrBufferExceeded    = stdxhtml.ErrBufferExceeded
	Parse                = stdxhtml.Parse
	ParseFragment        = stdxhtml.ParseFragment
	Render               = stdxhtml.Render
	EscapeString         = htmlstd.EscapeString
	UnescapeString       = htmlstd.UnescapeString
	NewTokenizer         = stdxhtml.NewTokenizer
	NewTokenizerFragment = stdxhtml.NewTokenizerFragment

	videoRegex = regexp.MustCompile(`(?i)https?://[^\s<>"',;)}\]]{1,500}\.(?:mp4|webm|ogg|mov|avi|wmv|flv|mkv|m4v|3gp)`)
	audioRegex = regexp.MustCompile(`(?i)https?://[^\s<>"',;)}\]]{1,500}\.(?:mp3|wav|ogg|m4a|aac|flac|wma|opus|oga)`)
)

type Processor struct {
	config *Config
	cache  *internal.Cache
	closed atomic.Bool
	stats  struct {
		totalProcessed   atomic.Int64
		cacheHits        atomic.Int64
		cacheMisses      atomic.Int64
		errorCount       atomic.Int64
		totalProcessTime atomic.Int64
	}
}

type Config struct {
	MaxInputSize       int
	MaxCacheEntries    int
	CacheTTL           time.Duration
	WorkerPoolSize     int
	EnableSanitization bool
	MaxDepth           int
	ProcessingTimeout  time.Duration
}

func DefaultConfig() Config {
	return Config{
		MaxInputSize:       DefaultMaxInputSize,
		MaxCacheEntries:    DefaultMaxCacheEntries,
		CacheTTL:           DefaultCacheTTL,
		WorkerPoolSize:     DefaultWorkerPoolSize,
		EnableSanitization: true,
		MaxDepth:           DefaultMaxDepth,
		ProcessingTimeout:  DefaultProcessingTimeout,
	}
}

func validateConfig(c Config) error {
	switch {
	case c.MaxInputSize <= 0:
		return fmt.Errorf("%w: MaxInputSize must be positive, got %d", ErrInvalidConfig, c.MaxInputSize)
	case c.MaxInputSize > maxConfigInputSize:
		return fmt.Errorf("%w: MaxInputSize too large (max %d), got %d", ErrInvalidConfig, maxConfigInputSize, c.MaxInputSize)
	case c.MaxCacheEntries < 0:
		return fmt.Errorf("%w: MaxCacheEntries cannot be negative, got %d", ErrInvalidConfig, c.MaxCacheEntries)
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

type ImageInfo struct {
	URL          string `json:"url"`
	Alt          string `json:"alt"`
	Title        string `json:"title"`
	Width        string `json:"width"`
	Height       string `json:"height"`
	IsDecorative bool   `json:"is_decorative"`
	Position     int    `json:"position"`
}

type LinkInfo struct {
	URL        string `json:"url"`
	Text       string `json:"text"`
	Title      string `json:"title"`
	IsExternal bool   `json:"is_external"`
	IsNoFollow bool   `json:"is_nofollow"`
}

type VideoInfo struct {
	URL      string `json:"url"`
	Type     string `json:"type"`
	Poster   string `json:"poster"`
	Width    string `json:"width"`
	Height   string `json:"height"`
	Duration string `json:"duration"`
}

type AudioInfo struct {
	URL      string `json:"url"`
	Type     string `json:"type"`
	Duration string `json:"duration"`
}

type LinkResource struct {
	URL   string
	Title string
	Type  string
}

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

func resolveExtractConfig(configs ...ExtractConfig) ExtractConfig {
	if len(configs) > 0 {
		cfg := configs[0]
		// Validate and normalize TableFormat
		format := strings.ToLower(strings.TrimSpace(cfg.TableFormat))
		if format != "markdown" && format != "html" {
			format = "markdown"
		}
		cfg.TableFormat = format
		return cfg
	}
	return DefaultExtractConfig()
}

func resolveLinkExtractionConfig(configs ...LinkExtractionConfig) LinkExtractionConfig {
	if len(configs) > 0 {
		return configs[0]
	}
	return DefaultLinkExtractionConfig()
}

type Statistics struct {
	TotalProcessed     int64
	CacheHits          int64
	CacheMisses        int64
	ErrorCount         int64
	AverageProcessTime time.Duration
}

// New creates a new HTML processor with the given configuration.
// If no configuration is provided, it uses DefaultConfig().
//
// The function signature uses variadic arguments to make the config optional:
//
//	processor, err := html.New()              // Uses DefaultConfig()
//	processor, err := html.New(config)        // Uses custom config
//
// The returned processor must be closed when no longer needed:
//
//	processor, err := html.New()
//	defer processor.Close()
func New(configs ...Config) (*Processor, error) {
	var config Config

	if len(configs) == 0 {
		// No config provided, use defaults
		config = DefaultConfig()
	} else {
		// Use the provided config
		config = configs[0]
	}

	if err := validateConfig(config); err != nil {
		return nil, err
	}
	return &Processor{
		config: &config,
		cache:  internal.NewCache(config.MaxCacheEntries, config.CacheTTL),
	}, nil
}

// Extract extracts content from HTML bytes with automatic encoding detection.
// This is the main extraction method that processes HTML bytes after detecting
// and converting their character encoding to UTF-8.
//
// The method performs the following steps:
// 1. Validates processor state (not closed)
// 2. Resolves extraction configuration
// 3. Checks input size limits
// 4. Detects character encoding and converts to UTF-8
// 5. Processes content with caching support
// 6. Updates statistics and returns result
func (p *Processor) Extract(htmlBytes []byte, configs ...ExtractConfig) (*Result, error) {
	if p == nil {
		return nil, ErrProcessorClosed
	}
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	config := resolveExtractConfig(configs...)

	if len(htmlBytes) > p.config.MaxInputSize {
		p.stats.errorCount.Add(1)
		return nil, fmt.Errorf("%w: size=%d, max=%d", ErrInputTooLarge, len(htmlBytes), p.config.MaxInputSize)
	}

	startTime := time.Now()

	// Detect encoding and convert to UTF-8
	utf8String, _, err := internal.DetectAndConvertToUTF8String(htmlBytes, config.Encoding)
	if err != nil {
		p.stats.errorCount.Add(1)
		return nil, fmt.Errorf("encoding detection failed: %w", err)
	}

	// Use the converted UTF-8 string for cache key
	cacheKey := p.generateCacheKey(utf8String, config)
	if cached := p.cache.Get(cacheKey); cached != nil {
		p.stats.cacheHits.Add(1)
		p.stats.totalProcessed.Add(1)
		if result, ok := cached.(*Result); ok {
			return result, nil
		}
	}
	p.stats.cacheMisses.Add(1)

	// Process the content
	var result *Result
	if p.config.ProcessingTimeout > 0 {
		result, err = p.processWithTimeout(utf8String, config)
	} else {
		result, err = p.processContent(utf8String, config)
	}

	if err != nil {
		p.stats.errorCount.Add(1)
		return nil, err
	}

	processingTime := time.Since(startTime)
	result.ProcessingTime = processingTime
	p.stats.totalProcessTime.Add(int64(processingTime))
	p.stats.totalProcessed.Add(1)

	if p.config.MaxCacheEntries > 0 {
		p.cache.Set(cacheKey, result)
	}

	return result, nil
}

func (p *Processor) processWithTimeout(htmlContent string, config ExtractConfig) (*Result, error) {
	result, err := withTimeout(p.config.ProcessingTimeout, func() (*Result, error) {
		return p.processContent(htmlContent, config)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (p *Processor) ExtractFromFile(filePath string, configs ...ExtractConfig) (*Result, error) {
	if p == nil {
		return nil, ErrProcessorClosed
	}
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	// Validate file path
	if filePath == "" {
		return nil, fmt.Errorf("%w: empty file path", ErrInvalidFilePath)
	}

	// Clean the file path to resolve any "." or ".." components
	cleanPath := filepath.Clean(filePath)

	// After cleaning, check if the path contains parent directory references
	// This catches path traversal attempts like "../file", "subdir/../../file", etc.
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("%w: path traversal detected: %s", ErrInvalidFilePath, cleanPath)
	}

	config := resolveExtractConfig(configs...)

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, cleanPath)
		}
		return nil, fmt.Errorf("read file %q: %w", cleanPath, err)
	}

	return p.Extract(data, config)
}

func (p *Processor) ExtractBatch(htmlContents [][]byte, configs ...ExtractConfig) ([]*Result, error) {
	if p == nil {
		return nil, ErrProcessorClosed
	}
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	if len(htmlContents) == 0 {
		return []*Result{}, nil
	}

	config := resolveExtractConfig(configs...)

	results := make([]*Result, len(htmlContents))
	errs := make([]error, len(htmlContents))
	sem := make(chan struct{}, p.config.WorkerPoolSize)
	var wg sync.WaitGroup

	for i, content := range htmlContents {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, htmlBytes []byte) {
			defer wg.Done()
			defer func() { <-sem }()

			results[idx], errs[idx] = p.Extract(htmlBytes, config)
		}(i, content)
	}

	wg.Wait()
	return collectResults(results, errs, nil)
}

func (p *Processor) ExtractBatchFiles(filePaths []string, configs ...ExtractConfig) ([]*Result, error) {
	if p == nil {
		return nil, ErrProcessorClosed
	}
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	if len(filePaths) == 0 {
		return []*Result{}, nil
	}

	config := resolveExtractConfig(configs...)

	results := make([]*Result, len(filePaths))
	errs := make([]error, len(filePaths))
	sem := make(chan struct{}, p.config.WorkerPoolSize)
	var wg sync.WaitGroup

	for i, path := range filePaths {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, filePath string) {
			defer wg.Done()
			defer func() { <-sem }()

			results[idx], errs[idx] = p.ExtractFromFile(filePath, config)
		}(i, path)
	}

	wg.Wait()
	return collectResults(results, errs, filePaths)
}

// ExtractAllLinks extracts all links from HTML bytes with automatic encoding detection.
// The method automatically detects character encoding and converts to UTF-8 before
// extracting links, ensuring that link titles and text are properly decoded.
func (p *Processor) ExtractAllLinks(htmlBytes []byte, configs ...LinkExtractionConfig) ([]LinkResource, error) {
	if p == nil {
		return nil, ErrProcessorClosed
	}
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	// Validate input
	if len(htmlBytes) == 0 {
		return []LinkResource{}, nil
	}

	config := resolveLinkExtractionConfig(configs...)

	if len(htmlBytes) > p.config.MaxInputSize {
		p.stats.errorCount.Add(1)
		return nil, fmt.Errorf("%w: size=%d, max=%d", ErrInputTooLarge, len(htmlBytes), p.config.MaxInputSize)
	}

	startTime := time.Now()

	// Detect encoding and convert to UTF-8
	utf8String, _, err := internal.DetectAndConvertToUTF8String(htmlBytes, "")
	if err != nil {
		p.stats.errorCount.Add(1)
		return nil, fmt.Errorf("encoding detection failed: %w", err)
	}

	// Process with timeout if configured
	var links []LinkResource
	if p.config.ProcessingTimeout > 0 {
		links, err = p.extractLinksWithTimeout(utf8String, config)
	} else {
		links, err = p.extractAllLinksFromContent(utf8String, config)
	}

	if err != nil {
		p.stats.errorCount.Add(1)
		return nil, err
	}

	processingTime := time.Since(startTime)
	p.stats.totalProcessTime.Add(int64(processingTime))
	p.stats.totalProcessed.Add(1)

	return links, nil
}

func (p *Processor) extractLinksWithTimeout(htmlContent string, config LinkExtractionConfig) ([]LinkResource, error) {
	return withTimeout(p.config.ProcessingTimeout, func() ([]LinkResource, error) {
		return p.extractAllLinksFromContent(htmlContent, config)
	})
}

func collectResults(results []*Result, errs []error, names []string) ([]*Result, error) {
	var firstErr error
	successCount := 0
	failCount := 0

	for i, err := range errs {
		if err != nil {
			failCount++
			if firstErr == nil {
				if names != nil {
					firstErr = fmt.Errorf("%s: %w", names[i], err)
				} else {
					firstErr = fmt.Errorf("item %d: %w", i, err)
				}
			}
		} else {
			successCount++
		}
	}

	switch {
	case successCount == 0:
		return results, fmt.Errorf("all %d items failed: %w", len(results), firstErr)
	case failCount > 0:
		return results, fmt.Errorf("partial failure (%d/%d succeeded): %w", successCount, len(results), firstErr)
	default:
		return results, nil
	}
}

func (p *Processor) GetStatistics() Statistics {
	if p == nil {
		return Statistics{}
	}
	totalProcessed := p.stats.totalProcessed.Load()
	totalTime := time.Duration(p.stats.totalProcessTime.Load())
	var avgTime time.Duration
	if totalProcessed > 0 {
		avgTime = totalTime / time.Duration(totalProcessed)
	}
	return Statistics{
		TotalProcessed:     totalProcessed,
		CacheHits:          p.stats.cacheHits.Load(),
		CacheMisses:        p.stats.cacheMisses.Load(),
		ErrorCount:         p.stats.errorCount.Load(),
		AverageProcessTime: avgTime,
	}
}

// ClearCache clears the cache contents but preserves cumulative statistics.
// Use ResetStatistics to reset statistics counters.
func (p *Processor) ClearCache() {
	if p == nil {
		return
	}
	p.cache.Clear()
}

// ResetStatistics resets all statistics counters to zero.
// This preserves cache entries while clearing the accumulated metrics.
func (p *Processor) ResetStatistics() {
	if p == nil {
		return
	}
	p.stats.cacheHits.Store(0)
	p.stats.cacheMisses.Store(0)
	p.stats.errorCount.Store(0)
	p.stats.totalProcessed.Store(0)
	p.stats.totalProcessTime.Store(0)
}

func (p *Processor) Close() error {
	if p == nil {
		return nil
	}
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	p.cache.Clear()
	return nil
}

// withTimeout executes a function with a timeout, returning its result or an error if timeout expires.
// This is a generic helper that eliminates code duplication in timeout handling.
func withTimeout[T any](timeout time.Duration, fn func() (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type result struct {
		res T
		err error
	}

	resultChan := make(chan result, 1)
	go func() {
		res, err := fn()
		select {
		case resultChan <- result{res: res, err: err}:
		case <-ctx.Done():
		}
	}()

	select {
	case res := <-resultChan:
		return res.res, res.err
	case <-ctx.Done():
		var zero T
		return zero, ErrProcessingTimeout
	}
}

func (p *Processor) processContent(htmlContent string, opts ExtractConfig) (*Result, error) {
	if strings.TrimSpace(htmlContent) == "" {
		return &Result{}, nil
	}

	originalHTML := htmlContent

	if p.config.EnableSanitization {
		htmlContent = internal.SanitizeHTML(htmlContent)
	}

	doc, err := stdxhtml.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidHTML, err)
	}

	// Validate depth during extraction to avoid duplicate traversal
	if err := p.validateDepthTraversal(doc, 0); err != nil {
		return nil, err
	}

	return p.extractFromDocument(doc, originalHTML, opts)
}

// validateDepthTraversal validates DOM tree depth during a single traversal.
// This is more efficient than separately calling validateDepth and extractFromDocument.
func (p *Processor) validateDepthTraversal(n *Node, depth int) error {
	if depth > p.config.MaxDepth {
		return ErrMaxDepthExceeded
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if err := p.validateDepthTraversal(c, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (p *Processor) extractFromDocument(doc *Node, htmlContent string, opts ExtractConfig) (*Result, error) {
	result := &Result{}
	result.Title = p.extractTitle(doc)

	contentNode := doc
	if opts.ExtractArticle {
		if article := p.extractArticleNode(doc); article != nil {
			contentNode = article
		}
	}
	contentNode = internal.CleanContentNode(contentNode)

	format := strings.ToLower(strings.TrimSpace(opts.InlineImageFormat))
	if format == "" {
		format = "none"
	}

	if format != "none" {
		images := p.extractImagesWithPosition(contentNode)

		if opts.PreserveImages {
			result.Images = images
		}

		var sb strings.Builder
		sb.Grow(initialTextSize)
		imageCounter := 0
		internal.ExtractTextWithStructureAndImages(contentNode, &sb, 0, &imageCounter, opts.TableFormat)
		textWithPlaceholders := internal.CleanText(sb.String(), nil)
		result.Text = p.formatInlineImages(textWithPlaceholders, images, format)
	} else {
		result.Text = p.extractTextContent(contentNode, opts.TableFormat)

		if opts.PreserveImages {
			result.Images = p.extractImages(contentNode)
		}
	}

	result.WordCount = p.countWords(result.Text)
	result.ReadingTime = p.calculateReadingTime(result.WordCount)

	if opts.PreserveLinks {
		result.Links = p.extractLinks(contentNode)
	}
	if opts.PreserveVideos {
		result.Videos = p.extractVideos(doc, htmlContent)
	}
	if opts.PreserveAudios {
		result.Audios = p.extractAudios(doc, htmlContent)
	}
	return result, nil
}

func (p *Processor) extractTitle(doc *Node) string {
	if doc == nil {
		return ""
	}
	if titleNode := internal.FindElementByTag(doc, "title"); titleNode != nil {
		if title := internal.GetTextContent(titleNode); title != "" {
			return title
		}
	}
	if h1Node := internal.FindElementByTag(doc, "h1"); h1Node != nil {
		if title := internal.GetTextContent(h1Node); title != "" {
			return title
		}
	}
	if h2Node := internal.FindElementByTag(doc, "h2"); h2Node != nil {
		return internal.GetTextContent(h2Node)
	}
	return ""
}

func (p *Processor) extractArticleNode(doc *Node) *Node {
	if doc == nil {
		return nil
	}
	candidates := make(map[*Node]int)
	internal.WalkNodes(doc, func(n *Node) bool {
		if n.Type == ElementNode {
			if score := internal.ScoreContentNode(n); score > 0 {
				candidates[n] = score
			}
		}
		return true
	})
	if bestNode := internal.SelectBestCandidate(candidates); bestNode != nil {
		return bestNode
	}
	return internal.FindElementByTag(doc, "body")
}

func (p *Processor) extractTextContent(node *Node, tableFormat string) string {
	var sb strings.Builder
	sb.Grow(initialTextSize)
	internal.ExtractTextWithStructureAndImages(node, &sb, 0, nil, tableFormat)
	return internal.CleanText(sb.String(), nil)
}

func (p *Processor) formatInlineImages(textWithPlaceholders string, images []ImageInfo, format string) string {
	if len(images) == 0 || format == "placeholder" || format == "none" {
		return textWithPlaceholders
	}

	replacements := make([]string, 0, len(images)*2)

	switch format {
	case "markdown":
		for i := range images {
			if images[i].Position == 0 {
				continue
			}
			placeholder := fmt.Sprintf("[IMAGE:%d]", images[i].Position)
			altText := images[i].Alt
			if altText == "" {
				altText = fmt.Sprintf("Image %d", images[i].Position)
			}
			markdown := fmt.Sprintf("![%s](%s)", altText, images[i].URL)
			replacements = append(replacements, placeholder, markdown)
		}
	case "html":
		for i := range images {
			if images[i].Position == 0 {
				continue
			}
			placeholder := fmt.Sprintf("[IMAGE:%d]", images[i].Position)
			var htmlImg strings.Builder
			htmlImg.Grow(len(images[i].URL) + len(images[i].Alt) + len(images[i].Width) + len(images[i].Height) + imageHTMLBufExtra)
			htmlImg.WriteString(`<img src="`)
			htmlImg.WriteString(htmlstd.EscapeString(images[i].URL))
			htmlImg.WriteString(`" alt="`)
			htmlImg.WriteString(htmlstd.EscapeString(images[i].Alt))
			htmlImg.WriteString(`"`)
			if images[i].Width != "" {
				htmlImg.WriteString(` width="`)
				htmlImg.WriteString(htmlstd.EscapeString(images[i].Width))
				htmlImg.WriteString(`"`)
			}
			if images[i].Height != "" {
				htmlImg.WriteString(` height="`)
				htmlImg.WriteString(htmlstd.EscapeString(images[i].Height))
				htmlImg.WriteString(`"`)
			}
			htmlImg.WriteString(">")
			replacements = append(replacements, placeholder, htmlImg.String())
		}
	}

	if len(replacements) > 0 {
		replacer := strings.NewReplacer(replacements...)
		return replacer.Replace(textWithPlaceholders)
	}

	return textWithPlaceholders
}

func (p *Processor) extractImages(node *Node) []ImageInfo {
	images := make([]ImageInfo, 0, initialSliceCap)

	internal.WalkNodes(node, func(n *Node) bool {
		if n.Type == ElementNode && n.Data == "img" {
			img := p.parseImageNode(n, 0)
			if img.URL != "" {
				images = append(images, img)
			}
		}
		return true
	})

	return images
}

func (p *Processor) extractImagesWithPosition(node *Node) []ImageInfo {
	images := make([]ImageInfo, 0, initialSliceCap)
	position := 0

	internal.WalkNodes(node, func(n *Node) bool {
		if n.Type == ElementNode && n.Data == "img" {
			position++
			img := p.parseImageNode(n, position)
			if img.URL != "" {
				images = append(images, img)
			}
		}
		return true
	})

	return images
}

func (p *Processor) parseImageNode(n *Node, position int) ImageInfo {
	img := ImageInfo{Position: position}

	for _, attr := range n.Attr {
		switch attr.Key {
		case "src":
			if !internal.IsValidURL(attr.Val) {
				return ImageInfo{}
			}
			img.URL = attr.Val
		case "alt":
			img.Alt = attr.Val
		case "title":
			img.Title = attr.Val
		case "width":
			img.Width = attr.Val
		case "height":
			img.Height = attr.Val
		}
	}

	if img.URL == "" {
		return ImageInfo{}
	}

	img.IsDecorative = img.Alt == ""
	return img
}

func (p *Processor) extractLinks(node *Node) []LinkInfo {
	links := make([]LinkInfo, 0, initialSliceCap)

	internal.WalkNodes(node, func(n *Node) bool {
		if n.Type == ElementNode && n.Data == "a" {
			link := p.parseLinkNode(n)
			if link.URL != "" {
				links = append(links, link)
			}
		}
		return true
	})

	return links
}

func (p *Processor) parseLinkNode(n *Node) LinkInfo {
	link := LinkInfo{}

	for _, attr := range n.Attr {
		switch attr.Key {
		case "href":
			if !internal.IsValidURL(attr.Val) {
				return LinkInfo{}
			}
			link.URL = attr.Val
		case "title":
			link.Title = attr.Val
		case "rel":
			// Check for nofollow in rel attribute (case-insensitive)
			if strings.Contains(strings.ToLower(attr.Val), "nofollow") {
				link.IsNoFollow = true
			}
		}
	}

	if link.URL == "" {
		return LinkInfo{}
	}

	link.Text = internal.GetTextContent(n)
	link.IsExternal = internal.IsExternalURL(link.URL)
	return link
}

func (p *Processor) countWords(text string) int {
	if text == "" {
		return 0
	}
	words := strings.Fields(text)
	return len(words)
}

func (p *Processor) calculateReadingTime(wordCount int) time.Duration {
	if wordCount == 0 {
		return 0
	}
	minutes := float64(wordCount) / wordsPerMinute
	return time.Duration(minutes * float64(time.Minute))
}

func (p *Processor) extractVideos(node *Node, htmlContent string) []VideoInfo {
	videos := make([]VideoInfo, 0, initialSliceCap)
	seen := make(map[string]bool, initialMapCap)

	// First, extract from the HTML content directly for iframe/embed/object tags
	// These may be removed by sanitization, so we parse them from raw HTML first
	if len(htmlContent) > 0 && len(htmlContent) <= maxHTMLForRegex*10 {
		// Parse iframe tags
		iframeMatches := p.extractTagAttributes(htmlContent, "iframe", "src")
		for _, url := range iframeMatches {
			if internal.IsValidURL(url) && internal.IsVideoURL(url) && !seen[url] {
				seen[url] = true
				videos = append(videos, VideoInfo{
					URL:  url,
					Type: internal.DetectVideoType(url),
				})
			}
		}

		// Parse embed tags
		embedMatches := p.extractTagAttributes(htmlContent, "embed", "src", "data")
		for _, url := range embedMatches {
			if internal.IsValidURL(url) && internal.IsVideoURL(url) && !seen[url] {
				seen[url] = true
				videos = append(videos, VideoInfo{
					URL:  url,
					Type: internal.DetectVideoType(url),
				})
			}
		}

		// Parse object tags
		objectMatches := p.extractTagAttributes(htmlContent, "object", "data")
		for _, url := range objectMatches {
			if internal.IsValidURL(url) && internal.IsVideoURL(url) && !seen[url] {
				seen[url] = true
				videos = append(videos, VideoInfo{
					URL:  url,
					Type: internal.DetectVideoType(url),
				})
			}
		}
	}

	// Then extract from the DOM tree (for video tags and any iframe/embed/object that survived sanitization)
	internal.WalkNodes(node, func(n *Node) bool {
		if n.Type != ElementNode {
			return true
		}

		switch n.Data {
		case "video":
			if video := p.parseVideoNode(n); video.URL != "" && !seen[video.URL] {
				seen[video.URL] = true
				videos = append(videos, video)
			}

		case "iframe":
			if video := p.parseIframeNode(n); video.URL != "" && !seen[video.URL] {
				seen[video.URL] = true
				videos = append(videos, video)
			}

		case "embed", "object":
			if video := p.parseEmbedNode(n); video.URL != "" && !seen[video.URL] {
				seen[video.URL] = true
				videos = append(videos, video)
			}
		}
		return true
	})

	// Finally, use regex to find any video URLs in the HTML content
	if len(htmlContent) <= maxHTMLForRegex {
		matches := videoRegex.FindAllString(htmlContent, maxRegexMatches)
		for _, url := range matches {
			if internal.IsValidURL(url) && !seen[url] {
				seen[url] = true
				videos = append(videos, VideoInfo{
					URL:  url,
					Type: internal.DetectVideoType(url),
				})
			}
		}
	}

	return videos
}

func (p *Processor) parseVideoNode(n *Node) VideoInfo {
	video := VideoInfo{}
	for _, attr := range n.Attr {
		switch attr.Key {
		case "src":
			if !internal.IsValidURL(attr.Val) {
				return VideoInfo{}
			}
			video.URL = attr.Val
		case "poster":
			video.Poster = attr.Val
		case "width":
			video.Width = attr.Val
		case "height":
			video.Height = attr.Val
		case "duration":
			video.Duration = attr.Val
		}
	}

	if video.URL == "" {
		video.URL, video.Type = p.findSourceURL(n)
	}

	if !internal.IsValidURL(video.URL) {
		return VideoInfo{}
	}

	return video
}

func (p *Processor) parseIframeNode(n *Node) VideoInfo {
	for _, attr := range n.Attr {
		if attr.Key == "src" && internal.IsValidURL(attr.Val) && internal.IsVideoURL(attr.Val) {
			video := VideoInfo{URL: attr.Val, Type: "embed"}
			for _, a := range n.Attr {
				switch a.Key {
				case "width":
					video.Width = a.Val
				case "height":
					video.Height = a.Val
				}
			}
			return video
		}
	}
	return VideoInfo{}
}

func (p *Processor) parseEmbedNode(n *Node) VideoInfo {
	for _, attr := range n.Attr {
		if (attr.Key == "src" || attr.Key == "data") && internal.IsValidURL(attr.Val) && internal.IsVideoURL(attr.Val) {
			video := VideoInfo{URL: attr.Val}
			for _, a := range n.Attr {
				switch a.Key {
				case "type":
					video.Type = a.Val
				case "width":
					video.Width = a.Val
				case "height":
					video.Height = a.Val
				}
			}
			return video
		}
	}
	return VideoInfo{}
}

// extractTagAttributes extracts specified attributes from all occurrences of a tag in HTML content.
// This function operates on raw HTML strings before sanitization, allowing extraction from
// tags that might be removed during HTML sanitization (e.g., iframe, embed, object).
//
// Parameters:
//   - htmlContent: The raw HTML content to search
//   - tagName: The name of the tag to search for (e.g., "iframe", "embed")
//   - attrNames: One or more attribute names to extract (e.g., "src", "data")
//
// Returns a slice of attribute values found in matching tags.
func (p *Processor) extractTagAttributes(htmlContent, tagName string, attrNames ...string) []string {
	results := make([]string, 0, extractTagCap)
	// Convert tag name to lowercase once for comparison
	lowerTag := "<" + strings.ToLower(tagName)

	pos := 0
	for pos < len(htmlContent) {
		// Find the next occurrence of the tag using case-insensitive search
		// We'll search in chunks to avoid converting the entire HTML to lowercase
		tagStart := findTagIgnoreCase(htmlContent[pos:], lowerTag)
		if tagStart == -1 {
			break
		}
		tagStart += pos

		// Verify it's a complete tag name (not a partial match)
		if tagStart+len(lowerTag) < len(htmlContent) {
			nextChar := htmlContent[tagStart+len(lowerTag)]
			// The tag name should be followed by whitespace, '>', or '/'
			if nextChar != ' ' && nextChar != '\t' && nextChar != '\n' &&
				nextChar != '\r' && nextChar != '>' && nextChar != '/' {
				pos = tagStart + len(lowerTag)
				continue
			}
		}

		// Find the end of the opening tag
		tagEnd := strings.IndexByte(htmlContent[tagStart:], '>')
		if tagEnd == -1 {
			break
		}
		tagEnd += tagStart + 1

		tagContent := htmlContent[tagStart:tagEnd]

		// Extract requested attributes from this tag
		for _, attrName := range attrNames {
			if value := extractAttributeValue(tagContent, attrName); value != "" {
				results = append(results, value)
			}
		}

		pos = tagEnd
	}

	return results
}

// findTagIgnoreCase performs case-insensitive tag search more efficiently
// by using a combination of Index for candidate positions and EqualFold for verification
func findTagIgnoreCase(html, lowerTag string) int {
	if len(lowerTag) == 0 || len(html) < len(lowerTag) {
		return -1
	}

	// Fast path: try exact match first (most common case)
	if idx := strings.Index(html, lowerTag); idx >= 0 {
		return idx
	}

	// For case-insensitive search, check positions where first character matches (case-insensitive)
	tagLen := len(lowerTag)
	firstChar := lowerTag[0]

	for i := 0; i <= len(html)-tagLen; i++ {
		c := html[i]
		// Quick ASCII case-insensitive check for first character
		cfc := c
		if cfc >= 'A' && cfc <= 'Z' {
			cfc += 32
		}
		if cfc != firstChar {
			continue
		}

		// Found potential match, verify with EqualFold for full case-insensitive comparison
		candidate := html[i : i+tagLen]
		if strings.EqualFold(candidate, lowerTag) {
			return i
		}
	}

	return -1
}

// extractAttributeValue extracts a single attribute value from a tag string.
// It handles quoted (single and double) and unquoted attribute values.
func extractAttributeValue(tagContent, attrName string) string {
	lowerTag := strings.ToLower(tagContent)
	lowerAttr := strings.ToLower(attrName) + "="

	// Find the attribute
	attrIdx := strings.Index(lowerTag, lowerAttr)
	if attrIdx == -1 {
		return ""
	}

	// Verify we're matching a complete attribute name (not a substring)
	if attrIdx > 0 {
		prevChar := lowerTag[attrIdx-1]
		// Attribute should start at beginning or after whitespace
		if prevChar != ' ' && prevChar != '\t' && prevChar != '\n' && prevChar != '\r' {
			return ""
		}
	}

	valueStart := attrIdx + len(attrName) + 1

	// Skip whitespace after '='
	for valueStart < len(tagContent) {
		c := tagContent[valueStart]
		if c != ' ' && c != '\t' {
			break
		}
		valueStart++
	}

	if valueStart >= len(tagContent) {
		return ""
	}

	// Extract quoted or unquoted value
	var value string
	var quote byte

	switch tagContent[valueStart] {
	case '"', '\'':
		// Quoted value
		quote = tagContent[valueStart]
		valueStart++
		valueEnd := strings.IndexByte(tagContent[valueStart:], quote)
		if valueEnd == -1 {
			// Unclosed quote, return rest of tag content
			value = tagContent[valueStart:]
		} else {
			value = tagContent[valueStart : valueStart+valueEnd]
		}
	default:
		// Unquoted value - extract until whitespace or '>'
		valueEnd := valueStart
		for valueEnd < len(tagContent) {
			c := tagContent[valueEnd]
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '>' {
				break
			}
			valueEnd++
		}
		value = tagContent[valueStart:valueEnd]
	}

	return strings.TrimSpace(value)
}

func (p *Processor) extractAudios(node *Node, htmlContent string) []AudioInfo {
	audios := make([]AudioInfo, 0, initialSliceCap)
	seen := make(map[string]bool, initialMapCap)

	internal.WalkNodes(node, func(n *Node) bool {
		if n.Type == ElementNode && n.Data == "audio" {
			if audio := p.parseAudioNode(n); audio.URL != "" && !seen[audio.URL] {
				seen[audio.URL] = true
				audios = append(audios, audio)
			}
		}
		return true
	})

	if len(htmlContent) <= maxHTMLForRegex {
		matches := audioRegex.FindAllString(htmlContent, maxRegexMatches)
		for _, url := range matches {
			if internal.IsValidURL(url) && !seen[url] {
				seen[url] = true
				audios = append(audios, AudioInfo{
					URL:  url,
					Type: internal.DetectAudioType(url),
				})
			}
		}
	}

	return audios
}

func (p *Processor) parseAudioNode(n *Node) AudioInfo {
	audio := AudioInfo{}
	for _, attr := range n.Attr {
		switch attr.Key {
		case "src":
			if !internal.IsValidURL(attr.Val) {
				return AudioInfo{}
			}
			audio.URL = attr.Val
		case "duration":
			audio.Duration = attr.Val
		}
	}

	if audio.URL == "" {
		audio.URL, audio.Type = p.findSourceURL(n)
	}

	if !internal.IsValidURL(audio.URL) {
		return AudioInfo{}
	}

	return audio
}

func (p *Processor) findSourceURL(n *Node) (url, mediaType string) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == ElementNode && c.Data == "source" {
			var srcURL, srcType string
			for _, attr := range c.Attr {
				switch attr.Key {
				case "src":
					srcURL = attr.Val
				case "type":
					srcType = attr.Val
				}
			}
			if srcURL != "" {
				return srcURL, srcType
			}
		}
	}
	return "", ""
}

// generateCacheKey creates a SHA-256 hash for cache key generation.
// Uses multi-point sampling for large documents to better distinguish similar content.
func (p *Processor) generateCacheKey(content string, opts ExtractConfig) string {
	h := sha256.New()

	// Build config key more efficiently
	boolToByte := func(b bool) byte {
		if b {
			return '1'
		}
		return '0'
	}

	configBytes := []byte{
		boolToByte(opts.ExtractArticle), ',',
		boolToByte(opts.PreserveImages), ',',
		boolToByte(opts.PreserveLinks), ',',
		boolToByte(opts.PreserveVideos), ',',
		boolToByte(opts.PreserveAudios), ',',
	}
	h.Write(configBytes)
	h.Write([]byte(opts.InlineImageFormat))
	h.Write([]byte{','})
	h.Write([]byte(opts.TableFormat))
	h.Write([]byte{0})

	contentLen := len(content)
	if contentLen <= maxCacheKeySize {
		h.Write([]byte(content))
	} else {
		// Multi-point sampling for large documents
		// Sample from 5 positions: beginning, 25%, 50%, 75%, and end
		// This better captures differences throughout the document
		sampleCount := 5
		samples := sampleCount
		sampleSize := cacheKeySample / samples
		if sampleSize < 512 {
			sampleSize = 512 // Minimum sample size per position
		}

		for i := 0; i < sampleCount; i++ {
			// Calculate sample position
			var start, end int
			if i == sampleCount-1 {
				// Last sample: take from end
				end = contentLen
				start = contentLen - cacheKeySample
				if start < 0 {
					start = 0
				}
			} else {
				// Evenly distributed samples
				offset := (contentLen * i) / sampleCount
				start = offset
				end = start + sampleSize
				if end > contentLen {
					end = contentLen
				}
			}

			if start < end {
				h.Write([]byte(content[start:end]))
			}
		}

		// Include content length to distinguish documents of different sizes
		h.Write([]byte(strconv.Itoa(contentLen)))
	}

	var buf [32]byte
	sum := h.Sum(buf[:0])
	return hex.EncodeToString(sum)
}

func (p *Processor) extractAllLinksFromContent(htmlContent string, config LinkExtractionConfig) ([]LinkResource, error) {
	if strings.TrimSpace(htmlContent) == "" {
		return []LinkResource{}, nil
	}

	doc, err := stdxhtml.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidHTML, err)
	}

	// Validate depth during extraction to avoid duplicate traversal
	if err := p.validateDepthTraversal(doc, 0); err != nil {
		return nil, err
	}

	baseURL := config.BaseURL
	if config.ResolveRelativeURLs && baseURL == "" {
		baseURL = p.detectBaseURL(doc)
	}

	linkMap := make(map[string]LinkResource, linkMapCap)
	p.extractLinksFromDocument(doc, baseURL, config, linkMap)

	links := make([]LinkResource, 0, len(linkMap))
	for _, link := range linkMap {
		links = append(links, link)
	}

	return links, nil
}

// detectBaseURL attempts to detect base URL from HTML document.
func (p *Processor) detectBaseURL(doc *Node) string {
	if baseNode := internal.FindElementByTag(doc, "base"); baseNode != nil {
		for _, attr := range baseNode.Attr {
			if attr.Key == "href" && attr.Val != "" {
				return p.normalizeBaseURL(attr.Val)
			}
		}
	}

	var canonicalURL, canonicalLink, firstAbsoluteURL string
	internal.WalkNodes(doc, func(n *Node) bool {
		if n.Type != ElementNode {
			return true
		}

		switch n.Data {
		case "meta":
			if canonicalURL == "" {
				var property, content string
				for _, attr := range n.Attr {
					if attr.Key == "property" {
						property = attr.Val
					} else if attr.Key == "content" {
						content = attr.Val
					}
				}
				if (property == "og:url" || property == "canonical") && content != "" {
					canonicalURL = content
				}
			}
		case "link":
			if canonicalLink == "" {
				var rel, href string
				for _, attr := range n.Attr {
					if attr.Key == "rel" {
						rel = attr.Val
					} else if attr.Key == "href" {
						href = attr.Val
					}
				}
				if rel == "canonical" && href != "" {
					canonicalLink = href
				}
			}
		default:
			if firstAbsoluteURL == "" {
				for _, attr := range n.Attr {
					if (attr.Key == "href" || attr.Key == "src") && internal.IsExternalURL(attr.Val) {
						if base := p.extractBaseFromURL(attr.Val); base != "" {
							firstAbsoluteURL = base
							break
						}
					}
				}
			}
		}
		return canonicalURL == "" || canonicalLink == "" || firstAbsoluteURL == ""
	})

	if canonicalURL != "" {
		return p.normalizeBaseURL(canonicalURL)
	}
	if canonicalLink != "" {
		return p.normalizeBaseURL(canonicalLink)
	}
	return firstAbsoluteURL
}

func (p *Processor) normalizeBaseURL(baseURL string) string {
	return internal.NormalizeBaseURL(baseURL)
}

func (p *Processor) extractBaseFromURL(url string) string {
	return internal.ExtractBaseFromURL(url)
}

func (p *Processor) isDifferentDomain(baseURL, targetURL string) bool {
	return internal.IsDifferentDomain(baseURL, targetURL)
}

func (p *Processor) extractDomain(url string) string {
	return internal.ExtractDomain(url)
}

func (p *Processor) resolveURL(baseURL, relativeURL string) string {
	return internal.ResolveURL(baseURL, relativeURL)
}

func (p *Processor) extractLinksFromDocument(doc *Node, baseURL string, config LinkExtractionConfig, linkMap map[string]LinkResource) {
	internal.WalkNodes(doc, func(n *Node) bool {
		if n.Type != ElementNode {
			return true
		}

		switch n.Data {
		case "a":
			if config.IncludeContentLinks || config.IncludeExternalLinks {
				p.extractContentLinks(n, baseURL, config, linkMap)
			}
		case "img":
			if config.IncludeImages {
				p.extractImageLinks(n, baseURL, linkMap)
			}
		case "video":
			if config.IncludeVideos {
				p.extractMediaLink(n, baseURL, linkMap, "video")
			}
		case "audio":
			if config.IncludeAudios {
				p.extractMediaLink(n, baseURL, linkMap, "audio")
			}
		case "source":
			if config.IncludeVideos || config.IncludeAudios {
				p.extractSourceLinks(n, baseURL, linkMap)
			}
		case "link":
			p.extractLinkTagLinks(n, baseURL, config, linkMap)
		case "script":
			if config.IncludeJS {
				p.extractScriptLinks(n, baseURL, linkMap)
			}
		case "iframe", "embed", "object":
			if config.IncludeVideos {
				p.extractEmbedLinks(n, baseURL, linkMap)
			}
		}
		return true
	})
}

func (p *Processor) extractContentLinks(n *Node, baseURL string, config LinkExtractionConfig, linkMap map[string]LinkResource) {
	var href, title string
	for _, attr := range n.Attr {
		switch attr.Key {
		case "href":
			href = attr.Val
		case "title":
			title = attr.Val
		}
	}

	if href == "" || !internal.IsValidURL(href) {
		return
	}

	isExternalOriginal := internal.IsExternalURL(href)

	resolvedURL := href
	if config.ResolveRelativeURLs && baseURL != "" {
		resolvedURL = p.resolveURL(baseURL, href)
	}

	isExternal := isExternalOriginal
	if !isExternalOriginal && baseURL != "" {
		isExternal = p.isDifferentDomain(baseURL, resolvedURL)
	}

	if isExternal && !config.IncludeExternalLinks {
		return
	}
	if !isExternal && !config.IncludeContentLinks {
		return
	}

	if title == "" {
		title = strings.TrimSpace(internal.GetTextContent(n))
		if title == "" {
			title = "Link"
		}
	}

	linkType := "link"

	linkMap[resolvedURL] = LinkResource{
		URL:   resolvedURL,
		Title: title,
		Type:  linkType,
	}
}

func (p *Processor) extractImageLinks(n *Node, baseURL string, linkMap map[string]LinkResource) {
	var src, alt, title string
	for _, attr := range n.Attr {
		switch attr.Key {
		case "src":
			src = attr.Val
		case "alt":
			alt = attr.Val
		case "title":
			title = attr.Val
		}
	}

	if src == "" || !internal.IsValidURL(src) {
		return
	}

	resolvedURL := src
	if baseURL != "" {
		resolvedURL = p.resolveURL(baseURL, src)
	}

	displayName := title
	if displayName == "" {
		displayName = alt
	}
	if displayName == "" {
		if lastSlash := strings.LastIndex(resolvedURL, "/"); lastSlash >= 0 {
			displayName = resolvedURL[lastSlash+1:]
		} else {
			displayName = "Image"
		}
	}

	linkMap[resolvedURL] = LinkResource{
		URL:   resolvedURL,
		Title: displayName,
		Type:  "image",
	}
}

func (p *Processor) extractMediaLink(n *Node, baseURL string, linkMap map[string]LinkResource, mediaType string) {
	var src, title string
	for _, attr := range n.Attr {
		if attr.Key == "src" {
			src = attr.Val
		} else if attr.Key == "title" {
			title = attr.Val
		}
	}

	if src == "" || !internal.IsValidURL(src) {
		return
	}

	resolvedURL := src
	if baseURL != "" {
		resolvedURL = p.resolveURL(baseURL, src)
	}

	displayName := title
	if displayName == "" {
		if lastSlash := strings.LastIndex(resolvedURL, "/"); lastSlash >= 0 {
			displayName = resolvedURL[lastSlash+1:]
		}
		if displayName == "" {
			displayName = strings.ToUpper(mediaType[:1]) + mediaType[1:]
		}
	}

	linkMap[resolvedURL] = LinkResource{
		URL:   resolvedURL,
		Title: displayName,
		Type:  mediaType,
	}
}

func (p *Processor) extractSourceLinks(n *Node, baseURL string, linkMap map[string]LinkResource) {
	var src, mediaType string
	for _, attr := range n.Attr {
		switch attr.Key {
		case "src":
			src = attr.Val
		case "type":
			mediaType = attr.Val
		}
	}

	if src == "" || !internal.IsValidURL(src) {
		return
	}

	resolvedURL := src
	if baseURL != "" {
		resolvedURL = p.resolveURL(baseURL, src)
	}

	resourceType := "media"
	if strings.HasPrefix(mediaType, "video/") {
		resourceType = "video"
	} else if strings.HasPrefix(mediaType, "audio/") {
		resourceType = "audio"
	} else {
		if internal.DetectVideoType(resolvedURL) != "" {
			resourceType = "video"
		} else if internal.DetectAudioType(resolvedURL) != "" {
			resourceType = "audio"
		}
	}

	title := "Media"
	if lastSlash := strings.LastIndex(resolvedURL, "/"); lastSlash >= 0 {
		title = resolvedURL[lastSlash+1:]
	}

	linkMap[resolvedURL] = LinkResource{
		URL:   resolvedURL,
		Title: title,
		Type:  resourceType,
	}
}

func (p *Processor) extractLinkTagLinks(n *Node, baseURL string, config LinkExtractionConfig, linkMap map[string]LinkResource) {
	var href, rel, linkType, title string
	for _, attr := range n.Attr {
		switch attr.Key {
		case "href":
			href = attr.Val
		case "rel":
			rel = attr.Val
		case "type":
			linkType = attr.Val
		case "title":
			title = attr.Val
		}
	}

	if href == "" || !internal.IsValidURL(href) {
		return
	}

	resourceType := "link"
	include := false

	switch rel {
	case "stylesheet":
		if config.IncludeCSS {
			resourceType = "css"
			include = true
		}
		// Prevent fallthrough to next case
	case "icon", "shortcut icon", "apple-touch-icon", "apple-touch-icon-precomposed":
		if config.IncludeIcons {
			resourceType = "icon"
			include = true
		}
	case "preload", "prefetch", "dns-prefetch", "preconnect":
		for _, attr := range n.Attr {
			if attr.Key == "as" {
				switch attr.Val {
				case "style":
					if config.IncludeCSS {
						resourceType = "css"
						include = true
					}
				case "script":
					if config.IncludeJS {
						resourceType = "js"
						include = true
					}
				case "image":
					if config.IncludeImages {
						resourceType = "image"
						include = true
					}
				case "video":
					if config.IncludeVideos {
						resourceType = "video"
						include = true
					}
				case "audio":
					if config.IncludeAudios {
						resourceType = "audio"
						include = true
					}
				}
				break
			}
		}
	default:
		if strings.Contains(linkType, "css") && config.IncludeCSS {
			resourceType = "css"
			include = true
		} else if strings.Contains(linkType, "javascript") && config.IncludeJS {
			resourceType = "js"
			include = true
		}
	}

	if !include {
		return
	}

	resolvedURL := href
	if config.ResolveRelativeURLs && baseURL != "" {
		resolvedURL = p.resolveURL(baseURL, href)
	}

	if title == "" {
		if lastSlash := strings.LastIndex(resolvedURL, "/"); lastSlash >= 0 {
			title = resolvedURL[lastSlash+1:]
		} else {
			if len(resourceType) > 0 {
				title = strings.ToUpper(resourceType[:1]) + resourceType[1:]
			} else {
				title = resourceType
			}
		}
	}

	linkMap[resolvedURL] = LinkResource{
		URL:   resolvedURL,
		Title: title,
		Type:  resourceType,
	}
}

func (p *Processor) extractScriptLinks(n *Node, baseURL string, linkMap map[string]LinkResource) {
	var src, title string
	for _, attr := range n.Attr {
		switch attr.Key {
		case "src":
			src = attr.Val
		case "title":
			title = attr.Val
		}
	}

	if src == "" || !internal.IsValidURL(src) {
		return
	}

	resolvedURL := src
	if baseURL != "" {
		resolvedURL = p.resolveURL(baseURL, src)
	}

	displayName := title
	if displayName == "" {
		if lastSlash := strings.LastIndex(resolvedURL, "/"); lastSlash >= 0 {
			displayName = resolvedURL[lastSlash+1:]
		}
		if displayName == "" {
			displayName = "Script"
		}
	}

	linkMap[resolvedURL] = LinkResource{
		URL:   resolvedURL,
		Title: displayName,
		Type:  "js",
	}
}

func (p *Processor) extractEmbedLinks(n *Node, baseURL string, linkMap map[string]LinkResource) {
	var src, title string
	for _, attr := range n.Attr {
		switch attr.Key {
		case "src", "data":
			src = attr.Val
		case "title":
			title = attr.Val
		}
	}

	if src == "" || !internal.IsValidURL(src) {
		return
	}

	// Only include if it's a video URL (includes embed patterns)
	if !internal.IsVideoURL(src) {
		return
	}

	resolvedURL := src
	if baseURL != "" {
		resolvedURL = p.resolveURL(baseURL, src)
	}

	if title == "" {
		// Try to extract platform name from URL
		if strings.Contains(resolvedURL, "youtube") {
			title = "YouTube Video"
		} else if strings.Contains(resolvedURL, "vimeo") {
			title = "Vimeo Video"
		} else if strings.Contains(resolvedURL, "dailymotion") {
			title = "Dailymotion Video"
		} else {
			title = "Embedded Video"
		}
	}

	linkMap[resolvedURL] = LinkResource{
		URL:   resolvedURL,
		Title: title,
		Type:  "video",
	}
}

// ExtractToMarkdown converts HTML bytes to Markdown with automatic encoding detection.
// The method automatically detects character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
//
// Parameters:
//
//	htmlBytes - Raw HTML bytes (auto-detects encoding)
//
// Returns:
//
//	string - Markdown content in UTF-8
//	error - Error if conversion fails
//
// Example:
//
//	// HTTP response
//	resp, _ := http.Get(url)
//	bytes, _ := io.ReadAll(resp.Body)
//	markdown, _ := html.ExtractToMarkdown(bytes)
//
//	// File
//	bytes, _ := os.ReadFile("document.html")
//	markdown, _ := html.ExtractToMarkdown(bytes)
func ExtractToMarkdown(htmlBytes []byte) (string, error) {
	processor, _ := New()
	defer processor.Close()

	config := DefaultExtractConfig()
	config.InlineImageFormat = "markdown"
	result, err := processor.Extract(htmlBytes, config)
	if err != nil {
		return "", err
	}

	return result.Text, nil
}

// jsonResult wraps Result for custom JSON marshaling with duration formatting
type jsonResult struct {
	Text             string      `json:"text"`
	Title            string      `json:"title"`
	Images           []ImageInfo `json:"images,omitempty"`
	Links            []LinkInfo  `json:"links,omitempty"`
	Videos           []VideoInfo `json:"videos,omitempty"`
	Audios           []AudioInfo `json:"audios,omitempty"`
	ProcessingTimeMS int64       `json:"processing_time_ms"`
	WordCount        int         `json:"word_count"`
	ReadingTimeMS    int64       `json:"reading_time_ms"`
}

func (r *Result) MarshalJSON() ([]byte, error) {
	jr := jsonResult{
		Text:             r.Text,
		Title:            r.Title,
		Images:           r.Images,
		Links:            r.Links,
		Videos:           r.Videos,
		Audios:           r.Audios,
		ProcessingTimeMS: r.ProcessingTime.Milliseconds(),
		WordCount:        r.WordCount,
		ReadingTimeMS:    r.ReadingTime.Milliseconds(),
	}
	return json.Marshal(jr)
}

func ExtractToJSON(htmlBytes []byte) ([]byte, error) {
	result, err := Extract(htmlBytes)
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}
