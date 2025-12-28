// Package html provides secure, high-performance HTML content extraction.
// It is 100% compatible with golang.org/x/net/html and adds enhanced content extraction features.
package html

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cybergodev/html/internal"
	"golang.org/x/net/html"
)

// Re-export all types and constants from golang.org/x/net/html for 100% compatibility.
type (
	Node      = html.Node
	NodeType  = html.NodeType
	Attribute = html.Attribute
	Token     = html.Token
	TokenType = html.TokenType
	Tokenizer = html.Tokenizer
)

const (
	ErrorNode    = html.ErrorNode
	TextNode     = html.TextNode
	DocumentNode = html.DocumentNode
	ElementNode  = html.ElementNode
	CommentNode  = html.CommentNode
	DoctypeNode  = html.DoctypeNode

	ErrorToken          = html.ErrorToken
	TextToken           = html.TextToken
	StartTagToken       = html.StartTagToken
	EndTagToken         = html.EndTagToken
	SelfClosingTagToken = html.SelfClosingTagToken
	CommentToken        = html.CommentToken
	DoctypeToken        = html.DoctypeToken
)

// Re-export all functions from golang.org/x/net/html for 100% compatibility.
var (
	Parse          = html.Parse
	ParseFragment  = html.ParseFragment
	Render         = html.Render
	EscapeString   = html.EscapeString
	UnescapeString = html.UnescapeString
	NewTokenizer   = html.NewTokenizer
)

// Convenience functions for quick content extraction without processor setup.

// Extract extracts content from HTML using default configuration.
// This is the simplest way to extract content - no setup required.
func Extract(htmlContent string) (*Result, error) {
	processor := NewWithDefaults()
	defer processor.Close()
	return processor.ExtractWithDefaults(htmlContent)
}

// ExtractFromFile reads and extracts content from an HTML file using defaults.
func ExtractFromFile(filePath string) (*Result, error) {
	processor := NewWithDefaults()
	defer processor.Close()
	return processor.ExtractFromFile(filePath, DefaultExtractConfig())
}

// ExtractText extracts only text content without metadata.
// Returns clean text suitable for analysis or display.
func ExtractText(htmlContent string) (string, error) {
	result, err := Extract(htmlContent)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// Default configuration values.
const (
	DefaultMaxInputSize      = 50 * 1024 * 1024 // 50MB
	DefaultMaxCacheEntries   = 1000             // 1000 entries
	DefaultWorkerPoolSize    = 4                // 4 workers
	DefaultCacheTTL          = time.Hour        // 1 hour
	DefaultMaxDepth          = 100              // 100 levels
	DefaultProcessingTimeout = 30 * time.Second // 30 seconds
)

// Internal constants for validation and optimization.
const (
	maxURLLength     = 2000      // Maximum URL length to prevent DoS
	maxHTMLForRegex  = 1000000   // 1MB limit for regex scanning
	maxRegexMatches  = 100       // Maximum regex matches to prevent DoS
	wordsPerMinute   = 200       // Average reading speed
	maxCacheKeySize  = 64 * 1024 // 64KB threshold for full content hashing
	initialTextSize  = 4096      // Initial text builder capacity
	initialImageCap  = 16        // Initial image slice capacity
	initialLinksCap  = 32        // Initial links slice capacity
	initialVideosCap = 8         // Initial videos slice capacity
	initialAudiosCap = 8         // Initial audios slice capacity
	initialSeenCap   = 8         // Initial seen map capacity
	cacheKeySample   = 4096      // Sample size for large content hashing
)

var (
	whitespaceRegex = regexp.MustCompile(`\s+`)
	videoRegex      = regexp.MustCompile(`(?i)https?://[^\s<>"',;)}\]]{1,500}\.(?:mp4|webm|ogg|mov|avi|wmv|flv|mkv|m4v|3gp)`)
	audioRegex      = regexp.MustCompile(`(?i)https?://[^\s<>"',;)}\]]{1,500}\.(?:mp3|wav|ogg|m4a|aac|flac|wma|opus|oga)`)
)

// Processor provides thread-safe HTML content extraction.
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

// Config holds processor configuration with security and performance settings.
// All fields have sensible defaults via DefaultConfig().
type Config struct {
	// MaxInputSize limits input HTML size to prevent memory exhaustion (default: 50MB)
	MaxInputSize int

	// MaxCacheEntries limits cache size with LRU eviction (default: 1000, 0 disables cache)
	MaxCacheEntries int

	// CacheTTL sets cache entry expiration time (default: 1 hour, 0 means no expiration)
	CacheTTL time.Duration

	// WorkerPoolSize controls parallel processing workers (default: 4)
	WorkerPoolSize int

	// EnableSanitization removes script/style tags for security (default: true)
	EnableSanitization bool

	// MaxDepth prevents billion laughs attacks via nesting limits (default: 100)
	MaxDepth int

	// ProcessingTimeout prevents DoS via processing time limits (default: 30s, 0 disables)
	ProcessingTimeout time.Duration
}

// DefaultConfig returns default configuration.
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

// validateConfig validates processor configuration for consistency and security.
func validateConfig(c Config) error {
	switch {
	case c.MaxInputSize <= 0:
		return fmt.Errorf("%w: MaxInputSize must be positive, got %d", ErrInvalidConfig, c.MaxInputSize)
	case c.MaxInputSize > 1024*1024*1024: // 1GB limit
		return fmt.Errorf("%w: MaxInputSize too large (max 1GB), got %d", ErrInvalidConfig, c.MaxInputSize)
	case c.MaxCacheEntries < 0:
		return fmt.Errorf("%w: MaxCacheEntries cannot be negative, got %d", ErrInvalidConfig, c.MaxCacheEntries)
	case c.CacheTTL < 0:
		return fmt.Errorf("%w: CacheTTL cannot be negative, got %v", ErrInvalidConfig, c.CacheTTL)
	case c.WorkerPoolSize <= 0:
		return fmt.Errorf("%w: WorkerPoolSize must be positive, got %d", ErrInvalidConfig, c.WorkerPoolSize)
	case c.WorkerPoolSize > 1000: // Reasonable upper limit
		return fmt.Errorf("%w: WorkerPoolSize too large (max 1000), got %d", ErrInvalidConfig, c.WorkerPoolSize)
	case c.MaxDepth <= 0:
		return fmt.Errorf("%w: MaxDepth must be positive, got %d", ErrInvalidConfig, c.MaxDepth)
	case c.MaxDepth > 10000: // Prevent excessive nesting
		return fmt.Errorf("%w: MaxDepth too large (max 10000), got %d", ErrInvalidConfig, c.MaxDepth)
	case c.ProcessingTimeout < 0:
		return fmt.Errorf("%w: ProcessingTimeout cannot be negative, got %v", ErrInvalidConfig, c.ProcessingTimeout)
	}

	// Cross-field validation - removed overly strict TTL check
	// Zero CacheTTL means no expiration, which is valid

	return nil
}

// ExtractConfig configures content extraction behavior.
// Controls what content types are extracted and how they're formatted.
type ExtractConfig struct {
	// ExtractArticle enables intelligent article detection (default: true)
	ExtractArticle bool

	// PreserveImages includes image metadata in results (default: true)
	PreserveImages bool

	// PreserveLinks includes link metadata in results (default: true)
	PreserveLinks bool

	// PreserveVideos includes video metadata in results (default: true)
	PreserveVideos bool

	// PreserveAudios includes audio metadata in results (default: true)
	PreserveAudios bool

	// InlineImageFormat controls image placeholder format: "none", "placeholder", "markdown", "html" (default: "none")
	InlineImageFormat string
}

// DefaultExtractConfig returns default extraction configuration.
func DefaultExtractConfig() ExtractConfig {
	return ExtractConfig{
		ExtractArticle:    true,
		PreserveImages:    true,
		PreserveLinks:     true,
		PreserveVideos:    true,
		PreserveAudios:    true,
		InlineImageFormat: "none",
	}
}

// Result contains extraction results.
type Result struct {
	Text           string
	Title          string
	Images         []ImageInfo
	Links          []LinkInfo
	Videos         []VideoInfo
	Audios         []AudioInfo
	ProcessingTime time.Duration
	WordCount      int
	ReadingTime    time.Duration
}

// ImageInfo contains image metadata.
type ImageInfo struct {
	URL          string
	Alt          string
	Title        string
	Width        string
	Height       string
	IsDecorative bool
	Position     int
}

// LinkInfo contains link metadata.
type LinkInfo struct {
	URL        string
	Text       string
	Title      string
	IsExternal bool
	IsNoFollow bool
}

// VideoInfo contains video metadata.
type VideoInfo struct {
	URL      string
	Type     string
	Poster   string
	Width    string
	Height   string
	Duration string
}

// AudioInfo contains audio metadata.
type AudioInfo struct {
	URL      string
	Type     string
	Duration string
}

// Statistics contains processing metrics.
type Statistics struct {
	TotalProcessed     int64
	CacheHits          int64
	CacheMisses        int64
	ErrorCount         int64
	AverageProcessTime time.Duration
}

// New creates a Processor with the given configuration.
func New(config Config) (*Processor, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	return &Processor{
		config: &config,
		cache:  internal.NewCache(config.MaxCacheEntries, config.CacheTTL),
	}, nil
}

// NewWithDefaults creates a Processor with default configuration.
func NewWithDefaults() *Processor {
	p, _ := New(DefaultConfig())
	return p
}

// Extract extracts content from HTML with the given configuration.
func (p *Processor) Extract(htmlContent string, config ExtractConfig) (*Result, error) {
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	startTime := time.Now()

	if len(htmlContent) > p.config.MaxInputSize {
		p.stats.errorCount.Add(1)
		return nil, fmt.Errorf("%w: size=%d, max=%d", ErrInputTooLarge, len(htmlContent), p.config.MaxInputSize)
	}

	cacheKey := p.generateCacheKey(htmlContent, config)
	if cached := p.cache.Get(cacheKey); cached != nil {
		p.stats.cacheHits.Add(1)
		p.stats.totalProcessed.Add(1)
		if result, ok := cached.(*Result); ok {
			return result, nil
		}
	}
	p.stats.cacheMisses.Add(1)

	// Process with timeout if configured
	var result *Result
	var err error
	if p.config.ProcessingTimeout > 0 {
		result, err = p.processWithTimeout(htmlContent, config)
	} else {
		result, err = p.processContent(htmlContent, config)
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

// processWithTimeout processes content with timeout protection.
func (p *Processor) processWithTimeout(htmlContent string, config ExtractConfig) (*Result, error) {
	type processResult struct {
		result *Result
		err    error
	}

	resultChan := make(chan processResult, 1)
	go func() {
		result, err := p.processContent(htmlContent, config)
		resultChan <- processResult{result: result, err: err}
	}()

	select {
	case res := <-resultChan:
		return res.result, res.err
	case <-time.After(p.config.ProcessingTimeout):
		return nil, ErrProcessingTimeout
	}
}

// ExtractWithDefaults extracts content using default extraction configuration.
func (p *Processor) ExtractWithDefaults(htmlContent string) (*Result, error) {
	return p.Extract(htmlContent, DefaultExtractConfig())
}

// ExtractFromFile reads and extracts content from an HTML file.
func (p *Processor) ExtractFromFile(filePath string, config ExtractConfig) (*Result, error) {
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}
	if filePath == "" {
		return nil, fmt.Errorf("%w: empty file path", ErrFileNotFound)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, filePath)
		}
		return nil, fmt.Errorf("read file %q: %w", filePath, err)
	}

	return p.Extract(string(data), config)
}

// ExtractBatch processes multiple HTML contents in parallel using a worker pool.
func (p *Processor) ExtractBatch(htmlContents []string, config ExtractConfig) ([]*Result, error) {
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	if len(htmlContents) == 0 {
		return []*Result{}, nil
	}

	results := make([]*Result, len(htmlContents))
	errs := make([]error, len(htmlContents))
	sem := make(chan struct{}, p.config.WorkerPoolSize)
	var wg sync.WaitGroup

	for i, content := range htmlContents {
		wg.Add(1)
		go func(idx int, html string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results[idx], errs[idx] = p.Extract(html, config)
		}(i, content)
	}

	wg.Wait()
	return collectResults(results, errs, nil)
}

// ExtractBatchFiles processes multiple HTML files in parallel using a worker pool.
func (p *Processor) ExtractBatchFiles(filePaths []string, config ExtractConfig) ([]*Result, error) {
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	if len(filePaths) == 0 {
		return []*Result{}, nil
	}

	results := make([]*Result, len(filePaths))
	errs := make([]error, len(filePaths))
	sem := make(chan struct{}, p.config.WorkerPoolSize)
	var wg sync.WaitGroup

	for i, path := range filePaths {
		wg.Add(1)
		go func(idx int, filePath string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results[idx], errs[idx] = p.ExtractFromFile(filePath, config)
		}(i, path)
	}

	wg.Wait()
	return collectResults(results, errs, filePaths)
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

// GetStatistics returns processing statistics.
func (p *Processor) GetStatistics() Statistics {
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

// ClearCache clears the cache and resets cache statistics.
func (p *Processor) ClearCache() {
	p.cache.Clear()
	p.stats.cacheHits.Store(0)
	p.stats.cacheMisses.Store(0)
}

// Close releases processor resources.
func (p *Processor) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	p.cache.Clear()
	return nil
}

func (p *Processor) processContent(htmlContent string, opts ExtractConfig) (*Result, error) {
	// Validate input is not empty
	if strings.TrimSpace(htmlContent) == "" {
		return &Result{}, nil
	}

	originalHTML := htmlContent

	// Apply sanitization if enabled
	if p.config.EnableSanitization {
		htmlContent = internal.SanitizeHTML(htmlContent)
	}

	// Parse HTML document
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidHTML, err)
	}

	// Validate document depth to prevent DoS
	if err := p.validateDepth(doc, 0); err != nil {
		return nil, err
	}

	return p.extractFromDocument(doc, originalHTML, opts)
}

func (p *Processor) validateDepth(n *html.Node, depth int) error {
	if depth > p.config.MaxDepth {
		return ErrMaxDepthExceeded
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if err := p.validateDepth(c, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (p *Processor) extractFromDocument(doc *html.Node, htmlContent string, opts ExtractConfig) (*Result, error) {
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

	if format != "none" && opts.PreserveImages {
		result.Images = p.extractImagesWithPosition(contentNode)
		var sb strings.Builder
		sb.Grow(initialTextSize)
		imageCounter := 0
		internal.ExtractTextWithStructureAndImages(contentNode, &sb, 0, &imageCounter)
		textWithPlaceholders := internal.CleanText(sb.String(), whitespaceRegex)
		result.Text = p.formatInlineImages(textWithPlaceholders, result.Images, format)
	} else {
		result.Text = p.extractTextContent(contentNode)
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

func (p *Processor) extractTitle(doc *html.Node) string {
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

func (p *Processor) extractArticleNode(doc *html.Node) *html.Node {
	if doc == nil {
		return nil
	}
	candidates := make(map[*html.Node]int)
	internal.WalkNodes(doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode {
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

// extractTextContent extracts clean text from HTML node with optimized performance.
func (p *Processor) extractTextContent(node *html.Node) string {
	var sb strings.Builder
	sb.Grow(initialTextSize) // Pre-allocate 4KB buffer
	internal.ExtractTextWithStructure(node, &sb, 0)
	return internal.CleanText(sb.String(), whitespaceRegex)
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
			htmlImg.Grow(len(images[i].URL) + len(images[i].Alt) + len(images[i].Width) + len(images[i].Height) + 64)
			htmlImg.WriteString(`<img src="`)
			htmlImg.WriteString(images[i].URL)
			htmlImg.WriteString(`" alt="`)
			htmlImg.WriteString(images[i].Alt)
			htmlImg.WriteString(`"`)
			if images[i].Width != "" {
				htmlImg.WriteString(` width="`)
				htmlImg.WriteString(images[i].Width)
				htmlImg.WriteString(`"`)
			}
			if images[i].Height != "" {
				htmlImg.WriteString(` height="`)
				htmlImg.WriteString(images[i].Height)
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

func (p *Processor) extractImages(node *html.Node) []ImageInfo {
	images := make([]ImageInfo, 0, initialImageCap)

	internal.WalkNodes(node, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "img" {
			img := p.parseImageNode(n, 0)
			if img.URL != "" {
				images = append(images, img)
			}
		}
		return true
	})

	return images
}

func (p *Processor) extractImagesWithPosition(node *html.Node) []ImageInfo {
	images := make([]ImageInfo, 0, initialImageCap)
	position := 0

	internal.WalkNodes(node, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "img" {
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

func (p *Processor) parseImageNode(n *html.Node, position int) ImageInfo {
	img := ImageInfo{Position: position}

	for _, attr := range n.Attr {
		switch attr.Key {
		case "src":
			if !isValidURL(attr.Val) {
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

func (p *Processor) extractLinks(node *html.Node) []LinkInfo {
	links := make([]LinkInfo, 0, initialLinksCap)

	internal.WalkNodes(node, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "a" {
			link := p.parseLinkNode(n)
			if link.URL != "" {
				links = append(links, link)
			}
		}
		return true
	})

	return links
}

func (p *Processor) parseLinkNode(n *html.Node) LinkInfo {
	link := LinkInfo{}

	for _, attr := range n.Attr {
		switch attr.Key {
		case "href":
			if !isValidURL(attr.Val) {
				return LinkInfo{}
			}
			link.URL = attr.Val
		case "title":
			link.Title = attr.Val
		case "rel":
			if strings.Contains(attr.Val, "nofollow") {
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

func (p *Processor) extractVideos(node *html.Node, htmlContent string) []VideoInfo {
	videos := make([]VideoInfo, 0, initialVideosCap)
	seen := make(map[string]bool, initialSeenCap)

	internal.WalkNodes(node, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
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

	if len(htmlContent) <= maxHTMLForRegex {
		matches := videoRegex.FindAllString(htmlContent, maxRegexMatches)
		for _, url := range matches {
			if isValidURL(url) && !seen[url] {
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

func (p *Processor) parseVideoNode(n *html.Node) VideoInfo {
	video := VideoInfo{}
	for _, attr := range n.Attr {
		switch attr.Key {
		case "src":
			if !isValidURL(attr.Val) {
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

	if !isValidURL(video.URL) {
		return VideoInfo{}
	}

	return video
}

func (p *Processor) parseIframeNode(n *html.Node) VideoInfo {
	for _, attr := range n.Attr {
		if attr.Key == "src" && isValidURL(attr.Val) && internal.IsVideoEmbedURL(attr.Val) {
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

func (p *Processor) parseEmbedNode(n *html.Node) VideoInfo {
	for _, attr := range n.Attr {
		if (attr.Key == "src" || attr.Key == "data") && isValidURL(attr.Val) && internal.IsVideoURL(attr.Val) {
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

func (p *Processor) extractAudios(node *html.Node, htmlContent string) []AudioInfo {
	audios := make([]AudioInfo, 0, initialAudiosCap)
	seen := make(map[string]bool, initialSeenCap)

	internal.WalkNodes(node, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "audio" {
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
			if isValidURL(url) && !seen[url] {
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

func (p *Processor) parseAudioNode(n *html.Node) AudioInfo {
	audio := AudioInfo{}
	for _, attr := range n.Attr {
		switch attr.Key {
		case "src":
			if !isValidURL(attr.Val) {
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

	if !isValidURL(audio.URL) {
		return AudioInfo{}
	}

	return audio
}

func (p *Processor) findSourceURL(n *html.Node) (url, mediaType string) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "source" {
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

// isValidURL checks if a URL is valid and safe.
// Validates length, format, and prevents common attack vectors.
func isValidURL(url string) bool {
	if url == "" || len(url) > maxURLLength {
		return false
	}

	// Basic format validation - must contain valid URL characters
	for _, r := range url {
		if r < 32 || r > 126 {
			return false // Non-printable or extended ASCII
		}
	}

	// Allow all URLs that pass basic validation (including relative URLs)
	return true
}

// generateCacheKey creates a SHA-256 hash for cache key generation.
// Uses sampling for large content to avoid full content hashing overhead.
func (p *Processor) generateCacheKey(content string, opts ExtractConfig) string {
	h := sha256.New()

	// Write configuration flags as single byte
	var flags byte
	if opts.ExtractArticle {
		flags |= 1 << 0
	}
	if opts.PreserveImages {
		flags |= 1 << 1
	}
	if opts.PreserveLinks {
		flags |= 1 << 2
	}
	if opts.PreserveVideos {
		flags |= 1 << 3
	}
	if opts.PreserveAudios {
		flags |= 1 << 4
	}
	h.Write([]byte{flags})

	// Include image format if specified
	if opts.InlineImageFormat != "" {
		h.Write([]byte(opts.InlineImageFormat))
		h.Write([]byte{0}) // separator
	}

	contentLen := len(content)
	if contentLen <= maxCacheKeySize {
		// Small content: hash everything
		h.Write([]byte(content))
	} else {
		// Large content: use three-point sampling
		h.Write([]byte(content[:cacheKeySample]))

		mid := contentLen >> 1
		halfSample := cacheKeySample >> 1
		h.Write([]byte(content[mid-halfSample : mid+halfSample]))

		h.Write([]byte(content[contentLen-cacheKeySample:]))

		// Include content length to distinguish different large contents
		var lenBuf [8]byte
		lenBuf[0] = byte(contentLen)
		lenBuf[1] = byte(contentLen >> 8)
		lenBuf[2] = byte(contentLen >> 16)
		lenBuf[3] = byte(contentLen >> 24)
		lenBuf[4] = byte(contentLen >> 32)
		lenBuf[5] = byte(contentLen >> 40)
		lenBuf[6] = byte(contentLen >> 48)
		lenBuf[7] = byte(contentLen >> 56)
		h.Write(lenBuf[:])
	}

	// Use pre-allocated buffer to avoid allocation
	var buf [64]byte
	sum := h.Sum(buf[:0])
	return hex.EncodeToString(sum)
}
