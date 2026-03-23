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
	c, err := resolveConfig(cfg...)
	if err != nil {
		return nil, err
	}
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
	c, err := resolveConfig(cfg...)
	if err != nil {
		return nil, err
	}
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
	c, err := resolveConfig(cfg...)
	if err != nil {
		return "", err
	}
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
	c, err := resolveConfig(cfg...)
	if err != nil {
		return "", err
	}
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
func (p *Processor) validateDepthTraversal(root *stdxhtml.Node, initialDepth int) error {
	// Use iterative approach with explicit stack to avoid stack overflow
	// on deeply nested documents (MaxDepth can be up to 500)
	type stackEntry struct {
		node  *stdxhtml.Node
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

func (p *Processor) extractFromDocument(doc *stdxhtml.Node, htmlContent string) (*Result, error) {
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

func (p *Processor) extractTitle(doc *stdxhtml.Node) string {
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

func (p *Processor) extractArticleNode(doc *stdxhtml.Node) *stdxhtml.Node {
	if doc == nil {
		return nil
	}
	// Pre-allocate map with initial capacity to reduce resizing
	candidates := make(map[*stdxhtml.Node]int, initialMapCap)

	// Determine scorer once outside the loop to avoid repeated nil checks
	scorer := p.scorer
	useDefaultScorer := scorer == nil

	internal.WalkNodes(doc, func(n *stdxhtml.Node) bool {
		if n.Type == stdxhtml.ElementNode {
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

func (p *Processor) extractTextContent(node *stdxhtml.Node, tableFormat string) string {
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

func (p *Processor) extractImages(node *stdxhtml.Node) []ImageInfo {
	images := make([]ImageInfo, 0, initialSliceCap)

	internal.WalkNodes(node, func(n *stdxhtml.Node) bool {
		if n.Type == stdxhtml.ElementNode && n.Data == "img" {
			img := p.parseImageNode(n, 0)
			if img.URL != "" {
				images = append(images, img)
			}
		}
		return true
	})

	return images
}

func (p *Processor) extractImagesWithPosition(node *stdxhtml.Node) []ImageInfo {
	images := make([]ImageInfo, 0, initialSliceCap)
	position := 0

	internal.WalkNodes(node, func(n *stdxhtml.Node) bool {
		if n.Type == stdxhtml.ElementNode && n.Data == "img" {
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

func (p *Processor) parseImageNode(n *stdxhtml.Node, position int) ImageInfo {
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

func (p *Processor) extractLinks(node *stdxhtml.Node) []LinkInfo {
	links := make([]LinkInfo, 0, initialSliceCap)

	internal.WalkNodes(node, func(n *stdxhtml.Node) bool {
		if n.Type == stdxhtml.ElementNode && n.Data == "a" {
			link := p.parseLinkNode(n)
			if link.URL != "" {
				links = append(links, link)
			}
		}
		return true
	})

	return links
}

func (p *Processor) extractLinksWithPosition(node *stdxhtml.Node) []LinkInfo {
	links := make([]LinkInfo, 0, initialSliceCap)
	position := 0

	internal.WalkNodes(node, func(n *stdxhtml.Node) bool {
		if n.Type == stdxhtml.ElementNode && n.Data == "a" {
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

func (p *Processor) parseLinkNode(n *stdxhtml.Node) LinkInfo {
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
// Uses xxHash-style algorithm optimized for maximum throughput.
// Uses multi-point sampling for large documents to reduce collision risk.
//
// SECURITY: This function uses 5-point sampling for better collision resistance
// against hash-flooding attacks. The sampling strategy ensures that
// modifications anywhere in the document are likely to change the hash.
//
// Performance optimization: Uses inline hashing to reduce function call overhead
// and processes data in larger chunks for better CPU cache utilization.
func (p *Processor) generateCacheKey(content string) string {
	// Initialize hash with seed
	h := prime64_5

	// Pack boolean flags into a single uint8
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

	// Mix flags and string options - optimized inline
	h ^= uint64(flags)
	h = hashMixInline(h)

	// Inline short string hashing to reduce function call overhead
	h = hashMixStringInline(h, p.config.InlineImageFormat)
	h = hashMixStringInline(h, p.config.InlineLinkFormat)
	h = hashMixStringInline(h, p.config.TableFormat)

	contentLen := len(content)
	if contentLen <= maxCacheKeySize {
		// Hash the entire content - use zero-copy conversion
		h = hashMixBytesInline(h, internal.StringToBytes(content))
	} else {
		// SECURITY: Multi-point sampling for large documents
		// Using 5 sampling points for better collision resistance
		const sampleCount = 5
		sampleSize := cacheKeySample / sampleCount
		if sampleSize < 256 {
			sampleSize = 256
		}

		// Pre-compute step size for even distribution
		for i := 0; i < sampleCount; i++ {
			var start, end int
			if i == sampleCount-1 {
				end = contentLen
				start = contentLen - sampleSize
				if start < 0 {
					start = 0
				}
			} else if i == 0 {
				start = 0
				end = sampleSize
				if end > contentLen {
					end = contentLen
				}
			} else {
				offset := (contentLen * i) / (sampleCount - 1)
				start = offset - sampleSize/2
				if start < 0 {
					start = 0
				}
				end = start + sampleSize
				if end > contentLen {
					end = contentLen
					start = end - sampleSize
					if start < 0 {
						start = 0
					}
				}
			}

			if start < end {
				h = hashMixBytesInline(h, internal.StringToBytes(content[start:end]))
			}
		}

		// Mix content length for additional uniqueness
		h ^= uint64(contentLen) * prime64_4
		h = hashMixInline(h)
	}

	// Final avalanche for better distribution
	h ^= h >> 33
	h *= prime64_2
	h ^= h >> 29

	// Generate 16-byte hash for collision resistance
	var buf [16]byte
	binary.LittleEndian.PutUint64(buf[:8], h)
	h2 := h ^ prime64_1
	h2 = hashMixInline(h2)
	binary.LittleEndian.PutUint64(buf[8:], h2)

	return string(buf[:])
}

// xxHash-inspired constants for fast hashing
const (
	prime64_1 uint64 = 0x9E3779B185EBCA87
	prime64_2 uint64 = 0xC2B2AE3D27D4EB4F
	prime64_3 uint64 = 0x165667B19E3779F9
	prime64_4 uint64 = 0x85EBCA6798F3B1AD
	prime64_5 uint64 = 0x27D4EB2F165667C5
)


// hashMixInline is an inline version of hashMixFast for critical paths.
// This reduces function call overhead in hot code paths.
//
//nolint:deadcode // Performance: inlined version for cache key generation hot path
func hashMixInline(h uint64) uint64 {
	h ^= h >> 31
	h *= prime64_3
	return h
}

// hashMixStringInline hashes a string using optimized inline operations.
// This is an inline-optimized version for the cache key generation hot path.
//
//nolint:deadcode // Performance: inlined version for cache key generation hot path
func hashMixStringInline(h uint64, s string) uint64 {
	n := len(s)
	if n == 0 {
		return h
	}

	// Mix length first
	h ^= uint64(n) * prime64_5

	// For very short strings, use safe byte-by-byte processing
	if n < 8 {
		var v uint64
		for j := 0; j < n; j++ {
			v = (v << 8) | uint64(s[j])
		}
		h ^= v * prime64_4
		h = hashMixInline(h)
		return h
	}

	ptr := unsafe.Pointer(unsafe.StringData(s))
	i := 0

	// Process 32 bytes at a time using 4 accumulators
	var acc1, acc2, acc3, acc4 uint64 = prime64_1, prime64_2, prime64_3, prime64_4

	for i+32 <= n {
		acc1 += *(*uint64)(unsafe.Add(ptr, i)) * prime64_2
		acc1 = (acc1 << 31) | (acc1 >> 33)
		acc1 *= prime64_1

		acc2 += *(*uint64)(unsafe.Add(ptr, i+8)) * prime64_2
		acc2 = (acc2 << 31) | (acc2 >> 33)
		acc2 *= prime64_1

		acc3 += *(*uint64)(unsafe.Add(ptr, i+16)) * prime64_2
		acc3 = (acc3 << 31) | (acc3 >> 33)
		acc3 *= prime64_1

		acc4 += *(*uint64)(unsafe.Add(ptr, i+24)) * prime64_2
		acc4 = (acc4 << 31) | (acc4 >> 33)
		acc4 *= prime64_1

		i += 32
	}

	// Merge accumulators if we processed any full blocks
	if i > 0 {
		h ^= acc1 + acc2 + acc3 + acc4
		h = hashMixInline(h)
	}

	// Process remaining 8-byte chunks
	for i+8 <= n {
		h ^= *(*uint64)(unsafe.Add(ptr, i)) * prime64_3
		h = hashMixInline(h)
		i += 8
	}

	// Handle remaining bytes using safe indexing
	if i < n {
		var v uint64
		for j := i; j < n; j++ {
			v = (v << 8) | uint64(s[j])
		}
		h ^= v * prime64_4
		h = hashMixInline(h)
	}

	return h
}

// hashMixBytesInline hashes a byte slice using optimized inline operations.
// This is an inline-optimized version for the cache key generation hot path.
//
//nolint:deadcode // Performance: inlined version for cache key generation hot path
func hashMixBytesInline(h uint64, data []byte) uint64 {
	n := len(data)
	if n == 0 {
		return h
	}

	// Mix length first
	h ^= uint64(n) * prime64_5

	// For very small slices, use safe byte-by-byte processing
	if n < 8 {
		var v uint64
		for j := 0; j < n; j++ {
			v = (v << 8) | uint64(data[j])
		}
		h ^= v * prime64_4
		h = hashMixInline(h)
		return h
	}

	ptr := unsafe.Pointer(unsafe.SliceData(data))
	i := 0

	// Process 32 bytes at a time using 4 accumulators
	var acc1, acc2, acc3, acc4 uint64 = prime64_1, prime64_2, prime64_3, prime64_4

	for i+32 <= n {
		acc1 += *(*uint64)(unsafe.Add(ptr, i)) * prime64_2
		acc1 = (acc1 << 31) | (acc1 >> 33)
		acc1 *= prime64_1

		acc2 += *(*uint64)(unsafe.Add(ptr, i+8)) * prime64_2
		acc2 = (acc2 << 31) | (acc2 >> 33)
		acc2 *= prime64_1

		acc3 += *(*uint64)(unsafe.Add(ptr, i+16)) * prime64_2
		acc3 = (acc3 << 31) | (acc3 >> 33)
		acc3 *= prime64_1

		acc4 += *(*uint64)(unsafe.Add(ptr, i+24)) * prime64_2
		acc4 = (acc4 << 31) | (acc4 >> 33)
		acc4 *= prime64_1

		i += 32
	}

	// Merge accumulators if we processed any full blocks
	if i > 0 {
		h ^= acc1 + acc2 + acc3 + acc4
		h = hashMixInline(h)
	}

	// Process remaining 8-byte chunks
	for i+8 <= n {
		h ^= *(*uint64)(unsafe.Add(ptr, i)) * prime64_3
		h = hashMixInline(h)
		i += 8
	}

	// Handle remaining bytes using safe indexing
	if i < n {
		var v uint64
		for j := i; j < n; j++ {
			v = (v << 8) | uint64(data[j])
		}
		h ^= v * prime64_4
		h = hashMixInline(h)
	}

	return h
}

