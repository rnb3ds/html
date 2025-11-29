// Package html provides secure, high-performance HTML content extraction.
// It is 100% compatible with golang.org/x/net/html and adds enhanced content extraction features.
package html

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

const (
	DefaultMaxInputSize      = 50 * 1024 * 1024
	DefaultMaxCacheEntries   = 1000
	DefaultProcessingTimeout = 30 * time.Second
	DefaultWorkerPoolSize    = 4
	DefaultCacheTTL          = time.Hour
	DefaultMaxDepth          = 100

	maxURLLength    = 2000
	maxHTMLForRegex = 1000000
	maxRegexMatches = 100
	wordsPerMinute  = 200
	initialTextSize = 4096
)

var (
	ErrInputTooLarge    = errors.New("html: input size exceeds maximum allowed")
	ErrInvalidHTML      = errors.New("html: invalid HTML structure")
	ErrProcessorClosed  = errors.New("html: processor has been closed")
	ErrMaxDepthExceeded = errors.New("html: maximum nesting depth exceeded")
	ErrInvalidConfig    = errors.New("html: invalid configuration")
)

// Processor provides thread-safe HTML content extraction.
// All methods are safe for concurrent use by multiple goroutines.
type Processor struct {
	// Immutable after creation - no lock needed
	config          *Config
	whitespaceRegex *regexp.Regexp
	videoRegex      *regexp.Regexp
	audioRegex      *regexp.Regexp

	// Thread-safe cache with internal locking
	cache *internal.Cache

	// Atomic flag for closed state
	closed atomic.Bool

	// Atomic statistics counters
	stats struct {
		totalProcessed   atomic.Int64
		cacheHits        atomic.Int64
		cacheMisses      atomic.Int64
		errorCount       atomic.Int64
		totalProcessTime atomic.Int64
	}
}

// Config holds processor configuration.
type Config struct {
	MaxInputSize       int
	ProcessingTimeout  time.Duration
	MaxCacheEntries    int
	CacheTTL           time.Duration
	WorkerPoolSize     int
	EnableSanitization bool
	MaxDepth           int
}

// DefaultConfig returns default configuration with secure settings.
func DefaultConfig() Config {
	return Config{
		MaxInputSize:       DefaultMaxInputSize,
		ProcessingTimeout:  DefaultProcessingTimeout,
		MaxCacheEntries:    DefaultMaxCacheEntries,
		CacheTTL:           DefaultCacheTTL,
		WorkerPoolSize:     DefaultWorkerPoolSize,
		EnableSanitization: true,
		MaxDepth:           DefaultMaxDepth,
	}
}

func validateConfig(c Config) error {
	if c.MaxInputSize <= 0 {
		return fmt.Errorf("%w: MaxInputSize must be positive", ErrInvalidConfig)
	}
	if c.ProcessingTimeout <= 0 {
		return fmt.Errorf("%w: ProcessingTimeout must be positive", ErrInvalidConfig)
	}
	if c.MaxCacheEntries < 0 {
		return fmt.Errorf("%w: MaxCacheEntries cannot be negative", ErrInvalidConfig)
	}
	if c.CacheTTL < 0 {
		return fmt.Errorf("%w: CacheTTL cannot be negative", ErrInvalidConfig)
	}
	if c.WorkerPoolSize <= 0 {
		return fmt.Errorf("%w: WorkerPoolSize must be positive", ErrInvalidConfig)
	}
	if c.MaxDepth <= 0 {
		return fmt.Errorf("%w: MaxDepth must be positive", ErrInvalidConfig)
	}
	return nil
}

// ExtractConfig provides extraction configuration.
type ExtractConfig struct {
	ExtractArticle    bool
	PreserveImages    bool
	PreserveLinks     bool
	PreserveVideos    bool
	PreserveAudios    bool
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

// Result represents extraction results.
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

// Statistics tracks processing metrics.
type Statistics struct {
	TotalProcessed     int64
	CacheHits          int64
	CacheMisses        int64
	ErrorCount         int64
	AverageProcessTime time.Duration
}

// New creates a new Processor with the given configuration.
func New(config Config) (*Processor, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	whitespaceRegex, err := regexp.Compile(`\s+`)
	if err != nil {
		return nil, fmt.Errorf("compile whitespace regex: %w", err)
	}

	videoRegex, err := regexp.Compile(`(?i)https?://[^\s<>"',;)}\]]{1,500}\.(?:mp4|webm|ogg|mov|avi|wmv|flv|mkv|m4v|3gp)`)
	if err != nil {
		return nil, fmt.Errorf("compile video regex: %w", err)
	}

	audioRegex, err := regexp.Compile(`(?i)https?://[^\s<>"',;)}\]]{1,500}\.(?:mp3|wav|ogg|m4a|aac|flac|wma|opus|oga)`)
	if err != nil {
		return nil, fmt.Errorf("compile audio regex: %w", err)
	}

	p := &Processor{
		config:          &config,
		cache:           internal.NewCache(config.MaxCacheEntries, config.CacheTTL),
		whitespaceRegex: whitespaceRegex,
		videoRegex:      videoRegex,
		audioRegex:      audioRegex,
	}

	return p, nil
}

// NewWithDefaults creates a new Processor with default configuration.
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

	result, err := p.processContent(htmlContent, config)
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

// ExtractWithDefaults extracts content using default extraction configuration.
func (p *Processor) ExtractWithDefaults(htmlContent string) (*Result, error) {
	return p.Extract(htmlContent, DefaultExtractConfig())
}

// ExtractFromFile reads HTML from a file and extracts content.
func (p *Processor) ExtractFromFile(filePath string, config ExtractConfig) (*Result, error) {
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
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

	type item struct {
		index   int
		content string
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
	var errorMsgs []string
	successCount := 0

	for i, err := range errs {
		if err != nil {
			if names != nil {
				errorMsgs = append(errorMsgs, fmt.Sprintf("%s: %v", names[i], err))
			} else {
				errorMsgs = append(errorMsgs, fmt.Sprintf("item %d: %v", i, err))
			}
		} else {
			successCount++
		}
	}

	if successCount == 0 {
		return results, fmt.Errorf("all items failed: %s", strings.Join(errorMsgs, "; "))
	}

	if len(errorMsgs) > 0 {
		return results, fmt.Errorf("partial failure (%d/%d succeeded): %s", successCount, len(results), strings.Join(errorMsgs, "; "))
	}

	return results, nil
}

// GetStatistics returns current processing statistics.
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

// ClearCache empties the cache and resets cache statistics.
func (p *Processor) ClearCache() {
	p.cache.Clear()
	p.stats.cacheHits.Store(0)
	p.stats.cacheMisses.Store(0)
}

// Close releases all resources held by the processor.
func (p *Processor) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}

	p.cache.Clear()
	return nil
}

func (p *Processor) processContent(htmlContent string, opts ExtractConfig) (*Result, error) {
	if strings.TrimSpace(htmlContent) == "" {
		return &Result{
			Text:        "",
			Title:       "",
			Images:      []ImageInfo{},
			Links:       []LinkInfo{},
			Videos:      []VideoInfo{},
			Audios:      []AudioInfo{},
			WordCount:   0,
			ReadingTime: 0,
		}, nil
	}

	originalHTML := htmlContent

	if p.config.EnableSanitization {
		htmlContent = internal.SanitizeHTML(htmlContent)
	}

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidHTML, err)
	}

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
	result := &Result{
		Images: []ImageInfo{},
		Links:  []LinkInfo{},
		Videos: []VideoInfo{},
		Audios: []AudioInfo{},
	}

	result.Title = p.extractTitle(doc)

	var contentNode *html.Node
	if opts.ExtractArticle {
		contentNode = p.extractArticleNode(doc)
	}
	if contentNode == nil {
		contentNode = doc
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
		textWithPlaceholders := internal.CleanText(sb.String(), p.whitespaceRegex)

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
		title := internal.GetTextContent(titleNode)
		if title != "" {
			return title
		}
	}
	if h1Node := internal.FindElementByTag(doc, "h1"); h1Node != nil {
		title := internal.GetTextContent(h1Node)
		if title != "" {
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
			score := internal.ScoreContentNode(n)
			if score > 0 {
				candidates[n] = score
			}
		}
		return true
	})

	bestNode := internal.SelectBestCandidate(candidates)

	if bestNode == nil {
		bodyNode := internal.FindElementByTag(doc, "body")
		if bodyNode != nil {
			return bodyNode
		}
	}

	return bestNode
}

func (p *Processor) extractTextContent(node *html.Node) string {
	if node == nil {
		return ""
	}
	var sb strings.Builder
	sb.Grow(initialTextSize)
	internal.ExtractTextWithStructure(node, &sb, 0)
	return internal.CleanText(sb.String(), p.whitespaceRegex)
}

func (p *Processor) formatInlineImages(textWithPlaceholders string, images []ImageInfo, format string) string {
	if len(images) == 0 || format == "placeholder" || format == "none" {
		return textWithPlaceholders
	}

	imageMap := make(map[int]ImageInfo, len(images))
	for _, img := range images {
		imageMap[img.Position] = img
	}

	text := textWithPlaceholders
	switch format {
	case "markdown":
		for i := 1; i <= len(images); i++ {
			if img, ok := imageMap[i]; ok {
				placeholder := fmt.Sprintf("[IMAGE:%d]", i)
				altText := img.Alt
				if altText == "" {
					altText = fmt.Sprintf("Image %d", i)
				}
				markdown := fmt.Sprintf("![%s](%s)", altText, img.URL)
				text = strings.ReplaceAll(text, placeholder, markdown)
			}
		}
	case "html":
		for i := 1; i <= len(images); i++ {
			if img, ok := imageMap[i]; ok {
				placeholder := fmt.Sprintf("[IMAGE:%d]", i)
				var htmlImg strings.Builder
				htmlImg.Grow(128)
				htmlImg.WriteString(`<img src="`)
				htmlImg.WriteString(img.URL)
				htmlImg.WriteString(`" alt="`)
				htmlImg.WriteString(img.Alt)
				htmlImg.WriteString(`"`)
				if img.Width != "" {
					htmlImg.WriteString(` width="`)
					htmlImg.WriteString(img.Width)
					htmlImg.WriteString(`"`)
				}
				if img.Height != "" {
					htmlImg.WriteString(` height="`)
					htmlImg.WriteString(img.Height)
					htmlImg.WriteString(`"`)
				}
				htmlImg.WriteString(">")
				text = strings.ReplaceAll(text, placeholder, htmlImg.String())
			}
		}
	}

	return text
}

func (p *Processor) extractImages(node *html.Node) []ImageInfo {
	if node == nil {
		return []ImageInfo{}
	}

	images := make([]ImageInfo, 0, 16)

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
	if node == nil {
		return []ImageInfo{}
	}

	images := make([]ImageInfo, 0, 16)
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

	if img.URL != "" && len(img.URL) <= maxURLLength {
		img.IsDecorative = img.Alt == ""
		return img
	}

	return ImageInfo{}
}

func (p *Processor) extractLinks(node *html.Node) []LinkInfo {
	if node == nil {
		return []LinkInfo{}
	}

	links := make([]LinkInfo, 0, 32)

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
			link.URL = attr.Val
		case "title":
			link.Title = attr.Val
		case "rel":
			if strings.Contains(attr.Val, "nofollow") {
				link.IsNoFollow = true
			}
		}
	}

	if link.URL != "" && len(link.URL) <= maxURLLength {
		link.Text = internal.GetTextContent(n)
		link.IsExternal = internal.IsExternalURL(link.URL)
		return link
	}

	return LinkInfo{}
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
	videos := make([]VideoInfo, 0, 8)
	seen := make(map[string]bool, 8)

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
		matches := p.videoRegex.FindAllString(htmlContent, maxRegexMatches)
		for _, url := range matches {
			if !seen[url] && len(url) <= maxURLLength {
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
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "source" {
				for _, attr := range c.Attr {
					if attr.Key == "src" && video.URL == "" {
						video.URL = attr.Val
					} else if attr.Key == "type" && video.Type == "" {
						video.Type = attr.Val
					}
				}
				if video.URL != "" {
					break
				}
			}
		}
	}

	if video.URL != "" && len(video.URL) <= maxURLLength {
		return video
	}
	return VideoInfo{}
}

func (p *Processor) parseIframeNode(n *html.Node) VideoInfo {
	for _, attr := range n.Attr {
		if attr.Key == "src" && len(attr.Val) <= maxURLLength && internal.IsVideoEmbedURL(attr.Val) {
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
		if (attr.Key == "src" || attr.Key == "data") && len(attr.Val) <= maxURLLength && internal.IsVideoURL(attr.Val) {
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
	audios := make([]AudioInfo, 0, 8)
	seen := make(map[string]bool, 8)

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
		matches := p.audioRegex.FindAllString(htmlContent, maxRegexMatches)
		for _, url := range matches {
			if !seen[url] && len(url) <= maxURLLength {
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
			audio.URL = attr.Val
		case "duration":
			audio.Duration = attr.Val
		}
	}

	if audio.URL == "" {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "source" {
				for _, attr := range c.Attr {
					if attr.Key == "src" && audio.URL == "" {
						audio.URL = attr.Val
					} else if attr.Key == "type" && audio.Type == "" {
						audio.Type = attr.Val
					}
				}
				if audio.URL != "" {
					break
				}
			}
		}
	}

	if audio.URL != "" && len(audio.URL) <= maxURLLength {
		return audio
	}
	return AudioInfo{}
}

func (p *Processor) generateCacheKey(content string, opts ExtractConfig) string {
	h := sha256.New()

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
	if opts.InlineImageFormat != "" {
		h.Write([]byte(opts.InlineImageFormat))
		h.Write([]byte{0}) // separator
	}
	h.Write([]byte(content))

	return hex.EncodeToString(h.Sum(nil))
}
