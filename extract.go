package html

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	htmlstd "html"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/cybergodev/html/internal"
	stdxhtml "golang.org/x/net/html"
)

// stringToBytes converts a string to a byte slice without memory allocation.
// The returned slice shares memory with the original string.
//
// SAFETY: The returned slice MUST NOT be modified. Go strings are immutable,
// and modifying the returned slice would violate this immutability, potentially
// causing undefined behavior in other code holding references to the string.
//
// LIFETIME: The returned slice is valid as long as the original string is not
// garbage collected. In practice, this means the slice should only be used
// within the same scope as the string, and should not be stored beyond the
// string's lifetime.
//
// PERFORMANCE: This function is used to avoid allocations when passing strings
// to functions that accept []byte (e.g., hash.Write). For short-lived operations
// where the string is guaranteed to remain in scope, this is safe and efficient.
func stringToBytes(s string) []byte {
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// Extract extracts content from HTML bytes with automatic encoding detection.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration (cache, timeout, etc.), use
// Processor.Extract instead.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
func Extract(htmlBytes []byte, configs ...ExtractConfig) (*Result, error) {
	processor, err := New()
	if err != nil {
		return nil, fmt.Errorf("create processor: %w", err)
	}
	defer processor.Close()
	return processor.Extract(htmlBytes, configs...)
}

// ExtractFromFile extracts content from an HTML file with automatic encoding detection.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration (cache, timeout, etc.), use
// Processor.ExtractFromFile instead.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
func ExtractFromFile(filePath string, configs ...ExtractConfig) (*Result, error) {
	processor, err := New()
	if err != nil {
		return nil, fmt.Errorf("create processor: %w", err)
	}
	defer processor.Close()
	return processor.ExtractFromFile(filePath, configs...)
}

// ExtractText extracts plain text from HTML bytes with automatic encoding detection.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration (cache, timeout, etc.), use
// Processor.ExtractText instead.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
func ExtractText(htmlBytes []byte, configs ...ExtractConfig) (string, error) {
	processor, err := New()
	if err != nil {
		return "", fmt.Errorf("create processor: %w", err)
	}
	defer processor.Close()
	return processor.ExtractText(htmlBytes, configs...)
}

// ExtractTextFromFile extracts plain text from an HTML file with automatic encoding detection.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration (cache, timeout, etc.), use
// Processor.ExtractTextFromFile instead.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
func ExtractTextFromFile(filePath string, configs ...ExtractConfig) (string, error) {
	processor, err := New()
	if err != nil {
		return "", fmt.Errorf("create processor: %w", err)
	}
	defer processor.Close()
	return processor.ExtractTextFromFile(filePath, configs...)
}

// Extract extracts content from HTML bytes with automatic encoding detection.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
//
// This is the primary method for HTML content extraction when the source encoding
// may not be UTF-8, such as content from HTTP responses, databases, or files.
func (p *Processor) Extract(htmlBytes []byte, configs ...ExtractConfig) (result *Result, err error) {
	// Defense-in-depth: recover from unexpected panics
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrInternalPanic, r)
		}
	}()

	if p == nil {
		return nil, ErrProcessorClosed
	}
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	config := resolveExtractConfig(configs...)

	if len(htmlBytes) > p.config.MaxInputSize {
		p.audit.RecordInputViolation(len(htmlBytes), p.config.MaxInputSize, "input_too_large")
		p.stats.errorCount.Add(1)
		return nil, fmt.Errorf("%w: size=%d, max=%d", ErrInputTooLarge, len(htmlBytes), p.config.MaxInputSize)
	}

	startTime := time.Now()

	// Detect encoding and convert to UTF-8
	utf8String, detectedEncoding, convErr := internal.DetectAndConvertToUTF8String(htmlBytes, config.Encoding)
	if convErr != nil {
		p.audit.RecordEncodingIssue(config.Encoding, convErr.Error())
		p.stats.errorCount.Add(1)
		return nil, fmt.Errorf("encoding detection failed: %w", convErr)
	}
	_ = detectedEncoding // Used for audit logging if needed

	// Use the converted UTF-8 string for cache key
	cacheKey := p.generateCacheKey(utf8String, config)
	if cached := p.cache.Get(cacheKey); cached != nil {
		p.stats.cacheHits.Add(1)
		p.stats.totalProcessed.Add(1)
		if cachedResult, ok := cached.(*Result); ok {
			return cachedResult, nil
		}
	}
	p.stats.cacheMisses.Add(1)

	// Process the content
	if p.config.ProcessingTimeout > 0 {
		result, err = p.processWithTimeout(utf8String, config)
	} else {
		result, err = p.processContent(utf8String, config)
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

	if p.config.MaxCacheEntries > 0 {
		p.cache.Set(cacheKey, result)
	}

	return result, nil
}

// ExtractFromFile extracts content from an HTML file with automatic encoding detection.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
// Use this when you have a file path instead of raw bytes.
func (p *Processor) ExtractFromFile(filePath string, configs ...ExtractConfig) (result *Result, err error) {
	// Defense-in-depth: recover from unexpected panics
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrInternalPanic, r)
		}
	}()

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
	cleanPath := filepathClean(filePath)

	// After cleaning, check if the path contains parent directory references
	// This catches path traversal attempts like "../file", "subdir/../../file", etc.
	if stringsContains(cleanPath, "..") {
		p.audit.RecordPathTraversal(filePath)
		return nil, fmt.Errorf("%w: path traversal detected: %s", ErrInvalidFilePath, cleanPath)
	}

	config := resolveExtractConfig(configs...)

	data, readErr := readFile(cleanPath)
	if readErr != nil {
		if osIsNotExist(readErr) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, cleanPath)
		}
		return nil, fmt.Errorf("read file %q: %w", cleanPath, readErr)
	}

	return p.Extract(data, config)
}

// ExtractText extracts plain text from HTML bytes with automatic encoding detection.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
// This is a convenience method that returns only the text content without other metadata.
func (p *Processor) ExtractText(htmlBytes []byte, configs ...ExtractConfig) (string, error) {
	result, err := p.Extract(htmlBytes, configs...)
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
func (p *Processor) ExtractTextFromFile(filePath string, configs ...ExtractConfig) (string, error) {
	result, err := p.ExtractFromFile(filePath, configs...)
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
func withTimeout[T any](timeout time.Duration, fn func() (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type result struct {
		res T
		err error
	}

	resultChan := make(chan result, 1)

	// Launch the operation in a goroutine
	go func() {
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
		return zero, ErrProcessingTimeout
	}
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

func (p *Processor) processContent(htmlContent string, opts ExtractConfig) (*Result, error) {
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

	return p.extractFromDocument(doc, originalHTML, opts)
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

		sb := internal.GetBuilder()
		sb.Grow(initialTextSize)
		imageCounter := 0
		internal.ExtractTextWithStructureAndImages(contentNode, sb, 0, &imageCounter, opts.TableFormat)
		textWithPlaceholders := internal.CleanText(sb.String(), nil)
		internal.PutBuilder(sb)
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
	// Pre-allocate map with initial capacity to reduce resizing
	candidates := make(map[*Node]int, initialMapCap)
	internal.WalkNodes(doc, func(n *Node) bool {
		if n.Type == ElementNode {
			var score int
			if p.scorer != nil {
				score = p.scorer.Score(n)
			} else {
				score = internal.ScoreContentNode(n)
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
	internal.ExtractTextWithStructureAndImages(node, sb, 0, nil, tableFormat)
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

// generateCacheKey creates a hash for cache key generation.
// Uses a custom wyhash-inspired algorithm that is faster than FNV-128a.
// Uses multi-point sampling for large documents to better distinguish similar content.
// Optimized to avoid heap allocations.
func (p *Processor) generateCacheKey(content string, opts ExtractConfig) string {
	// Initialize hash with config flags using a fast mixing function
	var h1, h2 uint64 = 0x6a09e667f3bcc908, 0xbb67ae8584caa73b // wyhash constants

	// Mix config flags
	h1 = wyhashMix(h1, uint64(boolToUint(opts.ExtractArticle)))
	h1 = wyhashMix(h1, uint64(boolToUint(opts.PreserveImages)))
	h1 = wyhashMix(h1, uint64(boolToUint(opts.PreserveLinks)))
	h2 = wyhashMix(h2, uint64(boolToUint(opts.PreserveVideos)))
	h2 = wyhashMix(h2, uint64(boolToUint(opts.PreserveAudios)))

	// Mix string options
	h1 = wyhashString(h1, opts.InlineImageFormat)
	h2 = wyhashString(h2, opts.TableFormat)

	contentLen := len(content)
	if contentLen <= maxCacheKeySize {
		// Hash the entire content
		h1, h2 = wyhashBytes(h1, h2, stringToBytes(content))
	} else {
		// Optimized multi-point sampling for large documents
		// Sample from 5 positions: beginning, 25%, 50%, 75%, and end
		const sampleCount = 5
		sampleSize := cacheKeySample / sampleCount
		if sampleSize < 512 {
			sampleSize = 512
		}

		for i := 0; i < sampleCount; i++ {
			var start, end int
			if i == sampleCount-1 {
				end = contentLen
				start = contentLen - sampleSize
				if start < 0 {
					start = 0
				}
			} else {
				offset := (contentLen * i) / sampleCount
				start = offset
				end = start + sampleSize
				if end > contentLen {
					end = contentLen
				}
			}

			if start < end {
				h1, h2 = wyhashBytes(h1, h2, stringToBytes(content[start:end]))
			}
		}

		// Mix content length
		h1 = wyhashMix(h1, uint64(contentLen))
	}

	// Final mix
	h1 ^= h2
	h2 ^= h1
	h1 ^= h2

	// Return as 16-byte string
	var buf [16]byte
	binary.LittleEndian.PutUint64(buf[0:8], h1)
	binary.LittleEndian.PutUint64(buf[8:16], h2)
	return string(buf[:])
}

// boolToUint converts bool to uint64 (1 for true, 0 for false)
func boolToUint(b bool) uint {
	if b {
		return 1
	}
	return 0
}

// wyhashMix performs a single mixing step inspired by wyhash
func wyhashMix(a, b uint64) uint64 {
	a ^= b
	a *= 0x6a09e667f3bcc908
	a = (a ^ (a >> 31)) * 0xbb67ae8584caa73b
	return a ^ (a >> 31)
}

// wyhashString hashes a string into the hash state
func wyhashString(h uint64, s string) uint64 {
	for i := 0; i < len(s); i++ {
		h = wyhashMix(h, uint64(s[i]))
	}
	return h
}

// wyhashBytes hashes a byte slice into the hash state
func wyhashBytes(h1, h2 uint64, data []byte) (uint64, uint64) {
	// Process 8 bytes at a time
	n := len(data)
	i := 0

	for i+7 < n {
		v := binary.LittleEndian.Uint64(data[i : i+8])
		h1 = wyhashMix(h1, v)
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
		h2 = wyhashMix(h2, v)
	}

	return h1, h2
}

// boolToString returns "1" for true and "0" for false
func boolToString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
