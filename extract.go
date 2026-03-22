package html

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	htmlstd "html"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/cybergodev/html/internal"
	stdxhtml "golang.org/x/net/html"
)

// linkPlaceholderRegex matches [LINK:N]text[/LINK] pattern for inline link formatting.
// Pre-compiled at package initialization for performance.
var linkPlaceholderRegex = regexp.MustCompile(`\[LINK:(\d+)\]([^\[]*?)\[/LINK\]`)

// Timeout goroutine management constants.
// These prevent goroutine leaks when many timeout operations occur.
const (
	// maxTimeoutGoroutines limits concurrent timeout goroutines to prevent
	// resource exhaustion. Value 1000 allows ~1GB of goroutine stack overhead
	// assuming 1MB stack per goroutine, which is a reasonable safety limit.
	// When exceeded, new operations return ErrProcessingTimeout immediately.
	maxTimeoutGoroutines = 1000
)

// activeTimeoutGoroutines tracks the number of currently running timeout goroutines.
var activeTimeoutGoroutines atomic.Int64

// Extract extracts content from HTML bytes with automatic encoding detection.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// For better performance with multiple extractions, create a Processor directly:
//
//	processor, _ := html.New()
//	defer processor.Close()
//	for _, html := range htmlDocs {
//	    result, _ := processor.Extract(html)  // Uses cache
//	}
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
//
// An optional Config can be provided to customize extraction behavior.
// If no config is provided, DefaultConfig() is used.
//
// Examples:
//
//	// With default config
//	result, err := html.Extract(htmlBytes)
//
//	// With custom config
//	cfg := html.DefaultConfig()
//	cfg.PreserveImages = false
//	result, err := html.Extract(htmlBytes, cfg)
func Extract(htmlBytes []byte, cfg ...Config) (*Result, error) {
	c := resolveConfig(cfg...)
	processor, err := getProcessorWithConfig(c)
	if err != nil {
		return nil, err
	}
	defer putProcessorWithConfig(processor, c)
	return processor.Extract(htmlBytes)
}

// ExtractFromFile extracts content from an HTML file with automatic encoding detection.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
//
// An optional Config can be provided to customize extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractFromFile(filePath string, cfg ...Config) (*Result, error) {
	c := resolveConfig(cfg...)
	processor, err := getProcessorWithConfig(c)
	if err != nil {
		return nil, err
	}
	defer putProcessorWithConfig(processor, c)
	return processor.ExtractFromFile(filePath)
}

// ExtractText extracts plain text from HTML bytes with automatic encoding detection.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// For better performance with multiple extractions, create a Processor directly.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
//
// An optional Config can be provided to customize extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractText(htmlBytes []byte, cfg ...Config) (string, error) {
	c := resolveConfig(cfg...)
	processor, err := getProcessorWithConfig(c)
	if err != nil {
		return "", err
	}
	defer putProcessorWithConfig(processor, c)
	return processor.ExtractText(htmlBytes)
}

// ExtractTextFromFile extracts plain text from an HTML file with automatic encoding detection.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
//
// An optional Config can be provided to customize extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractTextFromFile(filePath string, cfg ...Config) (string, error) {
	c := resolveConfig(cfg...)
	processor, err := getProcessorWithConfig(c)
	if err != nil {
		return "", err
	}
	defer putProcessorWithConfig(processor, c)
	return processor.ExtractTextFromFile(filePath)
}

// Extract extracts content from HTML bytes with automatic encoding detection.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
//
// This is the primary method for HTML content extraction when the source encoding
// may not be UTF-8, such as content from HTTP responses, databases, or files.
func (p *Processor) Extract(htmlBytes []byte) (result *Result, err error) {
	// Defense-in-depth: recover from unexpected panics
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrInternalPanic, r)
		}
	}()

	// Validate processor state and input size
	if err := p.validateInput(htmlBytes); err != nil {
		return nil, err
	}

	startTime := time.Now()

	// Detect encoding and convert to UTF-8
	utf8String, err := p.detectEncoding(htmlBytes)
	if err != nil {
		return nil, err
	}

	// Check cache only if caching is enabled
	var cacheKey string
	if p.config.MaxCacheEntries > 0 {
		cacheKey = p.generateCacheKey(utf8String)
		if cached := p.cache.Get(cacheKey); cached != nil {
			p.stats.cacheHits.Add(1)
			p.stats.totalProcessed.Add(1)
			if cachedResult, ok := cached.(*Result); ok {
				return cachedResult, nil
			}
		}
		p.stats.cacheMisses.Add(1)
	}

	// Process the content
	if p.config.ProcessingTimeout > 0 {
		result, err = withTimeout(p.config.ProcessingTimeout, func() (*Result, error) {
			return p.processContent(utf8String)
		})
	} else {
		result, err = p.processContent(utf8String)
	}

	if err != nil {
		p.stats.errorCount.Add(1)
		// Log specific error types
		if errors.Is(err, ErrProcessingTimeout) {
			p.audit.RecordTimeout(p.config.ProcessingTimeout)
		} else if errors.Is(err, ErrMaxDepthExceeded) {
			p.audit.RecordDepthViolation(p.config.MaxDepth+1, p.config.MaxDepth)
		}
		return nil, err
	}

	processingTime := time.Since(startTime)
	result.ProcessingTime = processingTime
	p.stats.totalProcessTime.Add(int64(processingTime))
	p.stats.totalProcessed.Add(1)

	// Only cache if caching is enabled and we have a cache key
	if p.config.MaxCacheEntries > 0 && cacheKey != "" {
		p.cache.Set(cacheKey, result)
	}

	return result, nil
}

// ExtractWithContext extracts content from HTML bytes with context support for cancellation.
// This method provides cooperative cancellation, allowing long-running extractions to be
// interrupted when the context is cancelled.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
//
// Example usage:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	result, err := processor.ExtractWithContext(ctx, htmlBytes)
//	if errors.Is(err, context.Canceled) {
//	    // Extraction was cancelled
//	}
func (p *Processor) ExtractWithContext(ctx context.Context, htmlBytes []byte) (result *Result, err error) {
	// Defense-in-depth: recover from unexpected panics
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrInternalPanic, r)
		}
	}()

	// Early cancellation check
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Validate processor state and input size
	if err := p.validateInput(htmlBytes); err != nil {
		return nil, err
	}

	startTime := time.Now()

	// Check cancellation before encoding detection
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Detect encoding and convert to UTF-8
	utf8String, err := p.detectEncoding(htmlBytes)
	if err != nil {
		return nil, err
	}

	// Check cache only if caching is enabled
	var cacheKey string
	if p.config.MaxCacheEntries > 0 {
		cacheKey = p.generateCacheKey(utf8String)
		if cached := p.cache.Get(cacheKey); cached != nil {
			p.stats.cacheHits.Add(1)
			p.stats.totalProcessed.Add(1)
			if cachedResult, ok := cached.(*Result); ok {
				return cachedResult, nil
			}
		}
		p.stats.cacheMisses.Add(1)
	}

	// Check cancellation before content processing
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Process the content with context-aware timeout handling
	if p.config.ProcessingTimeout > 0 {
		// Use context-aware timeout
		ctxWithTimeout, cancel := context.WithTimeout(ctx, p.config.ProcessingTimeout)
		defer cancel()

		done := make(chan struct{})
		var procErr error
		go func() {
			defer close(done)
			result, procErr = p.processContent(utf8String)
		}()

		select {
		case <-ctxWithTimeout.Done():
			if errors.Is(ctxWithTimeout.Err(), context.DeadlineExceeded) {
				p.audit.RecordTimeout(p.config.ProcessingTimeout)
				return nil, ErrProcessingTimeout
			}
			return nil, ctxWithTimeout.Err()
		case <-done:
			err = procErr
		}
	} else {
		// No timeout, but still respect context cancellation
		result, err = p.processContentWithContext(ctx, utf8String)
	}

	if err != nil {
		p.stats.errorCount.Add(1)
		if errors.Is(err, ErrMaxDepthExceeded) {
			p.audit.RecordDepthViolation(p.config.MaxDepth+1, p.config.MaxDepth)
		}
		return nil, err
	}

	processingTime := time.Since(startTime)
	result.ProcessingTime = processingTime
	p.stats.totalProcessTime.Add(int64(processingTime))
	p.stats.totalProcessed.Add(1)

	// Only cache if caching is enabled and we have a cache key
	if p.config.MaxCacheEntries > 0 && cacheKey != "" {
		p.cache.Set(cacheKey, result)
	}

	return result, nil
}

// processContentWithContext processes HTML content with context cancellation support.
// This method implements cooperative cancellation at key processing stages.
func (p *Processor) processContentWithContext(ctx context.Context, htmlContent string) (*Result, error) {
	// Check for cancellation at start
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if strings.TrimSpace(htmlContent) == "" {
		return &Result{}, nil
	}

	originalHTML := htmlContent

	// Check context before sanitization
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if p.config.EnableSanitization {
		if p.audit != nil && p.config.Audit.Enabled {
			adapter := &auditRecorderAdapter{collector: p.audit}
			htmlContent = internal.SanitizeHTMLWithAudit(htmlContent, adapter)
		} else {
			htmlContent = internal.SanitizeHTML(htmlContent)
		}
	}

	// Check context before HTML parsing
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	doc, err := stdxhtml.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidHTML, err)
	}

	// Check context before depth validation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if err := p.validateDepthTraversal(doc, 0); err != nil {
		return nil, err
	}

	// Check context before document extraction
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return p.extractFromDocument(doc, originalHTML)
}

// ExtractFromFile extracts content from an HTML file with automatic encoding detection.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
// Use this when you have a file path instead of raw bytes.
func (p *Processor) ExtractFromFile(filePath string) (result *Result, err error) {
	// Defense-in-depth: recover from unexpected panics
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrInternalPanic, r)
		}
	}()

	// Validate processor state (no input size check for file paths)
	if p == nil || p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	data, err := p.validateAndReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return p.Extract(data)
}

// ExtractFromFileWithContext extracts content from an HTML file with context support.
// This method provides cooperative cancellation, allowing long-running extractions to be
// interrupted when the context is cancelled.
//
// Example usage:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	result, err := processor.ExtractFromFileWithContext(ctx, "page.html")
func (p *Processor) ExtractFromFileWithContext(ctx context.Context, filePath string) (result *Result, err error) {
	// Defense-in-depth: recover from unexpected panics
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrInternalPanic, r)
		}
	}()

	// Early cancellation check
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Validate processor state (no input size check for file paths)
	if p == nil || p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	data, err := p.validateAndReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return p.ExtractWithContext(ctx, data)
}

// ExtractText extracts plain text from HTML bytes with automatic encoding detection.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
// This is a convenience method that returns only the text content without other metadata.
func (p *Processor) ExtractText(htmlBytes []byte) (string, error) {
	result, err := p.Extract(htmlBytes)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// ExtractTextFromFile extracts plain text from an HTML file with automatic encoding detection.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
// Use this when you have a file path instead of raw bytes.
// This is a convenience method that returns only the text content without other metadata.
func (p *Processor) ExtractTextFromFile(filePath string) (string, error) {
	result, err := p.ExtractFromFile(filePath)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// withTimeout executes a function with a timeout, returning its result or an error if timeout expires.
// This is a generic helper that eliminates code duplication in timeout handling.
//
// IMPORTANT: If the timeout is reached, the timeout error is returned immediately, but the
// function fn() continues executing in the background until it completes. This is because
// Go does not support cooperative cancellation for arbitrary functions. The function fn()
// should be designed to complete relatively quickly to avoid resource accumulation.
// For long-running operations, consider using context-aware processing instead.
//
// Goroutine Safety: To prevent goroutine leaks under heavy load, this function limits the
// maximum number of concurrent timeout goroutines. If the limit is exceeded, an error is returned.
func withTimeout[T any](timeout time.Duration, fn func() (T, error)) (T, error) {
	// Check if we've exceeded the maximum number of concurrent timeout goroutines.
	// This prevents resource exhaustion when processing many documents with timeouts.
	if active := activeTimeoutGoroutines.Load(); active >= maxTimeoutGoroutines {
		var zero T
		return zero, fmt.Errorf("%w: too many concurrent operations (%d active)", ErrProcessingTimeout, active)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type result struct {
		res T
		err error
	}

	resultChan := make(chan result, 1)

	// Track active goroutines to prevent unbounded growth
	activeTimeoutGoroutines.Add(1)

	// Launch the operation in a goroutine
	go func() {
		// Ensure we decrement the counter when done
		defer activeTimeoutGoroutines.Add(-1)

		res, err := fn()
		// Try to send the result. If context is done, the result is discarded.
		// This is intentional - we don't want to block if nobody is listening.
		select {
		case resultChan <- result{res: res, err: err}:
			// Result sent successfully
		case <-ctx.Done():
			// Context cancelled/timed out, discard result
		}
	}()

	select {
	case res := <-resultChan:
		return res.res, res.err
	case <-ctx.Done():
		var zero T
		// Note: The goroutine above continues running until fn() completes.
		// This is a known limitation of non-cooperative cancellation in Go.
		// The resultChan has buffer size 1, so the goroutine won't block forever.
		// The activeTimeoutGoroutines counter ensures we don't spawn too many.
		return zero, ErrProcessingTimeout
	}
}

func (p *Processor) processContent(htmlContent string) (*Result, error) {
	if strings.TrimSpace(htmlContent) == "" {
		return &Result{}, nil
	}

	originalHTML := htmlContent

	if p.config.EnableSanitization {
		// Use audit-enabled sanitization if audit is configured
		if p.audit != nil && p.config.Audit.Enabled {
			adapter := &auditRecorderAdapter{collector: p.audit}
			htmlContent = internal.SanitizeHTMLWithAudit(htmlContent, adapter)
		} else {
			htmlContent = internal.SanitizeHTML(htmlContent)
		}
	}

	doc, err := stdxhtml.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidHTML, err)
	}

	// Validate depth during extraction to avoid duplicate traversal
	if err := p.validateDepthTraversal(doc, 0); err != nil {
		return nil, err
	}

	return p.extractFromDocument(doc, originalHTML)
}

// validateDepthTraversal validates DOM tree depth using an iterative approach
// to avoid potential stack overflow on deeply nested documents.
// This is more efficient than separately calling validateDepth and extractFromDocument.
func (p *Processor) validateDepthTraversal(root *Node, initialDepth int) error {
	// Use iterative approach with explicit stack to avoid stack overflow
	// on deeply nested documents (MaxDepth can be up to 500)
	type stackEntry struct {
		node  *Node
		depth int
	}

	// Pre-allocate stack with reasonable initial capacity
	stack := make([]stackEntry, 0, 64)
	stack = append(stack, stackEntry{root, initialDepth})

	for len(stack) > 0 {
		entry := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if entry.depth > p.config.MaxDepth {
			return ErrMaxDepthExceeded
		}

		// Add children to stack in reverse order for correct traversal order
		for c := entry.node.FirstChild; c != nil; c = c.NextSibling {
			stack = append(stack, stackEntry{c, entry.depth + 1})
		}
	}
	return nil
}

func (p *Processor) extractFromDocument(doc *Node, htmlContent string) (*Result, error) {
	result := &Result{}
	result.Title = p.extractTitle(doc)

	contentNode := doc
	if p.config.ExtractArticle {
		if article := p.extractArticleNode(doc); article != nil {
			contentNode = article
		}
	}
	contentNode = internal.CleanContentNode(contentNode)

	imageFormat := strings.ToLower(strings.TrimSpace(p.config.InlineImageFormat))
	if imageFormat == "" {
		imageFormat = "none"
	}
	linkFormat := strings.ToLower(strings.TrimSpace(p.config.InlineLinkFormat))
	if linkFormat == "" {
		linkFormat = "none"
	}

	// Use placeholder path if either image or link format is not "none"
	if imageFormat != "none" || linkFormat != "none" {
		images := p.extractImagesWithPosition(contentNode)
		links := p.extractLinksWithPosition(contentNode)

		if p.config.PreserveImages {
			result.Images = images
		}
		if p.config.PreserveLinks {
			result.Links = links
		}

		sb := internal.GetBuilder()
		sb.Grow(initialTextSize)
		imageCounter := 0
		linkCounter := 0
		internal.ExtractTextWithStructureAndImages(contentNode, sb, &imageCounter, &linkCounter, p.config.TableFormat)
		textWithPlaceholders := internal.CleanText(sb.String(), nil)
		internal.PutBuilder(sb)

		// Apply formatters in order: images first, then links
		result.Text = p.formatInlineImages(textWithPlaceholders, images, imageFormat)
		result.Text = p.formatInlineLinks(result.Text, links, linkFormat)
	} else {
		result.Text = p.extractTextContent(contentNode, p.config.TableFormat)

		if p.config.PreserveImages {
			result.Images = p.extractImages(contentNode)
		}
		if p.config.PreserveLinks {
			result.Links = p.extractLinks(contentNode)
		}
	}

	result.WordCount = p.countWords(result.Text)
	result.ReadingTime = p.calculateReadingTime(result.WordCount)

	if p.config.PreserveVideos {
		result.Videos = p.extractVideos(doc, htmlContent)
	}
	if p.config.PreserveAudios {
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
	// Pre-allocate map with initial capacity to reduce resizing
	candidates := make(map[*Node]int, initialMapCap)

	// Determine scorer once outside the loop to avoid repeated nil checks
	scorer := p.scorer
	useDefaultScorer := scorer == nil

	internal.WalkNodes(doc, func(n *Node) bool {
		if n.Type == ElementNode {
			var score int
			if useDefaultScorer {
				score = internal.ScoreContentNode(n)
			} else {
				score = scorer.Score(n)
			}
			if score > 0 {
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
	sb := internal.GetBuilder()
	sb.Grow(initialTextSize)
	internal.ExtractTextWithStructureAndImages(node, sb, nil, nil, tableFormat)
	result := internal.CleanText(sb.String(), nil)
	internal.PutBuilder(sb)
	return result
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
			placeholder := "[IMAGE:" + strconv.Itoa(images[i].Position) + "]"
			altText := images[i].Alt
			if altText == "" {
				altText = "Image " + strconv.Itoa(images[i].Position)
			}
			markdown := "![" + altText + "](" + images[i].URL + ")"
			replacements = append(replacements, placeholder, markdown)
		}
	case "html":
		// Use pooled builder for HTML image tags
		htmlImg := internal.GetBuilder()
		for i := range images {
			if images[i].Position == 0 {
				continue
			}
			placeholder := "[IMAGE:" + strconv.Itoa(images[i].Position) + "]"
			htmlImg.Reset()
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
		internal.PutBuilder(htmlImg)
	}

	if len(replacements) > 0 {
		replacer := strings.NewReplacer(replacements...)
		return replacer.Replace(textWithPlaceholders)
	}

	return textWithPlaceholders
}

func (p *Processor) formatInlineLinks(textWithPlaceholders string, links []LinkInfo, format string) string {
	if len(links) == 0 || format == "none" {
		return textWithPlaceholders
	}

	// Create a map of link position to link info for fast lookup
	linkMap := make(map[int]LinkInfo, len(links))
	for _, link := range links {
		if link.Position > 0 {
			linkMap[link.Position] = link
		}
	}

	// Use pre-compiled regex for performance
	result := linkPlaceholderRegex.ReplaceAllStringFunc(textWithPlaceholders, func(match string) string {
		// Extract position and text from the match
		submatches := linkPlaceholderRegex.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}

		position := 0
		fmt.Sscanf(submatches[1], "%d", &position)
		linkText := submatches[2]

		link, ok := linkMap[position]
		if !ok {
			return linkText // Return just the text if link not found
		}

		if linkText == "" {
			linkText = "Link " + strconv.Itoa(position)
		}

		switch format {
		case "markdown":
			return "[" + linkText + "](" + link.URL + ")"
		case "html":
			if link.Title != "" {
				return fmt.Sprintf(`<a href="%s" title="%s">%s</a>`,
					htmlstd.EscapeString(link.URL),
					htmlstd.EscapeString(link.Title),
					htmlstd.EscapeString(linkText))
			}
			return fmt.Sprintf(`<a href="%s">%s</a>`,
				htmlstd.EscapeString(link.URL),
				htmlstd.EscapeString(linkText))
		default:
			return linkText
		}
	})

	return result
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

func (p *Processor) extractLinksWithPosition(node *Node) []LinkInfo {
	links := make([]LinkInfo, 0, initialSliceCap)
	position := 0

	internal.WalkNodes(node, func(n *Node) bool {
		if n.Type == ElementNode && n.Data == "a" {
			position++
			link := p.parseLinkNode(n)
			link.Position = position
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

// generateCacheKey creates a hash for cache key generation.
// Uses a simplified xxHash-inspired algorithm optimized for speed.
// Uses multi-point sampling for large documents to better distinguish similar content.
// Optimized to minimize CPU overhead while maintaining good distribution.
func (p *Processor) generateCacheKey(content string) string {
	// Pack boolean flags into a single uint8 for faster hashing
	// Bits: 0=ExtractArticle, 1=PreserveImages, 2=PreserveLinks, 3=PreserveVideos, 4=PreserveAudios
	flags := uint8(0)
	if p.config.ExtractArticle {
		flags |= 1 << 0
	}
	if p.config.PreserveImages {
		flags |= 1 << 1
	}
	if p.config.PreserveLinks {
		flags |= 1 << 2
	}
	if p.config.PreserveVideos {
		flags |= 1 << 3
	}
	if p.config.PreserveAudios {
		flags |= 1 << 4
	}

	// Initialize hash with prime constants (xxHash-inspired)
	var h uint64 = 0x9E3779B185EBCA87 // Golden ratio fractional bits

	// Mix flags
	h ^= uint64(flags)
	h = hashMix(h)

	// Mix string options with reduced operations
	h = hashMixString(h, p.config.InlineImageFormat)
	h = hashMixString(h, p.config.InlineLinkFormat)
	h = hashMixString(h, p.config.TableFormat)

	contentLen := len(content)
	if contentLen <= maxCacheKeySize {
		// Hash the entire content using simplified algorithm
		h = hashMixBytes(h, internal.StringToBytes(content))
	} else {
		// Multi-point sampling for large documents to reduce collision risk.
		// Sample from 5 positions: beginning, 25%, 50%, 75%, and end.
		// This provides better distribution than 3-point sampling while
		// maintaining acceptable performance overhead.
		const sampleCount = 5
		sampleSize := cacheKeySample / sampleCount
		if sampleSize < 512 {
			sampleSize = 512
		}

		for i := 0; i < sampleCount; i++ {
			var start, end int
			if i == sampleCount-1 {
				// Last sample: end of content
				end = contentLen
				start = contentLen - sampleSize
				if start < 0 {
					start = 0
				}
			} else {
				// Distributed samples across the document
				offset := (contentLen * i) / (sampleCount - 1)
				start = offset
				end = start + sampleSize
				if end > contentLen {
					end = contentLen
				}
			}

			if start < end {
				h = hashMixBytes(h, internal.StringToBytes(content[start:end]))
			}
		}

		// Mix content length
		h ^= uint64(contentLen)
		h = hashMix(h)
	}

	// Final avalanche
	h ^= h >> 33
	h *= 0xFF51AFD7ED558CCD
	h ^= h >> 33

	// Generate 16-byte hash for better collision resistance
	// First 8 bytes: primary hash
	// Second 8 bytes: secondary hash using different mixing constant
	var buf [16]byte
	binary.LittleEndian.PutUint64(buf[:8], h)

	// Secondary hash with different seed for additional entropy
	h2 := h ^ 0x9E3779B185EBCA87 // XOR with golden ratio
	h2 = hashMix(h2)
	binary.LittleEndian.PutUint64(buf[8:], h2)

	return string(buf[:])
}

// hashMix performs a single mixing step (inlined for performance)
// Optimized with faster constants and fewer operations
func hashMix(h uint64) uint64 {
	// Use faster multiplication constant (MurmurHash3-style finalizer)
	// This has better avalanche properties and faster computation
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	return h
}

// hashMixString hashes a string into the hash state
// Optimized with unsafe direct memory reads and batch processing
// Processes 64 bytes per iteration for maximum throughput
func hashMixString(h uint64, s string) uint64 {
	n := len(s)
	if n == 0 {
		return h
	}

	// Hash length
	h ^= uint64(n)
	h = hashMix(h)

	// Use unsafe for direct memory access - avoids slice allocation
	ptr := unsafe.Pointer(unsafe.StringData(s))

	// Process 64 bytes at a time for maximum throughput
	i := 0
	for i+64 <= n {
		v0 := *(*uint64)(unsafe.Add(ptr, i))
		v1 := *(*uint64)(unsafe.Add(ptr, i+8))
		v2 := *(*uint64)(unsafe.Add(ptr, i+16))
		v3 := *(*uint64)(unsafe.Add(ptr, i+24))
		v4 := *(*uint64)(unsafe.Add(ptr, i+32))
		v5 := *(*uint64)(unsafe.Add(ptr, i+40))
		v6 := *(*uint64)(unsafe.Add(ptr, i+48))
		v7 := *(*uint64)(unsafe.Add(ptr, i+56))

		h ^= v0
		h = hashMix(h)
		h ^= v1 ^ v4
		h = hashMix(h)
		h ^= v2 ^ v5
		h = hashMix(h)
		h ^= v3 ^ v6
		h = hashMix(h)
		h ^= v7
		h = hashMix(h)
		i += 64
	}

	// Process remaining 32-byte chunks
	for i+32 <= n {
		v0 := *(*uint64)(unsafe.Add(ptr, i))
		v1 := *(*uint64)(unsafe.Add(ptr, i+8))
		v2 := *(*uint64)(unsafe.Add(ptr, i+16))
		v3 := *(*uint64)(unsafe.Add(ptr, i+24))

		h ^= v0 ^ v2
		h = hashMix(h)
		h ^= v1 ^ v3
		h = hashMix(h)
		i += 32
	}

	// Process remaining 8-byte chunks
	for i+8 <= n {
		v := *(*uint64)(unsafe.Add(ptr, i))
		h ^= v
		h = hashMix(h)
		i += 8
	}

	// Handle remaining bytes
	if i < n {
		var v uint64
		shift := uint(0)
		for j := i; j < n; j++ {
			v |= uint64(s[j]) << shift
			shift += 8
		}
		h ^= v
		h = hashMix(h)
	}

	return h
}

// hashMixBytes hashes a byte slice into the hash state
// Optimized with unsafe direct memory reads and batch processing
// Processes 64 bytes per iteration for maximum throughput
func hashMixBytes(h uint64, data []byte) uint64 {
	n := len(data)
	if n == 0 {
		return h
	}

	// Use unsafe for direct memory access
	ptr := unsafe.Pointer(unsafe.SliceData(data))

	// Process 64 bytes at a time (8 x 64-bit words) for maximum throughput
	// This reduces loop iterations and hashMix calls by 8x
	i := 0
	for i+64 <= n {
		// Load 8 words at once
		v0 := *(*uint64)(unsafe.Add(ptr, i))
		v1 := *(*uint64)(unsafe.Add(ptr, i+8))
		v2 := *(*uint64)(unsafe.Add(ptr, i+16))
		v3 := *(*uint64)(unsafe.Add(ptr, i+24))
		v4 := *(*uint64)(unsafe.Add(ptr, i+32))
		v5 := *(*uint64)(unsafe.Add(ptr, i+40))
		v6 := *(*uint64)(unsafe.Add(ptr, i+48))
		v7 := *(*uint64)(unsafe.Add(ptr, i+56))

		// Combine with rotating accumulation for better distribution
		h ^= v0
		h = hashMix(h)
		h ^= v1 ^ v4
		h = hashMix(h)
		h ^= v2 ^ v5
		h = hashMix(h)
		h ^= v3 ^ v6
		h = hashMix(h)
		h ^= v7
		h = hashMix(h)
		i += 64
	}

	// Process remaining 32-byte chunks
	for i+32 <= n {
		v0 := *(*uint64)(unsafe.Add(ptr, i))
		v1 := *(*uint64)(unsafe.Add(ptr, i+8))
		v2 := *(*uint64)(unsafe.Add(ptr, i+16))
		v3 := *(*uint64)(unsafe.Add(ptr, i+24))

		h ^= v0 ^ v2
		h = hashMix(h)
		h ^= v1 ^ v3
		h = hashMix(h)
		i += 32
	}

	// Process remaining 8-byte chunks
	for i+8 <= n {
		v := *(*uint64)(unsafe.Add(ptr, i))
		h ^= v
		h = hashMix(h)
		i += 8
	}

	// Handle remaining bytes
	if i < n {
		var v uint64
		shift := uint(0)
		for j := i; j < n; j++ {
			v |= uint64(data[j]) << shift
			shift += 8
		}
		h ^= v
		h = hashMix(h)
	}

	return h
}
