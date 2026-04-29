package html

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cybergodev/html/internal"
	stdxhtml "golang.org/x/net/html"
)

// ExtractAllLinks extracts all links from HTML bytes with automatic encoding detection.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before extracting links,
// ensuring that link titles and text are properly decoded.
func (p *Processor) ExtractAllLinks(htmlBytes []byte) ([]LinkResource, error) {
	return recoverLinks(func() ([]LinkResource, error) {
		// Validate input
		if len(htmlBytes) == 0 {
			return []LinkResource{}, nil
		}

		// Validate processor state and input size
		if err := p.validateInput(htmlBytes); err != nil {
			return nil, err
		}

		startTime := time.Now()

		// Detect encoding and convert to UTF-8 using configured encoding
		utf8String, err := p.detectEncoding(htmlBytes)
		if err != nil {
			return nil, err
		}

		// Process with timeout if configured
		var links []LinkResource
		if p.config.ProcessingTimeout > 0 {
			links, err = p.extractLinksWithTimeout(utf8String)
		} else {
			links, err = p.extractAllLinksFromContent(utf8String)
		}

		if err != nil {
			p.stats.errorCount.Add(1)
			return nil, err
		}

		processingTime := time.Since(startTime)
		p.stats.totalProcessTime.Add(int64(processingTime))
		p.stats.totalProcessed.Add(1)

		return links, nil
	})
}

// ExtractAllLinksFromFile extracts all links from an HTML file with automatic encoding detection.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before extracting links.
// Use this when you have a file path instead of raw bytes.
func (p *Processor) ExtractAllLinksFromFile(filePath string) ([]LinkResource, error) {
	return recoverLinks(func() ([]LinkResource, error) {
		if p == nil {
			return nil, ErrProcessorClosed
		}
		if p.closed.Load() {
			return nil, ErrProcessorClosed
		}

		data, err := p.validateAndReadFile(filePath)
		if err != nil {
			return nil, err
		}

		return p.ExtractAllLinks(data)
	})
}

// ExtractAllLinksWithContext extracts all links from HTML bytes with context support for cancellation.
// This method provides cooperative cancellation, allowing long-running extractions to be
// interrupted when the context is cancelled.
func (p *Processor) ExtractAllLinksWithContext(ctx context.Context, htmlBytes []byte) ([]LinkResource, error) {
	return recoverLinks(func() ([]LinkResource, error) {
		// Early cancellation check
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Validate input
		if len(htmlBytes) == 0 {
			return []LinkResource{}, nil
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

		// Detect encoding using configured encoding setting
		utf8String, err := p.detectEncoding(htmlBytes)
		if err != nil {
			return nil, err
		}

		// Check cancellation before processing
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Process with timeout if configured, using withTimeout for goroutine tracking
		var links []LinkResource
		if p.config.ProcessingTimeout > 0 {
			links, err = withTimeout(p.config.ProcessingTimeout, func() ([]LinkResource, error) {
				return p.extractAllLinksFromContent(utf8String)
			})
		} else {
			links, err = p.extractAllLinksFromContent(utf8String)
		}

		if err != nil {
			p.stats.errorCount.Add(1)
			return nil, err
		}

		processingTime := time.Since(startTime)
		p.stats.totalProcessTime.Add(int64(processingTime))
		p.stats.totalProcessed.Add(1)

		return links, nil
	})
}

// ExtractAllLinksFromFileWithContext extracts all links from an HTML file with context support.
func (p *Processor) ExtractAllLinksFromFileWithContext(ctx context.Context, filePath string) ([]LinkResource, error) {
	return recoverLinks(func() ([]LinkResource, error) {
		// Early cancellation check
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if p == nil {
			return nil, ErrProcessorClosed
		}
		if p.closed.Load() {
			return nil, ErrProcessorClosed
		}

		data, err := p.validateAndReadFile(filePath)
		if err != nil {
			return nil, err
		}

		return p.ExtractAllLinksWithContext(ctx, data)
	})
}

// ============================================================================
// Package-level Convenience Functions
// ============================================================================

// ExtractAllLinks extracts all links from HTML bytes with automatic encoding detection.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// It automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before extracting links,
// ensuring that link titles and text are properly decoded.
//
// An optional Config can be provided to customize link extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractAllLinks(htmlBytes []byte, cfg ...Config) ([]LinkResource, error) {
	c, pooled, err := resolveConfig(cfg...)
	if err != nil {
		return nil, err
	}
	return withProcessor(pooled, c, func(p *Processor) ([]LinkResource, error) {
		return p.ExtractAllLinks(htmlBytes)
	})
}

// ExtractAllLinksFromFile extracts all links from an HTML file with automatic encoding detection.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// It automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before extracting links.
//
// An optional Config can be provided to customize link extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractAllLinksFromFile(filePath string, cfg ...Config) ([]LinkResource, error) {
	c, pooled, err := resolveConfig(cfg...)
	if err != nil {
		return nil, err
	}
	return withProcessor(pooled, c, func(p *Processor) ([]LinkResource, error) {
		return p.ExtractAllLinksFromFile(filePath)
	})
}

// ExtractAllLinksWithContext extracts all links from HTML bytes with context support for cancellation.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// It automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before extracting links.
//
// An optional Config can be provided to customize link extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractAllLinksWithContext(ctx context.Context, htmlBytes []byte, cfg ...Config) ([]LinkResource, error) {
	c, pooled, err := resolveConfig(cfg...)
	if err != nil {
		return nil, err
	}
	return withProcessor(pooled, c, func(p *Processor) ([]LinkResource, error) {
		return p.ExtractAllLinksWithContext(ctx, htmlBytes)
	})
}

// ExtractAllLinksFromFileWithContext extracts all links from an HTML file with context support for cancellation.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// It automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before extracting links.
//
// An optional Config can be provided to customize link extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractAllLinksFromFileWithContext(ctx context.Context, filePath string, cfg ...Config) ([]LinkResource, error) {
	c, pooled, err := resolveConfig(cfg...)
	if err != nil {
		return nil, err
	}
	return withProcessor(pooled, c, func(p *Processor) ([]LinkResource, error) {
		return p.ExtractAllLinksFromFileWithContext(ctx, filePath)
	})
}

func (p *Processor) extractLinksWithTimeout(htmlContent string) ([]LinkResource, error) {
	return withTimeout(p.config.ProcessingTimeout, func() ([]LinkResource, error) {
		return p.extractAllLinksFromContent(htmlContent)
	})
}

func (p *Processor) extractAllLinksFromContent(htmlContent string) ([]LinkResource, error) {
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

	baseURL := p.config.BaseURL
	if p.config.ResolveRelativeURLs && baseURL == "" {
		baseURL = p.detectBaseURL(doc)
	}

	linkMap := make(map[string]LinkResource, linkMapCap)
	p.extractLinksFromDocument(doc, baseURL, linkMap)

	links := make([]LinkResource, 0, len(linkMap))
	for _, link := range linkMap {
		links = append(links, link)
	}

	return links, nil
}

// detectBaseURL attempts to detect base URL from HTML document.
func (p *Processor) detectBaseURL(doc *stdxhtml.Node) string {
	if baseNode := internal.FindElementByTag(doc, "base"); baseNode != nil {
		for _, attr := range baseNode.Attr {
			if attr.Key == "href" && attr.Val != "" {
				return internal.NormalizeBaseURL(attr.Val)
			}
		}
	}

	var canonicalURL, canonicalLink, firstAbsoluteURL string
	internal.WalkNodes(doc, func(n *stdxhtml.Node) bool {
		if n.Type != stdxhtml.ElementNode {
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
						if base := internal.ExtractBaseFromURL(attr.Val); base != "" {
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
		return internal.NormalizeBaseURL(canonicalURL)
	}
	if canonicalLink != "" {
		return internal.NormalizeBaseURL(canonicalLink)
	}
	return firstAbsoluteURL
}

func (p *Processor) extractLinksFromDocument(doc *stdxhtml.Node, baseURL string, linkMap map[string]LinkResource) {
	internal.WalkNodes(doc, func(n *stdxhtml.Node) bool {
		if n.Type != stdxhtml.ElementNode {
			return true
		}

		switch n.Data {
		case "a":
			if p.config.IncludeContentLinks || p.config.IncludeExternalLinks {
				p.extractContentLinks(n, baseURL, linkMap)
			}
		case "img":
			if p.config.IncludeImages {
				p.extractImageLinks(n, baseURL, linkMap)
			}
		case "video":
			if p.config.IncludeVideos {
				p.extractMediaLink(n, baseURL, linkMap, "video")
			}
		case "audio":
			if p.config.IncludeAudios {
				p.extractMediaLink(n, baseURL, linkMap, "audio")
			}
		case "source":
			if p.config.IncludeVideos || p.config.IncludeAudios {
				p.extractSourceLinks(n, baseURL, linkMap)
			}
		case "link":
			p.extractLinkTagLinks(n, baseURL, linkMap)
		case "script":
			if p.config.IncludeJS {
				p.extractScriptLinks(n, baseURL, linkMap)
			}
		case "iframe", "embed", "object":
			if p.config.IncludeVideos {
				p.extractEmbedLinks(n, baseURL, linkMap)
			}
		}
		return true
	})
}

func (p *Processor) extractContentLinks(n *stdxhtml.Node, baseURL string, linkMap map[string]LinkResource) {
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
	if p.config.ResolveRelativeURLs && baseURL != "" {
		resolvedURL = internal.ResolveURL(baseURL, href)
	}

	isExternal := isExternalOriginal
	if !isExternalOriginal && baseURL != "" {
		isExternal = internal.IsDifferentDomain(baseURL, resolvedURL)
	}

	if isExternal && !p.config.IncludeExternalLinks {
		return
	}
	if !isExternal && !p.config.IncludeContentLinks {
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

func (p *Processor) extractImageLinks(n *stdxhtml.Node, baseURL string, linkMap map[string]LinkResource) {
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
		resolvedURL = internal.ResolveURL(baseURL, src)
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

func (p *Processor) extractMediaLink(n *stdxhtml.Node, baseURL string, linkMap map[string]LinkResource, mediaType string) {
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
		resolvedURL = internal.ResolveURL(baseURL, src)
	}

	displayName := title
	if displayName == "" {
		if lastSlash := strings.LastIndex(resolvedURL, "/"); lastSlash >= 0 {
			displayName = resolvedURL[lastSlash+1:]
		}
		if displayName == "" {
			if mediaType != "" {
				displayName = strings.ToUpper(mediaType[:1]) + mediaType[1:]
			} else {
				displayName = "Media"
			}
		}
	}

	linkMap[resolvedURL] = LinkResource{
		URL:   resolvedURL,
		Title: displayName,
		Type:  mediaType,
	}
}

func (p *Processor) extractSourceLinks(n *stdxhtml.Node, baseURL string, linkMap map[string]LinkResource) {
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
		resolvedURL = internal.ResolveURL(baseURL, src)
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

func (p *Processor) extractLinkTagLinks(n *stdxhtml.Node, baseURL string, linkMap map[string]LinkResource) {
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
		if p.config.IncludeCSS {
			resourceType = "css"
			include = true
		}
	case "icon", "shortcut icon", "apple-touch-icon", "apple-touch-icon-precomposed":
		if p.config.IncludeIcons {
			resourceType = "icon"
			include = true
		}
	case "preload", "prefetch", "dns-prefetch", "preconnect":
		for _, attr := range n.Attr {
			if attr.Key == "as" {
				switch attr.Val {
				case "style":
					if p.config.IncludeCSS {
						resourceType = "css"
						include = true
					}
				case "script":
					if p.config.IncludeJS {
						resourceType = "js"
						include = true
					}
				case "image":
					if p.config.IncludeImages {
						resourceType = "image"
						include = true
					}
				case "video":
					if p.config.IncludeVideos {
						resourceType = "video"
						include = true
					}
				case "audio":
					if p.config.IncludeAudios {
						resourceType = "audio"
						include = true
					}
				}
				break
			}
		}
	default:
		if strings.Contains(linkType, "css") && p.config.IncludeCSS {
			resourceType = "css"
			include = true
		} else if strings.Contains(linkType, "javascript") && p.config.IncludeJS {
			resourceType = "js"
			include = true
		}
	}

	if !include {
		return
	}

	resolvedURL := href
	if baseURL != "" {
		resolvedURL = internal.ResolveURL(baseURL, href)
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

func (p *Processor) extractScriptLinks(n *stdxhtml.Node, baseURL string, linkMap map[string]LinkResource) {
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
		resolvedURL = internal.ResolveURL(baseURL, src)
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

func (p *Processor) extractEmbedLinks(n *stdxhtml.Node, baseURL string, linkMap map[string]LinkResource) {
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
		resolvedURL = internal.ResolveURL(baseURL, src)
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

// GroupLinksByType groups links by their type.
// This is a convenience function for organizing extracted links.
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
