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

	"github.com/cybergodev/html/internal"
	stdxhtml "golang.org/x/net/html"
)

// Extract extracts content from HTML bytes with automatic encoding detection.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration (cache, timeout, etc.), use
// Processor.Extract instead.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
func Extract(htmlBytes []byte) (*Result, error) {
	processor, err := New(DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("create processor: %w", err)
	}
	defer processor.Close()
	return processor.Extract(htmlBytes)
}

// ExtractFromFile extracts content from an HTML file with automatic encoding detection.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration (cache, timeout, etc.), use
// Processor.ExtractFromFile instead.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
func ExtractFromFile(filePath string) (*Result, error) {
	processor, err := New(DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("create processor: %w", err)
	}
	defer processor.Close()
	return processor.ExtractFromFile(filePath)
}

// ExtractText extracts plain text from HTML bytes with automatic encoding detection.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration (cache, timeout, etc.), use
// Processor.ExtractText instead.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
func ExtractText(htmlBytes []byte) (string, error) {
	processor, err := New(DefaultConfig())
	if err != nil {
		return "", fmt.Errorf("create processor: %w", err)
	}
	defer processor.Close()
	return processor.ExtractText(htmlBytes)
}

// ExtractTextFromFile extracts plain text from an HTML file with automatic encoding detection.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration (cache, timeout, etc.), use
// Processor.ExtractTextFromFile instead.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
func ExtractTextFromFile(filePath string) (string, error) {
	processor, err := New(DefaultConfig())
	if err != nil {
		return "", fmt.Errorf("create processor: %w", err)
	}
	defer processor.Close()
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

	if p == nil {
		return nil, ErrProcessorClosed
	}
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	config := p.getExtractConfig()

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
func (p *Processor) ExtractFromFile(filePath string) (result *Result, err error) {
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

	data, readErr := readFile(cleanPath)
	if readErr != nil {
		if osIsNotExist(readErr) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, cleanPath)
		}
		return nil, fmt.Errorf("read file %q: %w", cleanPath, readErr)
	}

	return p.Extract(data)
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
// Uses a simplified xxHash-inspired algorithm optimized for speed.
// Uses multi-point sampling for large documents to better distinguish similar content.
// Optimized to minimize CPU overhead while maintaining good distribution.
func (p *Processor) generateCacheKey(content string, opts ExtractConfig) string {
	// Pack boolean flags into a single uint8 for faster hashing
	// Bits: 0=ExtractArticle, 1=PreserveImages, 2=PreserveLinks, 3=PreserveVideos, 4=PreserveAudios
	flags := uint8(0)
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

	// Initialize hash with prime constants (xxHash-inspired)
	var h uint64 = 0x9E3779B185EBCA87 // Golden ratio fractional bits

	// Mix flags
	h ^= uint64(flags)
	h = hashMix(h)

	// Mix string options with reduced operations
	h = hashMixString(h, opts.InlineImageFormat)
	h = hashMixString(h, opts.TableFormat)

	contentLen := len(content)
	if contentLen <= maxCacheKeySize {
		// Hash the entire content using simplified algorithm
		h = hashMixBytes(h, internal.StringToBytes(content))
	} else {
		// Optimized multi-point sampling for large documents
		// Sample from 3 positions instead of 5 to reduce overhead
		const sampleCount = 3
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
				// First two samples: beginning and middle
				offset := (contentLen * i) / sampleCount
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

	// Return as 8-byte string (reduced from 16-byte for efficiency)
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], h)
	return string(buf[:])
}

// hashMix performs a single mixing step (inlined for performance)
func hashMix(h uint64) uint64 {
	h ^= h >> 23
	h *= 0x2127599BF4325C37
	h ^= h >> 47
	return h
}

// hashMixString hashes a string into the hash state
func hashMixString(h uint64, s string) uint64 {
	// Quick hash for short strings
	n := len(s)
	if n == 0 {
		return h
	}

	// Hash length
	h ^= uint64(n)
	h = hashMix(h)

	// Process 8 bytes at a time
	i := 0
	for i+7 < n {
		v := binary.LittleEndian.Uint64(internal.StringToBytes(s[i : i+8]))
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
func hashMixBytes(h uint64, data []byte) uint64 {
	n := len(data)
	if n == 0 {
		return h
	}

	// Process 8 bytes at a time
	i := 0
	for i+7 < n {
		v := binary.LittleEndian.Uint64(data[i : i+8])
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
