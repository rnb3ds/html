package html

import (
	"context"
	"errors"
	"fmt"
	htmlstd "html"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cybergodev/html/internal"
	stdxhtml "golang.org/x/net/html"
)

// linkPlaceholderRegex matches [LINK:N]text[/LINK] pattern for inline link formatting.
// Pre-compiled at package initialization for performance.
var linkPlaceholderRegex = regexp.MustCompile(`\[LINK:(\d+)\]([^\[]*?)\[/LINK\]`)

// markdownEscapeReplacer escapes characters that could break Markdown link/image syntax.
var markdownEscapeReplacer = strings.NewReplacer(
	`\`, `\\`,
	`[`, `\[`,
	`]`, `\]`,
)

// containsASCIIFold reports whether substr is contained in s, case-insensitively (ASCII only).
func containsASCIIFold(s, substr string) bool {
	substrLen := len(substr)
	sLen := len(s)
	if substrLen > sLen {
		return false
	}
	for i := 0; i <= sLen-substrLen; i++ {
		match := true
		for j := 0; j < substrLen; j++ {
			c := s[i+j]
			sc := substr[j]
			if c >= 'A' && c <= 'Z' {
				c += 32
			}
			if c != sc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// escapeMarkdownText escapes characters that could break Markdown link/image syntax.
// Escapes ], [, and \ to prevent injection of arbitrary Markdown content.
func escapeMarkdownText(s string) string {
	return markdownEscapeReplacer.Replace(s)
}

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

// recoverPanic wraps a function with defense-in-depth panic recovery.
func recoverPanic[T any](fn func() (T, error)) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrInternalPanic, r)
		}
	}()
	return fn()
}

func recoverResult(fn func() (*Result, error)) (*Result, error) { return recoverPanic(fn) }
func recoverLinks(fn func() ([]LinkResource, error)) ([]LinkResource, error) {
	return recoverPanic(fn)
}
func recoverString(fn func() (string, error)) (string, error) { return recoverPanic(fn) }
func recoverBytes(fn func() ([]byte, error)) ([]byte, error)  { return recoverPanic(fn) }

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
// It automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
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
	c, pooled, err := resolveConfig(cfg...)
	if err != nil {
		return nil, err
	}
	return withProcessor(pooled, c, func(p *Processor) (*Result, error) {
		return p.Extract(htmlBytes)
	})
}

// ExtractFromFile extracts content from an HTML file with automatic encoding detection.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// It automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
//
// An optional Config can be provided to customize extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractFromFile(filePath string, cfg ...Config) (*Result, error) {
	c, pooled, err := resolveConfig(cfg...)
	if err != nil {
		return nil, err
	}
	return withProcessor(pooled, c, func(p *Processor) (*Result, error) {
		return p.ExtractFromFile(filePath)
	})
}

// ExtractText extracts plain text from HTML bytes with automatic encoding detection.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// For better performance with multiple extractions, create a Processor directly.
//
// It automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
//
// An optional Config can be provided to customize extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractText(htmlBytes []byte, cfg ...Config) (string, error) {
	c, pooled, err := resolveConfig(cfg...)
	if err != nil {
		return "", err
	}
	return withProcessor(pooled, c, func(p *Processor) (string, error) {
		return p.ExtractText(htmlBytes)
	})
}

// ExtractTextFromFile extracts plain text from an HTML file with automatic encoding detection.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// It automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
//
// An optional Config can be provided to customize extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractTextFromFile(filePath string, cfg ...Config) (string, error) {
	c, pooled, err := resolveConfig(cfg...)
	if err != nil {
		return "", err
	}
	return withProcessor(pooled, c, func(p *Processor) (string, error) {
		return p.ExtractTextFromFile(filePath)
	})
}

// ExtractTextWithContext extracts plain text from HTML bytes with context support for cancellation.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// An optional Config can be provided to customize extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractTextWithContext(ctx context.Context, htmlBytes []byte, cfg ...Config) (string, error) {
	c, pooled, err := resolveConfig(cfg...)
	if err != nil {
		return "", err
	}
	return withProcessor(pooled, c, func(p *Processor) (string, error) {
		return p.ExtractTextWithContext(ctx, htmlBytes)
	})
}

// ExtractTextFromFileWithContext extracts plain text from an HTML file with context support for cancellation.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// An optional Config can be provided to customize extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractTextFromFileWithContext(ctx context.Context, filePath string, cfg ...Config) (string, error) {
	c, pooled, err := resolveConfig(cfg...)
	if err != nil {
		return "", err
	}
	return withProcessor(pooled, c, func(p *Processor) (string, error) {
		return p.ExtractTextFromFileWithContext(ctx, filePath)
	})
}

// ExtractWithContext extracts content from HTML bytes with context support for cancellation.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// An optional Config can be provided to customize extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractWithContext(ctx context.Context, htmlBytes []byte, cfg ...Config) (*Result, error) {
	c, pooled, err := resolveConfig(cfg...)
	if err != nil {
		return nil, err
	}
	return withProcessor(pooled, c, func(p *Processor) (*Result, error) {
		return p.ExtractWithContext(ctx, htmlBytes)
	})
}

// ExtractFromFileWithContext extracts content from an HTML file with context support for cancellation.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// An optional Config can be provided to customize extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractFromFileWithContext(ctx context.Context, filePath string, cfg ...Config) (*Result, error) {
	c, pooled, err := resolveConfig(cfg...)
	if err != nil {
		return nil, err
	}
	return withProcessor(pooled, c, func(p *Processor) (*Result, error) {
		return p.ExtractFromFileWithContext(ctx, filePath)
	})
}

// Extract extracts content from HTML bytes with automatic encoding detection.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
//
// This is the primary method for HTML content extraction when the source encoding
// may not be UTF-8, such as content from HTTP responses, databases, or files.
func (p *Processor) Extract(htmlBytes []byte) (*Result, error) {
	return recoverResult(func() (*Result, error) {
		return p.extractCoreWithContext(context.Background(), htmlBytes)
	})
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
func (p *Processor) ExtractWithContext(ctx context.Context, htmlBytes []byte) (*Result, error) {
	return recoverResult(func() (*Result, error) {
		return p.extractCoreWithContext(ctx, htmlBytes)
	})
}

func (p *Processor) extractCoreWithContext(ctx context.Context, htmlBytes []byte) (*Result, error) {
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

	// Process content with optional timeout and context support
	var result *Result
	if p.config.ProcessingTimeout > 0 {
		result, err = withTimeout(p.config.ProcessingTimeout, func() (*Result, error) {
			return p.processContentWithContext(ctx, utf8String)
		})
	} else {
		result, err = p.processContentWithContext(ctx, utf8String)
	}

	if err != nil {
		p.stats.errorCount.Add(1)
		if p.audit != nil {
			if errors.Is(err, ErrProcessingTimeout) {
				p.audit.RecordTimeout(p.config.ProcessingTimeout)
			} else if errors.Is(err, ErrMaxDepthExceeded) {
				p.audit.RecordDepthViolation(p.config.MaxDepth+1, p.config.MaxDepth)
			}
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

	if p.isBlankContent(htmlContent) {
		return &Result{}, nil
	}

	originalHTML := htmlContent

	// Check context before sanitization
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	htmlContent = p.sanitizeContent(htmlContent)

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
func (p *Processor) ExtractFromFile(filePath string) (*Result, error) {
	return recoverResult(func() (*Result, error) {
		// Validate processor state (no input size check for file paths)
		if p == nil || p.closed.Load() {
			return nil, ErrProcessorClosed
		}

		data, err := p.validateAndReadFile(filePath)
		if err != nil {
			return nil, err
		}

		return p.Extract(data)
	})
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
func (p *Processor) ExtractFromFileWithContext(ctx context.Context, filePath string) (*Result, error) {
	return recoverResult(func() (*Result, error) {
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
	})
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

// ExtractTextWithContext extracts plain text from HTML bytes with context support for cancellation.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
// This is a convenience method that returns only the text content without other metadata.
func (p *Processor) ExtractTextWithContext(ctx context.Context, htmlBytes []byte) (string, error) {
	result, err := p.ExtractWithContext(ctx, htmlBytes)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// ExtractTextFromFileWithContext extracts plain text from an HTML file with context support for cancellation.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
// Use this when you have a file path instead of raw bytes.
// This is a convenience method that returns only the text content without other metadata.
func (p *Processor) ExtractTextFromFileWithContext(ctx context.Context, filePath string) (string, error) {
	result, err := p.ExtractFromFileWithContext(ctx, filePath)
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
	// Atomically check-and-add to prevent TOCTOU race under heavy concurrency.
	// This prevents resource exhaustion when processing many documents with timeouts.
	for {
		current := activeTimeoutGoroutines.Load()
		if current >= maxTimeoutGoroutines {
			var zero T
			return zero, fmt.Errorf("%w: too many concurrent operations (%d active)", ErrProcessingTimeout, current)
		}
		if activeTimeoutGoroutines.CompareAndSwap(current, current+1) {
			break
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type result struct {
		res T
		err error
	}

	resultChan := make(chan result, 1)

	// Launch the operation in a goroutine
	// (counter already incremented by CAS loop above)
	go func() {
		// Ensure we decrement the counter when done
		defer activeTimeoutGoroutines.Add(-1)

		// Recover from panics inside fn() to prevent crashing the entire process.
		// Without this, a panic in the spawned goroutine bypasses the caller's
		// recover() (which only catches panics in the same goroutine).
		res, err := func() (res T, err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("%w: %v", ErrInternalPanic, r)
				}
			}()
			return fn()
		}()
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

// isBlankContent checks if content is empty or whitespace-only without allocating.
func (p *Processor) isBlankContent(content string) bool {
	n := len(content)
	if n == 0 {
		return true
	}
	for i := 0; i < n; i++ {
		c := content[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return false
		}
	}
	return true
}

// sanitizeContent applies HTML sanitization if enabled in the config.
func (p *Processor) sanitizeContent(htmlContent string) string {
	if !p.config.EnableSanitization {
		return htmlContent
	}
	if p.audit != nil && p.config.Audit.Enabled {
		return internal.SanitizeHTMLWithAudit(htmlContent, p.auditAdapter)
	}
	return internal.SanitizeHTML(htmlContent)
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

	imageFormat := p.imageFormat
	linkFormat := p.linkFormat

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
			result.Images = p.extractImagesWithPosition(contentNode)
		}
		if p.config.PreserveLinks {
			result.Links = p.extractLinksWithPosition(contentNode)
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

	internal.WalkNodes(doc, func(n *stdxhtml.Node) bool {
		if n.Type == stdxhtml.ElementNode {
			if score := p.scorer.Score(n); score > 0 {
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

	switch format {
	case "markdown":
		// Build replacements and use direct string replacement
		replacements := make([]string, 0, len(images)*2)
		for i := range images {
			if images[i].Position == 0 {
				continue
			}
			placeholder := "[IMAGE:" + strconv.Itoa(images[i].Position) + "]"
			altText := images[i].Alt
			if altText == "" {
				altText = "Image " + strconv.Itoa(images[i].Position)
			}
			markdown := "![" + escapeMarkdownText(altText) + "](" + images[i].URL + ")"
			replacements = append(replacements, placeholder, markdown)
		}
		if len(replacements) > 0 {
			replacer := strings.NewReplacer(replacements...)
			return replacer.Replace(textWithPlaceholders)
		}

	case "html":
		// Build replacements with pooled builder for HTML tags
		htmlImg := internal.GetBuilder()
		replacements := make([]string, 0, len(images)*2)
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
		if len(replacements) > 0 {
			replacer := strings.NewReplacer(replacements...)
			return replacer.Replace(textWithPlaceholders)
		}
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
		submatches := linkPlaceholderRegex.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}

		linkText := submatches[2]

		position := 0
		if _, err := fmt.Sscanf(submatches[1], "%d", &position); err != nil {
			return linkText
		}

		link, ok := linkMap[position]
		if !ok {
			return linkText
		}

		if linkText == "" {
			linkText = "Link " + strconv.Itoa(position)
		}

		switch format {
		case "markdown":
			return "[" + escapeMarkdownText(linkText) + "](" + link.URL + ")"
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
			// Check for nofollow in rel attribute (case-insensitive, no allocation)
			if containsASCIIFold(attr.Val, "nofollow") {
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
	n := len(text)
	count := 0
	inWord := false
	for i := 0; i < n; i++ {
		c := text[i]
		isSpace := c == ' ' || c == '\t' || c == '\n' || c == '\r'
		if isSpace {
			inWord = false
		} else if !inWord {
			inWord = true
			count++
		}
	}
	return count
}

func (p *Processor) calculateReadingTime(wordCount int) time.Duration {
	if wordCount == 0 {
		return 0
	}
	minutes := float64(wordCount) / wordsPerMinute
	return time.Duration(minutes * float64(time.Minute))
}
