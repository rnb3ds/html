package html

import (
	"fmt"
	"strings"

	"github.com/cybergodev/html/internal"
	stdxhtml "golang.org/x/net/html"
)

// ExtractAllLinks extracts all links from HTML bytes with automatic encoding detection.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before extracting links,
// ensuring that link titles and text are properly decoded.
func (p *Processor) ExtractAllLinks(htmlBytes []byte) (links []LinkResource, err error) {
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

	// Validate input
	if len(htmlBytes) == 0 {
		return []LinkResource{}, nil
	}

	config := p.getLinkExtractionConfig()

	if len(htmlBytes) > p.config.MaxInputSize {
		p.stats.errorCount.Add(1)
		return nil, fmt.Errorf("%w: size=%d, max=%d", ErrInputTooLarge, len(htmlBytes), p.config.MaxInputSize)
	}

	startTime := now()

	// Detect encoding and convert to UTF-8
	utf8String, _, convErr := internal.DetectAndConvertToUTF8String(htmlBytes, "")
	if convErr != nil {
		p.stats.errorCount.Add(1)
		return nil, fmt.Errorf("encoding detection failed: %w", convErr)
	}

	// Process with timeout if configured
	if p.config.ProcessingTimeout > 0 {
		links, err = p.extractLinksWithTimeout(utf8String, config)
	} else {
		links, err = p.extractAllLinksFromContent(utf8String, config)
	}

	if err != nil {
		p.stats.errorCount.Add(1)
		return nil, err
	}

	processingTime := since(startTime)
	p.stats.totalProcessTime.Add(int64(processingTime))
	p.stats.totalProcessed.Add(1)

	return links, nil
}

// ExtractAllLinksFromFile extracts all links from an HTML file with automatic encoding detection.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before extracting links.
// Use this when you have a file path instead of raw bytes.
func (p *Processor) ExtractAllLinksFromFile(filePath string) (links []LinkResource, err error) {
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

	return p.ExtractAllLinks(data)
}

// ============================================================================
// Deprecated Package Functions (for backward compatibility)
// ============================================================================

// ExtractAllLinks extracts all links from HTML bytes with automatic encoding detection.
// This is a convenience function that creates a temporary Processor with default settings.
//
// Deprecated: Use Processor.ExtractAllLinks instead for better performance with repeated calls.
func ExtractAllLinks(htmlBytes []byte) ([]LinkResource, error) {
	processor, err := New(DefaultConfig())
	if err != nil {
		return nil, err
	}
	defer processor.Close()
	return processor.ExtractAllLinks(htmlBytes)
}

// ExtractAllLinksFromFile extracts all links from an HTML file with automatic encoding detection.
// This is a convenience function that creates a temporary Processor with default settings.
//
// Deprecated: Use Processor.ExtractAllLinksFromFile instead for better performance with repeated calls.
func ExtractAllLinksFromFile(filePath string) ([]LinkResource, error) {
	processor, err := New(DefaultConfig())
	if err != nil {
		return nil, err
	}
	defer processor.Close()
	return processor.ExtractAllLinksFromFile(filePath)
}

func (p *Processor) extractLinksWithTimeout(htmlContent string, config LinkExtractionConfig) ([]LinkResource, error) {
	return withTimeout(p.config.ProcessingTimeout, func() ([]LinkResource, error) {
		return p.extractAllLinksFromContent(htmlContent, config)
	})
}

func (p *Processor) extractAllLinksFromContent(htmlContent string, config LinkExtractionConfig) ([]LinkResource, error) {
	if strings.TrimSpace(htmlContent) == "" {
		return []LinkResource{}, nil
	}

	doc, err := stdxhtml.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidHTML, err)
	}

	// Validate depth during extraction to avoid duplicate traversal
	if err := p.validateDepthTraversal(doc, 0); err != nil {
		return nil, err
	}

	baseURL := config.BaseURL
	if config.ResolveRelativeURLs && baseURL == "" {
		baseURL = p.detectBaseURL(doc)
	}

	linkMap := make(map[string]LinkResource, linkMapCap)
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
					if (attr.Key == "href" || attr.Key == "src") && internal.IsExternalURL(attr.Val) {
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
	return internal.NormalizeBaseURL(baseURL)
}

func (p *Processor) extractBaseFromURL(url string) string {
	return internal.ExtractBaseFromURL(url)
}

func (p *Processor) isDifferentDomain(baseURL, targetURL string) bool {
	return internal.IsDifferentDomain(baseURL, targetURL)
}

func (p *Processor) resolveURL(baseURL, relativeURL string) string {
	return internal.ResolveURL(baseURL, relativeURL)
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

	if href == "" || !internal.IsValidURL(href) {
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

	if src == "" || !internal.IsValidURL(src) {
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

	if src == "" || !internal.IsValidURL(src) {
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

	if src == "" || !internal.IsValidURL(src) {
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

	if href == "" || !internal.IsValidURL(href) {
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
	if baseURL != "" {
		resolvedURL = p.resolveURL(baseURL, href)
	}

	if title == "" {
		if lastSlash := strings.LastIndex(resolvedURL, "/"); lastSlash >= 0 {
			title = resolvedURL[lastSlash+1:]
		}
	}
	if title == "" {
		title = resourceType
	}

	linkMap[resolvedURL] = LinkResource{
		URL:   resolvedURL,
		Title: title,
		Type:  resourceType,
	}
}

func (p *Processor) extractScriptLinks(n *Node, baseURL string, linkMap map[string]LinkResource) {
	var src string
	for _, attr := range n.Attr {
		if attr.Key == "src" {
			src = attr.Val
			break
		}
	}

	if src == "" || !internal.IsValidURL(src) {
		return
	}

	resolvedURL := src
	if baseURL != "" {
		resolvedURL = p.resolveURL(baseURL, src)
	}

	title := ""
	if lastSlash := strings.LastIndex(resolvedURL, "/"); lastSlash >= 0 {
		title = resolvedURL[lastSlash+1:]
	}
	if title == "" {
		title = "Script"
	}

	linkMap[resolvedURL] = LinkResource{
		URL:   resolvedURL,
		Title: title,
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

	if src == "" || !internal.IsValidURL(src) {
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
