package html

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	htmlstd "html"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cybergodev/html/internal"
	stdxhtml "golang.org/x/net/html"
)

// Re-exports of commonly used types and constants from golang.org/x/net/html
type (
	Node        = stdxhtml.Node
	NodeType    = stdxhtml.NodeType
	Token       = stdxhtml.Token
	Attribute   = stdxhtml.Attribute
	Tokenizer   = stdxhtml.Tokenizer
	ParseOption = stdxhtml.ParseOption
)

// NodeType constants - all node types from golang.org/x/net/html
const (
	ErrorNode    = stdxhtml.ErrorNode
	TextNode     = stdxhtml.TextNode
	DocumentNode = stdxhtml.DocumentNode
	ElementNode  = stdxhtml.ElementNode
	CommentNode  = stdxhtml.CommentNode
	DoctypeNode  = stdxhtml.DoctypeNode
	RawNode      = stdxhtml.RawNode
)

// TokenType constants - all token types from golang.org/x/net/html
const (
	ErrorToken          = stdxhtml.ErrorToken
	TextToken           = stdxhtml.TextToken
	StartTagToken       = stdxhtml.StartTagToken
	EndTagToken         = stdxhtml.EndTagToken
	SelfClosingTagToken = stdxhtml.SelfClosingTagToken
	CommentToken        = stdxhtml.CommentToken
	DoctypeToken        = stdxhtml.DoctypeToken
)

var (
	// Errors
	ErrBufferExceeded = stdxhtml.ErrBufferExceeded

	// Parsing functions
	Parse                    = stdxhtml.Parse
	ParseFragment            = stdxhtml.ParseFragment
	ParseWithOptions         = stdxhtml.ParseWithOptions
	ParseFragmentWithOptions = stdxhtml.ParseFragmentWithOptions

	// Rendering functions
	Render         = stdxhtml.Render
	EscapeString   = htmlstd.EscapeString
	UnescapeString = stdxhtml.UnescapeString

	// Tokenizer functions
	NewTokenizer         = stdxhtml.NewTokenizer
	NewTokenizerFragment = stdxhtml.NewTokenizerFragment

	// Parse options
	ParseOptionEnableScripting = stdxhtml.ParseOptionEnableScripting
)

func Extract(htmlContent string, configs ...ExtractConfig) (*Result, error) {
	processor := NewWithDefaults()
	defer processor.Close()
	return processor.Extract(htmlContent, configs...)
}

func ExtractFromFile(filePath string, configs ...ExtractConfig) (*Result, error) {
	processor := NewWithDefaults()
	defer processor.Close()
	return processor.ExtractFromFile(filePath, configs...)
}

// ExtractText extracts only text content without metadata.
func ExtractText(htmlContent string) (string, error) {
	result, err := Extract(htmlContent)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

func ExtractAllLinks(htmlContent string, configs ...LinkExtractionConfig) ([]LinkResource, error) {
	processor := NewWithDefaults()
	defer processor.Close()
	return processor.ExtractAllLinks(htmlContent, configs...)
}

// GroupLinksByType groups LinkResource slice by their Type field.
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
	DefaultMaxCacheEntries   = 1000
	DefaultWorkerPoolSize    = 4
	DefaultCacheTTL          = time.Hour
	DefaultMaxDepth          = 100
	DefaultProcessingTimeout = 30 * time.Second

	maxURLLength        = 2000
	maxHTMLForRegex     = 1000000
	maxRegexMatches     = 100
	wordsPerMinute      = 200
	maxCacheKeySize     = 64 * 1024
	cacheKeySample      = 4096
	initialTextSize     = 4096
	initialSliceCap     = 16
	initialMapCap       = 8
	maxConfigInputSize  = 50 * 1024 * 1024
	maxConfigWorkerSize = 256
	maxConfigDepth      = 500
	maxDataURILength    = 100000
)

var (
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

func New(config Config) (*Processor, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	return &Processor{
		config: &config,
		cache:  internal.NewCache(config.MaxCacheEntries, config.CacheTTL),
	}, nil
}

func NewWithDefaults() *Processor {
	p, err := New(DefaultConfig())
	if err != nil {
		panic(err)
	}
	return p
}

func (p *Processor) Extract(htmlContent string, configs ...ExtractConfig) (*Result, error) {
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	config := resolveExtractConfig(configs...)

	if len(htmlContent) > p.config.MaxInputSize {
		p.stats.errorCount.Add(1)
		return nil, fmt.Errorf("%w: size=%d, max=%d", ErrInputTooLarge, len(htmlContent), p.config.MaxInputSize)
	}

	startTime := time.Now()

	cacheKey := p.generateCacheKey(htmlContent, config)
	if cached := p.cache.Get(cacheKey); cached != nil {
		p.stats.cacheHits.Add(1)
		p.stats.totalProcessed.Add(1)
		if result, ok := cached.(*Result); ok {
			return result, nil
		}
	}
	p.stats.cacheMisses.Add(1)

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

func (p *Processor) processWithTimeout(htmlContent string, config ExtractConfig) (*Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.config.ProcessingTimeout)
	defer cancel()

	type processResult struct {
		result *Result
		err    error
	}

	resultChan := make(chan processResult, 1)
	go func() {
		result, err := p.processContent(htmlContent, config)
		select {
		case resultChan <- processResult{result: result, err: err}:
		case <-ctx.Done():
		}
	}()

	select {
	case res := <-resultChan:
		return res.result, res.err
	case <-ctx.Done():
		return nil, ErrProcessingTimeout
	}
}

func (p *Processor) ExtractWithDefaults(htmlContent string) (*Result, error) {
	return p.Extract(htmlContent, DefaultExtractConfig())
}

func (p *Processor) ExtractFromFile(filePath string, configs ...ExtractConfig) (*Result, error) {
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}
	if filePath == "" {
		return nil, fmt.Errorf("%w: empty file path", ErrFileNotFound)
	}

	config := resolveExtractConfig(configs...)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, filePath)
		}
		return nil, fmt.Errorf("read file %q: %w", filePath, err)
	}

	return p.Extract(string(data), config)
}

func (p *Processor) ExtractBatch(htmlContents []string, configs ...ExtractConfig) ([]*Result, error) {
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

func (p *Processor) ExtractBatchFiles(filePaths []string, configs ...ExtractConfig) ([]*Result, error) {
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

func (p *Processor) ExtractAllLinks(htmlContent string, configs ...LinkExtractionConfig) ([]LinkResource, error) {
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	config := resolveLinkExtractionConfig(configs...)

	if len(htmlContent) > p.config.MaxInputSize {
		p.stats.errorCount.Add(1)
		return nil, fmt.Errorf("%w: size=%d, max=%d", ErrInputTooLarge, len(htmlContent), p.config.MaxInputSize)
	}

	startTime := time.Now()

	// Process with timeout if configured
	var links []LinkResource
	var err error
	if p.config.ProcessingTimeout > 0 {
		links, err = p.extractLinksWithTimeout(htmlContent, config)
	} else {
		links, err = p.extractAllLinksFromContent(htmlContent, config)
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
	ctx, cancel := context.WithTimeout(context.Background(), p.config.ProcessingTimeout)
	defer cancel()

	type linkResult struct {
		links []LinkResource
		err   error
	}

	resultChan := make(chan linkResult, 1)
	go func() {
		links, err := p.extractAllLinksFromContent(htmlContent, config)
		select {
		case resultChan <- linkResult{links: links, err: err}:
		case <-ctx.Done():
		}
	}()

	select {
	case res := <-resultChan:
		return res.links, res.err
	case <-ctx.Done():
		return nil, ErrProcessingTimeout
	}
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

func (p *Processor) ClearCache() {
	p.cache.Clear()
	p.stats.cacheHits.Store(0)
	p.stats.cacheMisses.Store(0)
}

func (p *Processor) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	p.cache.Clear()
	return nil
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
		return nil, fmt.Errorf("%w: %v", ErrInvalidHTML, err)
	}

	if err := p.validateDepth(doc, 0); err != nil {
		return nil, err
	}

	return p.extractFromDocument(doc, originalHTML, opts)
}

func (p *Processor) validateDepth(n *Node, depth int) error {
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
			htmlImg.Grow(len(images[i].URL) + len(images[i].Alt) + len(images[i].Width) + len(images[i].Height) + 64)
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

func (p *Processor) extractVideos(node *Node, htmlContent string) []VideoInfo {
	videos := make([]VideoInfo, 0, initialSliceCap)
	seen := make(map[string]bool, initialMapCap)

	// First, extract from the HTML content directly for iframe/embed/object tags
	// These may be removed by sanitization, so we parse them from raw HTML first
	if len(htmlContent) > 0 && len(htmlContent) <= maxHTMLForRegex*10 {
		// Parse iframe tags
		iframeMatches := p.extractTagAttributes(htmlContent, "iframe", "src")
		for _, url := range iframeMatches {
			if isValidURL(url) && internal.IsVideoURL(url) && !seen[url] {
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
			if isValidURL(url) && internal.IsVideoURL(url) && !seen[url] {
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
			if isValidURL(url) && internal.IsVideoURL(url) && !seen[url] {
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

func (p *Processor) parseVideoNode(n *Node) VideoInfo {
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

func (p *Processor) parseIframeNode(n *Node) VideoInfo {
	for _, attr := range n.Attr {
		if attr.Key == "src" && isValidURL(attr.Val) && internal.IsVideoURL(attr.Val) {
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

func (p *Processor) extractTagAttributes(htmlContent, tagName string, attrNames ...string) []string {
	results := make([]string, 0, 16)
	lowerHTML := strings.ToLower(htmlContent)
	lowerTag := "<" + tagName

	pos := 0
	for {
		tagStart := strings.Index(lowerHTML[pos:], lowerTag)
		if tagStart == -1 {
			break
		}
		tagStart += pos

		tagEnd := strings.IndexByte(htmlContent[tagStart:], '>')
		if tagEnd == -1 {
			break
		}
		tagEnd += tagStart + 1

		tagContent := htmlContent[tagStart:tagEnd]
		lowerTagContent := lowerHTML[tagStart:tagEnd]

		for _, attrName := range attrNames {
			lowerAttrName := strings.ToLower(attrName)
			attrPattern := lowerAttrName + "="
			attrPos := 0

			for attrPos < len(tagContent) {
				attrIndex := strings.Index(lowerTagContent[attrPos:], attrPattern)
				if attrIndex == -1 {
					break
				}
				attrIndex += attrPos

				valueStart := attrIndex + len(attrName) + 1
				for valueStart < len(tagContent) && (tagContent[valueStart] == ' ' || tagContent[valueStart] == '\t') {
					valueStart++
				}

				if valueStart >= len(tagContent) {
					break
				}

				var quote byte
				if tagContent[valueStart] == '"' || tagContent[valueStart] == '\'' {
					quote = tagContent[valueStart]
					valueStart++
				}

				var valueEnd int
				if quote != 0 {
					valueEnd = strings.IndexByte(tagContent[valueStart:], quote)
					if valueEnd == -1 {
						break
					}
					valueEnd += valueStart
				} else {
					for valueEnd = valueStart; valueEnd < len(tagContent); valueEnd++ {
						c := tagContent[valueEnd]
						if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '>' {
							break
						}
					}
				}

				if valueStart < valueEnd {
					results = append(results, tagContent[valueStart:valueEnd])
				}

				attrPos = valueEnd
			}
		}

		pos = tagEnd
	}

	return results
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

func (p *Processor) parseAudioNode(n *Node) AudioInfo {
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

// isValidURL checks if a URL is valid and safe for processing.
func isValidURL(url string) bool {
	urlLen := len(url)
	if urlLen == 0 || urlLen > maxURLLength {
		return false
	}

	// Special handling for data URLs - stricter validation with size limit
	if strings.HasPrefix(url, "data:") {
		if urlLen > maxDataURILength {
			return false
		}
		for i := 5; i < urlLen; i++ {
			b := url[i]
			if b < 32 || b > 126 || b == '<' || b == '>' || b == '"' || b == '\'' || b == '\\' {
				return false
			}
		}
		return true
	}

	// Validate non-data URLs: check for dangerous characters
	for i := 0; i < urlLen; i++ {
		b := url[i]
		if b < 32 || b == 127 || b == '<' || b == '>' || b == '"' || b == '\'' {
			return false
		}
	}

	// Accept absolute and protocol-relative URLs
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		return true
	}
	if strings.HasPrefix(url, "//") {
		return true
	}

	// Accept relative URLs and paths (starting with / or .)
	if url[0] == '/' || url[0] == '.' {
		return true
	}

	// Accept alphanumeric paths (legitimate filenames like img1.jpg, video.mp4)
	// but reject paths starting with special characters that might be used in injection attacks
	if urlLen > 0 {
		firstChar := url[0]
		if (firstChar >= 'a' && firstChar <= 'z') ||
			(firstChar >= 'A' && firstChar <= 'Z') ||
			(firstChar >= '0' && firstChar <= '9') {
			return true
		}
	}

	return false
}

// generateCacheKey creates a SHA-256 hash for cache key generation.
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
		h.Write([]byte(content[:cacheKeySample]))
		h.Write([]byte(content[contentLen-cacheKeySample:]))
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
		return nil, fmt.Errorf("%w: %v", ErrInvalidHTML, err)
	}

	if err := p.validateDepth(doc, 0); err != nil {
		return nil, err
	}

	baseURL := config.BaseURL
	if config.ResolveRelativeURLs && baseURL == "" {
		baseURL = p.detectBaseURL(doc)
	}

	linkMap := make(map[string]LinkResource, 64)
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
					if (attr.Key == "href" || attr.Key == "src") && p.isAbsoluteURL(attr.Val) {
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
	if baseURL == "" {
		return ""
	}

	// Skip non-HTTP URLs like data:, javascript:, mailto:, etc.
	if strings.Contains(baseURL, ":") && !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		return ""
	}

	lastSlash := strings.LastIndexByte(baseURL, '/')
	if lastSlash < 0 {
		return baseURL + "/"
	}

	if lastSlash < len(baseURL)-1 {
		return baseURL[:lastSlash+1]
	}

	return baseURL
}

func (p *Processor) isAbsoluteURL(url string) bool {
	return strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "//")
}

func (p *Processor) extractBaseFromURL(url string) string {
	if !p.isAbsoluteURL(url) {
		return ""
	}

	start := 0
	if idx := strings.Index(url, "://"); idx >= 0 {
		start = idx + 3
	} else if strings.HasPrefix(url, "//") {
		start = 2
	}

	if pathStart := strings.IndexByte(url[start:], '/'); pathStart >= 0 {
		return url[:start+pathStart+1]
	}

	return url + "/"
}

func (p *Processor) isDifferentDomain(baseURL, targetURL string) bool {
	if !p.isAbsoluteURL(baseURL) || !p.isAbsoluteURL(targetURL) {
		return false
	}

	baseDomain := p.extractDomain(baseURL)
	targetDomain := p.extractDomain(targetURL)

	return baseDomain != targetDomain
}

func (p *Processor) extractDomain(url string) string {
	start := 0
	if idx := strings.Index(url, "://"); idx >= 0 {
		start = idx + 3
	} else if strings.HasPrefix(url, "//") {
		start = 2
	}

	if pathStart := strings.IndexByte(url[start:], '/'); pathStart >= 0 {
		return url[start : start+pathStart]
	}

	return url[start:]
}

func (p *Processor) resolveURL(baseURL, relativeURL string) string {
	if relativeURL == "" || baseURL == "" {
		return relativeURL
	}

	if p.isAbsoluteURL(relativeURL) {
		return relativeURL
	}

	if len(relativeURL) >= 2 && relativeURL[0] == '/' && relativeURL[1] == '/' {
		if strings.HasPrefix(baseURL, "https:") {
			return "https:" + relativeURL
		}
		return "http:" + relativeURL
	}

	if relativeURL[0] == '/' {
		if idx := strings.Index(baseURL, "://"); idx >= 0 {
			if domainEnd := strings.IndexByte(baseURL[idx+3:], '/'); domainEnd >= 0 {
				return baseURL[:idx+3+domainEnd] + relativeURL
			}
			return baseURL + relativeURL
		}
		return relativeURL
	}

	return baseURL + relativeURL
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

	if href == "" || !isValidURL(href) {
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

	if src == "" || !isValidURL(src) {
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

	if src == "" || !isValidURL(src) {
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

	if src == "" || !isValidURL(src) {
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

	if href == "" || !isValidURL(href) {
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

	if src == "" || !isValidURL(src) {
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

	if src == "" || !isValidURL(src) {
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

func ExtractToMarkdown(htmlContent string) (string, error) {
	processor := NewWithDefaults()
	defer processor.Close()

	config := DefaultExtractConfig()
	config.InlineImageFormat = "markdown"
	result, err := processor.Extract(htmlContent, config)
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

func ExtractToJSON(htmlContent string) ([]byte, error) {
	result, err := Extract(htmlContent)
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}
