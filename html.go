package html

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	stdhtml "html"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cybergodev/html/internal"
	"golang.org/x/net/html"
)

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

var (
	Parse          = html.Parse
	ParseFragment  = html.ParseFragment
	Render         = html.Render
	EscapeString   = html.EscapeString
	UnescapeString = html.UnescapeString
	NewTokenizer   = html.NewTokenizer
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

	maxURLLength    = 2000
	maxHTMLForRegex = 1000000
	maxRegexMatches = 100
	wordsPerMinute  = 200
	maxCacheKeySize = 64 * 1024
	cacheKeySample  = 4096
	initialTextSize = 4096
	initialSliceCap = 16
	initialMapCap   = 8
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
	case c.MaxInputSize > 50*1024*1024: // 50MB limit
		return fmt.Errorf("%w: MaxInputSize too large (max 50MB), got %d", ErrInvalidConfig, c.MaxInputSize)
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

	return nil
}

type ExtractConfig struct {
	ExtractArticle    bool
	PreserveImages    bool
	PreserveLinks     bool
	PreserveVideos    bool
	PreserveAudios    bool
	InlineImageFormat string
}

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

type ImageInfo struct {
	URL          string
	Alt          string
	Title        string
	Width        string
	Height       string
	IsDecorative bool
	Position     int
}

type LinkInfo struct {
	URL        string
	Text       string
	Title      string
	IsExternal bool
	IsNoFollow bool
}

type VideoInfo struct {
	URL      string
	Type     string
	Poster   string
	Width    string
	Height   string
	Duration string
}

type AudioInfo struct {
	URL      string
	Type     string
	Duration string
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
		return configs[0]
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
	p, _ := New(DefaultConfig())
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

	if format != "none" {
		images := p.extractImagesWithPosition(contentNode)

		if opts.PreserveImages {
			result.Images = images
		}

		var sb strings.Builder
		sb.Grow(initialTextSize)
		imageCounter := 0
		internal.ExtractTextWithStructureAndImages(contentNode, &sb, 0, &imageCounter)
		textWithPlaceholders := internal.CleanText(sb.String(), nil)
		result.Text = p.formatInlineImages(textWithPlaceholders, images, format)
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

func (p *Processor) extractTextContent(node *html.Node) string {
	var sb strings.Builder
	sb.Grow(initialTextSize)
	internal.ExtractTextWithStructureAndImages(node, &sb, 0, nil)
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
			htmlImg.WriteString(stdhtml.EscapeString(images[i].URL))
			htmlImg.WriteString(`" alt="`)
			htmlImg.WriteString(stdhtml.EscapeString(images[i].Alt))
			htmlImg.WriteString(`"`)
			if images[i].Width != "" {
				htmlImg.WriteString(` width="`)
				htmlImg.WriteString(stdhtml.EscapeString(images[i].Width))
				htmlImg.WriteString(`"`)
			}
			if images[i].Height != "" {
				htmlImg.WriteString(` height="`)
				htmlImg.WriteString(stdhtml.EscapeString(images[i].Height))
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
	images := make([]ImageInfo, 0, initialSliceCap)

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
	images := make([]ImageInfo, 0, initialSliceCap)
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
	links := make([]LinkInfo, 0, initialSliceCap)

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
	videos := make([]VideoInfo, 0, initialSliceCap)
	seen := make(map[string]bool, initialMapCap)

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
	audios := make([]AudioInfo, 0, initialSliceCap)
	seen := make(map[string]bool, initialMapCap)

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

// isValidURL checks if a URL is valid and safe for processing.
func isValidURL(url string) bool {
	urlLen := len(url)
	if urlLen == 0 || urlLen > maxURLLength {
		return false
	}

	// Special handling for data URLs - stricter validation
	if strings.HasPrefix(url, "data:") {
		// data: URLs should only contain safe printable ASCII characters
		for i := 5; i < urlLen; i++ {
			b := url[i]
			// Allow only printable ASCII (32-126), excluding dangerous chars
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

	// Accept relative URLs and paths
	firstChar := url[0]
	if firstChar == '/' || firstChar == '.' {
		return true
	}

	// Accept alphanumeric paths
	return (firstChar >= 'a' && firstChar <= 'z') ||
		(firstChar >= 'A' && firstChar <= 'Z') ||
		(firstChar >= '0' && firstChar <= '9')
}

// generateCacheKey creates a SHA-256 hash for cache key generation.
func (p *Processor) generateCacheKey(content string, opts ExtractConfig) string {
	h := sha256.New()

	// Encode configuration into hash
	configKey := strconv.FormatBool(opts.ExtractArticle) + "," +
		strconv.FormatBool(opts.PreserveImages) + "," +
		strconv.FormatBool(opts.PreserveLinks) + "," +
		strconv.FormatBool(opts.PreserveVideos) + "," +
		strconv.FormatBool(opts.PreserveAudios) + "," +
		opts.InlineImageFormat
	h.Write([]byte(configKey))
	h.Write([]byte{0}) // separator

	contentLen := len(content)
	if contentLen <= maxCacheKeySize {
		// For small content, hash the entire content
		h.Write([]byte(content))
	} else {
		// For large content, hash beginning and end with length
		// This provides good cache distribution while avoiding memory pressure
		h.Write([]byte(content[:cacheKeySample]))
		h.Write([]byte(content[contentLen-cacheKeySample:]))
		h.Write([]byte(strconv.Itoa(contentLen)))
	}

	var buf [32]byte
	sum := h.Sum(buf[:0])
	return hex.EncodeToString(sum)
}

// extractAllLinksFromContent performs comprehensive link extraction from HTML content.
func (p *Processor) extractAllLinksFromContent(htmlContent string, config LinkExtractionConfig) ([]LinkResource, error) {
	// Validate input is not empty
	if strings.TrimSpace(htmlContent) == "" {
		return []LinkResource{}, nil
	}

	// Parse HTML document BEFORE sanitization to preserve all links
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidHTML, err)
	}

	// Validate document depth to prevent DoS
	if err := p.validateDepth(doc, 0); err != nil {
		return nil, err
	}

	// Detect base URL for relative link resolution
	baseURL := config.BaseURL
	if config.ResolveRelativeURLs && baseURL == "" {
		baseURL = p.detectBaseURL(doc)
	}

	// Extract all links with deduplication
	linkMap := make(map[string]LinkResource, 64) // Use map for deduplication
	p.extractLinksFromDocument(doc, baseURL, config, linkMap)

	// Convert map to slice
	links := make([]LinkResource, 0, len(linkMap))
	for _, link := range linkMap {
		links = append(links, link)
	}

	return links, nil
}

// detectBaseURL attempts to detect base URL from HTML document.
func (p *Processor) detectBaseURL(doc *html.Node) string {
	if baseNode := internal.FindElementByTag(doc, "base"); baseNode != nil {
		for _, attr := range baseNode.Attr {
			if attr.Key == "href" && attr.Val != "" {
				return p.normalizeBaseURL(attr.Val)
			}
		}
	}

	var canonicalURL, canonicalLink, firstAbsoluteURL string
	internal.WalkNodes(doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
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

// normalizeBaseURL normalizes base URL for consistent resolution.
func (p *Processor) normalizeBaseURL(baseURL string) string {
	if baseURL == "" {
		return ""
	}

	// Find last slash position
	lastSlash := strings.LastIndexByte(baseURL, '/')
	if lastSlash < 0 {
		return baseURL + "/"
	}

	// If there's content after last slash, it's a file - return directory
	if lastSlash < len(baseURL)-1 {
		return baseURL[:lastSlash+1]
	}

	// Already ends with slash
	return baseURL
}

// isAbsoluteURL checks if URL is absolute.
func (p *Processor) isAbsoluteURL(url string) bool {
	return strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "//")
}

// extractBaseFromURL extracts base URL from absolute URL.
func (p *Processor) extractBaseFromURL(url string) string {
	if !p.isAbsoluteURL(url) {
		return ""
	}

	// Find protocol end
	start := 0
	if idx := strings.Index(url, "://"); idx >= 0 {
		start = idx + 3
	} else if strings.HasPrefix(url, "//") {
		start = 2
	}

	// Find path start after domain
	if pathStart := strings.IndexByte(url[start:], '/'); pathStart >= 0 {
		return url[:start+pathStart+1]
	}

	return url + "/"
}

// isDifferentDomain checks if two URLs have different domains.
func (p *Processor) isDifferentDomain(baseURL, targetURL string) bool {
	if !p.isAbsoluteURL(baseURL) || !p.isAbsoluteURL(targetURL) {
		return false
	}

	baseDomain := p.extractDomain(baseURL)
	targetDomain := p.extractDomain(targetURL)

	return baseDomain != targetDomain
}

// extractDomain extracts domain from URL.
func (p *Processor) extractDomain(url string) string {
	// Remove protocol
	start := 0
	if idx := strings.Index(url, "://"); idx >= 0 {
		start = idx + 3
	} else if strings.HasPrefix(url, "//") {
		start = 2
	}

	// Find first slash after domain
	if pathStart := strings.IndexByte(url[start:], '/'); pathStart >= 0 {
		return url[start : start+pathStart]
	}

	return url[start:]
}

// resolveURL resolves relative URL against base URL.
func (p *Processor) resolveURL(baseURL, relativeURL string) string {
	if relativeURL == "" || baseURL == "" {
		return relativeURL
	}

	// Already absolute
	if p.isAbsoluteURL(relativeURL) {
		return relativeURL
	}

	// Protocol-relative URL (//example.com/path)
	if len(relativeURL) >= 2 && relativeURL[0] == '/' && relativeURL[1] == '/' {
		if strings.HasPrefix(baseURL, "https:") {
			return "https:" + relativeURL
		}
		return "http:" + relativeURL
	}

	// Root-relative URL (/path)
	if relativeURL[0] == '/' {
		// Extract protocol and domain from base URL
		if idx := strings.Index(baseURL, "://"); idx >= 0 {
			if domainEnd := strings.IndexByte(baseURL[idx+3:], '/'); domainEnd >= 0 {
				return baseURL[:idx+3+domainEnd] + relativeURL
			}
			return baseURL + relativeURL
		}
		return relativeURL
	}

	// Relative URL - append to base
	return baseURL + relativeURL
}

// extractLinksFromDocument extracts all types of links from HTML document.
func (p *Processor) extractLinksFromDocument(doc *html.Node, baseURL string, config LinkExtractionConfig, linkMap map[string]LinkResource) {
	internal.WalkNodes(doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
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

// extractContentLinks extracts content navigation links from <a> tags.
func (p *Processor) extractContentLinks(n *html.Node, baseURL string, config LinkExtractionConfig, linkMap map[string]LinkResource) {
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

	// Determine if external BEFORE URL resolution
	isExternalOriginal := internal.IsExternalURL(href)

	// Resolve relative URL
	resolvedURL := href
	if config.ResolveRelativeURLs && baseURL != "" {
		resolvedURL = p.resolveURL(baseURL, href)
	}

	// For classification, use original URL to determine if it's truly external
	// If original URL was relative or root-relative, it's internal
	isExternal := isExternalOriginal
	if !isExternalOriginal && baseURL != "" {
		// Check if resolved URL points to different domain than base
		isExternal = p.isDifferentDomain(baseURL, resolvedURL)
	}

	// Filter based on configuration
	if isExternal && !config.IncludeExternalLinks {
		return
	}
	if !isExternal && !config.IncludeContentLinks {
		return
	}

	// Get link text if no title
	if title == "" {
		title = strings.TrimSpace(internal.GetTextContent(n))
		if title == "" {
			title = "Link"
		}
	}

	linkType := "link"
	// All content links (a tags) are classified as "link" type
	// regardless of whether they are internal or external

	linkMap[resolvedURL] = LinkResource{
		URL:   resolvedURL,
		Title: title,
		Type:  linkType,
	}
}

// extractImageLinks extracts image resource links.
func (p *Processor) extractImageLinks(n *html.Node, baseURL string, linkMap map[string]LinkResource) {
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

	// Resolve relative URL
	resolvedURL := src
	if baseURL != "" {
		resolvedURL = p.resolveURL(baseURL, src)
	}

	// Use alt or title as resource name
	resourceName := title
	if resourceName == "" {
		resourceName = alt
	}
	if resourceName == "" {
		// Extract filename from URL
		if lastSlash := strings.LastIndex(resolvedURL, "/"); lastSlash >= 0 {
			resourceName = resolvedURL[lastSlash+1:]
		} else {
			resourceName = "Image"
		}
	}

	linkMap[resolvedURL] = LinkResource{
		URL:   resolvedURL,
		Title: resourceName,
		Type:  "image",
	}
}

// extractMediaLink extracts video or audio resource links.
func (p *Processor) extractMediaLink(n *html.Node, baseURL string, linkMap map[string]LinkResource, mediaType string) {
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

	if title == "" {
		if lastSlash := strings.LastIndex(resolvedURL, "/"); lastSlash >= 0 {
			title = resolvedURL[lastSlash+1:]
		} else {
			title = strings.ToUpper(mediaType[:1]) + mediaType[1:]
		}
	}

	linkMap[resolvedURL] = LinkResource{
		URL:   resolvedURL,
		Title: title,
		Type:  mediaType,
	}
}

// extractSourceLinks extracts source links from <source> tags.
func (p *Processor) extractSourceLinks(n *html.Node, baseURL string, linkMap map[string]LinkResource) {
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

	// Determine resource type from MIME type or URL
	resourceType := "media"
	if strings.HasPrefix(mediaType, "video/") {
		resourceType = "video"
	} else if strings.HasPrefix(mediaType, "audio/") {
		resourceType = "audio"
	} else {
		// Detect from URL extension
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

// extractLinkTagLinks extracts links from <link> tags (CSS, icons, etc.).
func (p *Processor) extractLinkTagLinks(n *html.Node, baseURL string, config LinkExtractionConfig, linkMap map[string]LinkResource) {
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

	// Determine resource type from rel attribute
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
		// Determine type from 'as' attribute or MIME type
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
		// Check MIME type for other link types
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
			// Capitalize first letter of resource type
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

// extractScriptLinks extracts JavaScript resource links.
func (p *Processor) extractScriptLinks(n *html.Node, baseURL string, linkMap map[string]LinkResource) {
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

	if title == "" {
		if lastSlash := strings.LastIndex(resolvedURL, "/"); lastSlash >= 0 {
			title = resolvedURL[lastSlash+1:]
		} else {
			title = "Script"
		}
	}

	linkMap[resolvedURL] = LinkResource{
		URL:   resolvedURL,
		Title: title,
		Type:  "js",
	}
}

// extractEmbedLinks extracts embedded video links from iframe, embed, object tags.
func (p *Processor) extractEmbedLinks(n *html.Node, baseURL string, linkMap map[string]LinkResource) {
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

// ============================================================================
// Package-Level Convenience Functions
// ============================================================================

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

var jsonBuilderPool = sync.Pool{
	New: func() any {
		sb := &strings.Builder{}
		sb.Grow(4096) // Pre-allocate for typical JSON size
		return sb
	},
}

func ExtractToJSON(htmlContent string) ([]byte, error) {
	result, err := Extract(htmlContent)
	if err != nil {
		return nil, err
	}

	// Acquire builder from pool
	buf := jsonBuilderPool.Get().(*strings.Builder)
	defer func() {
		buf.Reset()
		jsonBuilderPool.Put(buf)
	}()

	// Estimate size and pre-allocate
	estimatedSize := len(result.Text) + len(result.Title) + 512
	for _, img := range result.Images {
		estimatedSize += len(img.URL) + len(img.Alt) + 100
	}
	for _, link := range result.Links {
		estimatedSize += len(link.URL) + len(link.Text) + 100
	}
	buf.Grow(estimatedSize)

	buf.WriteString(`{"title":`)
	writeJSONStringFast(buf, result.Title)
	buf.WriteString(`,"text":`)
	writeJSONStringFast(buf, result.Text)
	buf.WriteString(`,"word_count":`)
	buf.WriteString(strconv.Itoa(result.WordCount))
	buf.WriteString(`,"reading_time_ms":`)
	buf.WriteString(strconv.FormatInt(result.ReadingTime.Milliseconds(), 10))
	buf.WriteString(`,"processing_time_ms":`)
	buf.WriteString(strconv.FormatInt(result.ProcessingTime.Milliseconds(), 10))

	if len(result.Images) > 0 {
		buf.WriteString(`,"images":[`)
		writeImagesJSON(buf, result.Images)
		buf.WriteString("]")
	}

	if len(result.Links) > 0 {
		buf.WriteString(`,"links":[`)
		writeLinksJSON(buf, result.Links)
		buf.WriteString("]")
	}

	if len(result.Videos) > 0 {
		buf.WriteString(`,"videos":[`)
		writeVideosJSON(buf, result.Videos)
		buf.WriteString("]")
	}

	if len(result.Audios) > 0 {
		buf.WriteString(`,"audios":[`)
		writeAudiosJSON(buf, result.Audios)
		buf.WriteString("]")
	}

	buf.WriteString("}")

	return []byte(buf.String()), nil
}

func writeJSONStringFast(sb *strings.Builder, s string) {
	sb.WriteByte('"')
	// Use strings.Builder's internal buffer more efficiently
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"', '\\', '\b', '\f', '\n', '\r', '\t':
			if start < i {
				sb.WriteString(s[start:i])
			}
			switch c {
			case '"':
				sb.WriteString(`\"`)
			case '\\':
				sb.WriteString(`\\`)
			case '\b':
				sb.WriteString(`\b`)
			case '\f':
				sb.WriteString(`\f`)
			case '\n':
				sb.WriteString(`\n`)
			case '\r':
				sb.WriteString(`\r`)
			case '\t':
				sb.WriteString(`\t`)
			}
			start = i + 1
		default:
			if c < 32 {
				if start < i {
					sb.WriteString(s[start:i])
				}
				sb.WriteString(`\u00`)
				sb.WriteString(strconv.FormatInt(int64(c), 16))
				start = i + 1
			}
		}
	}
	if start < len(s) {
		sb.WriteString(s[start:])
	}
	sb.WriteByte('"')
}

func writeImagesJSON(sb *strings.Builder, images []ImageInfo) {
	for i, img := range images {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(`{"url":"`)
		sb.WriteString(stdhtml.EscapeString(img.URL))
		sb.WriteString(`","alt":"`)
		sb.WriteString(stdhtml.EscapeString(img.Alt))
		sb.WriteString(`","title":"`)
		sb.WriteString(stdhtml.EscapeString(img.Title))
		sb.WriteString(`","width":"`)
		sb.WriteString(stdhtml.EscapeString(img.Width))
		sb.WriteString(`","height":"`)
		sb.WriteString(stdhtml.EscapeString(img.Height))
		sb.WriteString(`","is_decorative":`)
		sb.WriteString(strconv.FormatBool(img.IsDecorative))
		sb.WriteString(`,"position":`)
		sb.WriteString(strconv.Itoa(img.Position))
		sb.WriteByte('}')
	}
}

func writeLinksJSON(sb *strings.Builder, links []LinkInfo) {
	for i, link := range links {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(`{"url":"`)
		sb.WriteString(stdhtml.EscapeString(link.URL))
		sb.WriteString(`","text":"`)
		sb.WriteString(stdhtml.EscapeString(link.Text))
		sb.WriteString(`","title":"`)
		sb.WriteString(stdhtml.EscapeString(link.Title))
		sb.WriteString(`","is_external":`)
		sb.WriteString(strconv.FormatBool(link.IsExternal))
		sb.WriteString(`,"is_nofollow":`)
		sb.WriteString(strconv.FormatBool(link.IsNoFollow))
		sb.WriteByte('}')
	}
}

func writeVideosJSON(sb *strings.Builder, videos []VideoInfo) {
	for i, vid := range videos {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(`{"url":"`)
		sb.WriteString(stdhtml.EscapeString(vid.URL))
		sb.WriteString(`","type":"`)
		sb.WriteString(stdhtml.EscapeString(vid.Type))
		sb.WriteString(`","poster":"`)
		sb.WriteString(stdhtml.EscapeString(vid.Poster))
		sb.WriteString(`","width":"`)
		sb.WriteString(stdhtml.EscapeString(vid.Width))
		sb.WriteString(`","height":"`)
		sb.WriteString(stdhtml.EscapeString(vid.Height))
		sb.WriteString(`","duration":"`)
		sb.WriteString(stdhtml.EscapeString(vid.Duration))
		sb.WriteString(`"}`)
	}
}

func writeAudiosJSON(sb *strings.Builder, audios []AudioInfo) {
	for i, aud := range audios {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(`{"url":"`)
		sb.WriteString(stdhtml.EscapeString(aud.URL))
		sb.WriteString(`","type":"`)
		sb.WriteString(stdhtml.EscapeString(aud.Type))
		sb.WriteString(`","duration":"`)
		sb.WriteString(stdhtml.EscapeString(aud.Duration))
		sb.WriteString(`"}`)
	}
}

func ExtractWithTitle(htmlContent string) (string, string, error) {
	result, err := Extract(htmlContent)
	if err != nil {
		return "", "", err
	}
	return result.Title, result.Text, nil
}

func ExtractImages(htmlContent string) ([]ImageInfo, error) {
	result, err := Extract(htmlContent)
	if err != nil {
		return nil, err
	}
	return result.Images, nil
}

func ExtractVideos(htmlContent string) ([]VideoInfo, error) {
	result, err := Extract(htmlContent)
	if err != nil {
		return nil, err
	}
	return result.Videos, nil
}

func ExtractAudios(htmlContent string) ([]AudioInfo, error) {
	result, err := Extract(htmlContent)
	if err != nil {
		return nil, err
	}
	return result.Audios, nil
}

func ExtractLinks(htmlContent string) ([]LinkInfo, error) {
	result, err := Extract(htmlContent)
	if err != nil {
		return nil, err
	}
	return result.Links, nil
}

func ConfigForRSS() ExtractConfig {
	return ExtractConfig{
		ExtractArticle:    false,
		PreserveImages:    true,
		PreserveLinks:     true,
		PreserveVideos:    false,
		PreserveAudios:    false,
		InlineImageFormat: "none",
	}
}

func ConfigForSearchIndex() ExtractConfig {
	return ExtractConfig{
		ExtractArticle:    true,
		PreserveImages:    true,
		PreserveLinks:     true,
		PreserveVideos:    true,
		PreserveAudios:    true,
		InlineImageFormat: "none",
	}
}

func ConfigForSummary() ExtractConfig {
	return ExtractConfig{
		ExtractArticle:    true,
		PreserveImages:    false,
		PreserveLinks:     false,
		PreserveVideos:    false,
		PreserveAudios:    false,
		InlineImageFormat: "none",
	}
}

func ConfigForMarkdown() ExtractConfig {
	return ExtractConfig{
		ExtractArticle:    true,
		PreserveImages:    true,
		PreserveLinks:     true,
		PreserveVideos:    false,
		PreserveAudios:    false,
		InlineImageFormat: "markdown",
	}
}

func Summarize(htmlContent string, maxWords int) (string, error) {
	result, err := Extract(htmlContent)
	if err != nil {
		return "", err
	}

	if maxWords <= 0 {
		return result.Text, nil
	}

	words := strings.Fields(result.Text)
	if len(words) <= maxWords {
		return result.Text, nil
	}

	summary := strings.Join(words[:maxWords], " ")
	if len(words) > maxWords {
		summary += "..."
	}
	return summary, nil
}

func ExtractAndClean(htmlContent string) (string, error) {
	result, err := Extract(htmlContent)
	if err != nil {
		return "", err
	}

	cleaned := internal.WhitespaceRegex.ReplaceAllString(result.Text, " ")
	lines := strings.Split(cleaned, "\n")
	var nonEmptyLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			nonEmptyLines = append(nonEmptyLines, trimmed)
		}
	}

	return strings.Join(nonEmptyLines, "\n\n"), nil
}

func GetReadingTime(htmlContent string) (float64, error) {
	result, err := Extract(htmlContent)
	if err != nil {
		return 0, err
	}
	return result.ReadingTime.Minutes(), nil
}

func GetWordCount(htmlContent string) (int, error) {
	result, err := Extract(htmlContent)
	if err != nil {
		return 0, err
	}
	return result.WordCount, nil
}

func ExtractTitle(htmlContent string) (string, error) {
	result, err := Extract(htmlContent)
	if err != nil {
		return "", err
	}
	return result.Title, nil
}
