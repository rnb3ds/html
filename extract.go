package html

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
func Extract(htmlBytes []byte, configs ...ExtractConfig) (*Result, error) {
	processor, _ := New()
	defer processor.Close()
	return processor.Extract(htmlBytes, configs...)
}

// ExtractFromFile extracts content from an HTML file with automatic encoding detection.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration (cache, timeout, etc.), use
// Processor.ExtractFromFile instead.
func ExtractFromFile(filePath string, configs ...ExtractConfig) (*Result, error) {
	processor, _ := New()
	defer processor.Close()
	return processor.ExtractFromFile(filePath, configs...)
}

// ExtractText extracts plain text from HTML bytes with automatic encoding detection.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration, use Processor.ExtractText instead.
func ExtractText(htmlBytes []byte, configs ...ExtractConfig) (string, error) {
	result, err := Extract(htmlBytes, configs...)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// Extract extracts content from HTML bytes with automatic encoding detection.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
//
// This is the primary method for HTML content extraction when the source encoding
// may not be UTF-8, such as content from HTTP responses, databases, or files.
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

// ExtractFromFile extracts content from an HTML file with automatic encoding detection.
// Use this when you have a file path instead of raw bytes.
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
	cleanPath := filepathClean(filePath)

	// After cleaning, check if the path contains parent directory references
	// This catches path traversal attempts like "../file", "subdir/../../file", etc.
	if stringsContains(cleanPath, "..") {
		return nil, fmt.Errorf("%w: path traversal detected: %s", ErrInvalidFilePath, cleanPath)
	}

	config := resolveExtractConfig(configs...)

	data, err := readFile(cleanPath)
	if err != nil {
		if osIsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, cleanPath)
		}
		return nil, fmt.Errorf("read file %q: %w", cleanPath, err)
	}

	return p.Extract(data, config)
}

// ExtractText extracts plain text from HTML bytes with automatic encoding detection.
// The method automatically detects character encoding and converts to UTF-8.
// This is a convenience method that returns only the text content without other metadata.
func (p *Processor) ExtractText(htmlBytes []byte, configs ...ExtractConfig) (string, error) {
	result, err := p.Extract(htmlBytes, configs...)
	if err != nil {
		return "", err
	}
	return result.Text, nil
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
