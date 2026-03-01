package html

import (
	"strings"

	"github.com/cybergodev/html/internal"
)

func (p *Processor) extractVideos(node *Node, htmlContent string) []VideoInfo {
	videos := make([]VideoInfo, 0, initialSliceCap)
	seen := make(map[string]bool, initialMapCap)

	// First, extract from the HTML content directly for iframe/embed/object tags
	// These may be removed by sanitization, so we parse them from raw HTML first
	if len(htmlContent) > 0 && len(htmlContent) <= maxHTMLForRegex*10 {
		// Parse iframe tags
		iframeMatches := p.extractTagAttributes(htmlContent, "iframe", "src")
		for _, url := range iframeMatches {
			if internal.IsValidURL(url) && internal.IsVideoURL(url) && !seen[url] {
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
			if internal.IsValidURL(url) && internal.IsVideoURL(url) && !seen[url] {
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
			if internal.IsValidURL(url) && internal.IsVideoURL(url) && !seen[url] {
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
			if internal.IsValidURL(url) && !seen[url] {
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
			if !internal.IsValidURL(attr.Val) {
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

	if !internal.IsValidURL(video.URL) {
		return VideoInfo{}
	}

	return video
}

func (p *Processor) parseIframeNode(n *Node) VideoInfo {
	for _, attr := range n.Attr {
		if attr.Key == "src" && internal.IsValidURL(attr.Val) && internal.IsVideoURL(attr.Val) {
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
		if (attr.Key == "src" || attr.Key == "data") && internal.IsValidURL(attr.Val) && internal.IsVideoURL(attr.Val) {
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
			if internal.IsValidURL(url) && !seen[url] {
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
			if !internal.IsValidURL(attr.Val) {
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

	if !internal.IsValidURL(audio.URL) {
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

// extractTagAttributes extracts specified attributes from all occurrences of a tag in HTML content.
// This function operates on raw HTML strings before sanitization, allowing extraction from
// tags that might be removed during HTML sanitization (e.g., iframe, embed, object).
func (p *Processor) extractTagAttributes(htmlContent, tagName string, attrNames ...string) []string {
	results := make([]string, 0, extractTagCap)
	// Convert tag name to lowercase once for comparison
	lowerTag := "<" + strings.ToLower(tagName)

	pos := 0
	for pos < len(htmlContent) {
		// Find the next occurrence of the tag using case-insensitive search
		// We'll search in chunks to avoid converting the entire HTML to lowercase
		tagStart := findTagIgnoreCase(htmlContent[pos:], lowerTag)
		if tagStart == -1 {
			break
		}
		tagStart += pos

		// Verify it's a complete tag name (not a partial match)
		if tagStart+len(lowerTag) < len(htmlContent) {
			nextChar := htmlContent[tagStart+len(lowerTag)]
			// The tag name should be followed by whitespace, '>', or '/'
			if nextChar != ' ' && nextChar != '\t' && nextChar != '\n' &&
				nextChar != '\r' && nextChar != '>' && nextChar != '/' {
				pos = tagStart + len(lowerTag)
				continue
			}
		}

		// Find the end of the opening tag
		tagEnd := strings.IndexByte(htmlContent[tagStart:], '>')
		if tagEnd == -1 {
			break
		}
		tagEnd += tagStart + 1

		tagContent := htmlContent[tagStart:tagEnd]

		// Extract requested attributes from this tag
		for _, attrName := range attrNames {
			if value := extractAttributeValue(tagContent, attrName); value != "" {
				results = append(results, value)
			}
		}

		pos = tagEnd
	}

	return results
}

// findTagIgnoreCase performs case-insensitive tag search more efficiently
// by using a combination of Index for candidate positions and EqualFold for verification
func findTagIgnoreCase(html, lowerTag string) int {
	if len(lowerTag) == 0 || len(html) < len(lowerTag) {
		return -1
	}

	// Fast path: try exact match first (most common case)
	if idx := strings.Index(html, lowerTag); idx >= 0 {
		return idx
	}

	// For case-insensitive search, check positions where first character matches (case-insensitive)
	tagLen := len(lowerTag)
	firstChar := lowerTag[0]

	for i := 0; i <= len(html)-tagLen; i++ {
		c := html[i]
		// Quick ASCII case-insensitive check for first character
		cfc := c
		if cfc >= 'A' && cfc <= 'Z' {
			cfc += 32
		}
		if cfc != firstChar {
			continue
		}

		// Found potential match, verify with EqualFold for full case-insensitive comparison
		candidate := html[i : i+tagLen]
		if strings.EqualFold(candidate, lowerTag) {
			return i
		}
	}

	return -1
}

// extractAttributeValue extracts a single attribute value from a tag string.
// It handles quoted (single and double) and unquoted attribute values.
func extractAttributeValue(tagContent, attrName string) string {
	lowerTag := strings.ToLower(tagContent)
	lowerAttr := strings.ToLower(attrName) + "="

	// Find the attribute
	attrIdx := strings.Index(lowerTag, lowerAttr)
	if attrIdx == -1 {
		return ""
	}

	// Verify we're matching a complete attribute name (not a substring)
	if attrIdx > 0 {
		prevChar := lowerTag[attrIdx-1]
		// Attribute should start at beginning or after whitespace
		if prevChar != ' ' && prevChar != '\t' && prevChar != '\n' && prevChar != '\r' {
			return ""
		}
	}

	valueStart := attrIdx + len(attrName) + 1

	// Skip whitespace after '='
	for valueStart < len(tagContent) {
		c := tagContent[valueStart]
		if c != ' ' && c != '\t' {
			break
		}
		valueStart++
	}

	if valueStart >= len(tagContent) {
		return ""
	}

	// Extract quoted or unquoted value
	var value string
	var quote byte

	switch tagContent[valueStart] {
	case '"', '\'':
		// Quoted value
		quote = tagContent[valueStart]
		valueStart++
		valueEnd := strings.IndexByte(tagContent[valueStart:], quote)
		if valueEnd == -1 {
			// Unclosed quote, return rest of tag content
			value = tagContent[valueStart:]
		} else {
			value = tagContent[valueStart : valueStart+valueEnd]
		}
	default:
		// Unquoted value - extract until whitespace or '>'
		valueEnd := valueStart
		for valueEnd < len(tagContent) {
			c := tagContent[valueEnd]
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '>' {
				break
			}
			valueEnd++
		}
		value = tagContent[valueStart:valueEnd]
	}

	return strings.TrimSpace(value)
}
