package internal

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

var tagsToRemove = []string{
	"script", "style", "noscript", "iframe",
	"embed", "object", "form", "input", "button",
}

var dangerousAttributes = map[string]bool{
	// Mouse events
	"onclick":      true,
	"ondblclick":   true,
	"onmousedown":  true,
	"onmouseup":    true,
	"onmouseover":  true,
	"onmousemove":  true,
	"onmouseout":   true,
	"onmouseenter": true,
	"onmouseleave": true,
	// Keyboard events
	"onkeydown":  true,
	"onkeypress": true,
	"onkeyup":    true,
	// Focus events
	"onfocus": true,
	"onblur":  true,
	// Form events
	"onsubmit": true,
	"onreset":  true,
	"onchange": true,
	"onselect": true,
	// UI events
	"onload":        true,
	"onunload":      true,
	"onabort":       true,
	"onerror":       true,
	"onresize":      true,
	"onscroll":      true,
	"oncontextmenu": true,
	// Drag and drop events
	"ondrag":      true,
	"ondragstart": true,
	"ondragend":   true,
	"ondragenter": true,
	"ondragleave": true,
	"ondragover":  true,
	"ondrop":      true,
	// Clipboard events
	"oncopy":  true,
	"oncut":   true,
	"onpaste": true,
	// Media events
	"onplay":         true,
	"onpause":        true,
	"onended":        true,
	"onvolumechange": true,
	// Mutation events (deprecated but still dangerous)
	"onDOMAttrModified":          true,
	"onDOMCharacterDataModified": true,
	"onDOMNodeInserted":          true,
	"onDOMNodeRemoved":           true,
	// Animation events
	"onanimationstart":     true,
	"onanimationend":       true,
	"onanimationiteration": true,
	// Transition events
	"ontransitionend": true,
	// Touch events
	"ontouchstart":  true,
	"ontouchend":    true,
	"ontouchmove":   true,
	"ontouchcancel": true,
	// Pointer events
	"onpointerdown":   true,
	"onpointerup":     true,
	"onpointermove":   true,
	"onpointercancel": true,
	"onpointerenter":  true,
	"onpointerleave":  true,
	"onpointerover":   true,
	"onpointerout":    true,
	// Other dangerous attributes
	"formaction": true, // Can override form action
	"autofocus":  true, // Can be used for phishing
}

var uriAttributes = map[string]bool{
	"href":       true,
	"src":        true,
	"cite":       true,
	"action":     true,
	"data":       true,
	"formaction": true,
	"poster":     true,
	"background": true,
	"longdesc":   true,
	"usemap":     true,
	"profile":    true,
}

func SanitizeHTML(htmlContent string) string {
	if htmlContent == "" {
		return ""
	}

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return ""
	}

	sanitizeNode(doc)

	// Find body element and extract its content properly
	body := findBodyElement(doc)
	if body == nil {
		// No body element found, render the entire document (fragment case)
		var buf bytes.Buffer
		if err := html.Render(&buf, doc); err != nil {
			return ""
		}
		result := buf.String()
		// Remove the automatic html/head/body wrapper for fragments
		result = strings.ReplaceAll(result, "<html><head></head><body>", "")
		result = strings.ReplaceAll(result, "</body></html>", "")
		return result
	}

	var buf bytes.Buffer
	for child := body.FirstChild; child != nil; child = child.NextSibling {
		if err := html.Render(&buf, child); err != nil {
			continue
		}
	}

	return buf.String()
}

// findBodyElement locates the body element in the parsed HTML document
func findBodyElement(doc *html.Node) *html.Node {
	if doc.Type != html.DocumentNode {
		return nil
	}
	for child := doc.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "body" {
			return child
		}
	}
	return nil
}

func sanitizeNode(n *html.Node) {
	if n.Type == html.ElementNode {
		for _, tag := range tagsToRemove {
			if strings.EqualFold(n.Data, tag) {
				removeNode(n)
				return
			}
		}

		var filteredAttrs []html.Attribute
		for _, attr := range n.Attr {
			attrKey := strings.ToLower(attr.Key)
			if dangerousAttributes[attrKey] {
				continue
			}
			if uriAttributes[attrKey] {
				if !isSafeURI(attr.Val) {
					continue
				}
			}
			filteredAttrs = append(filteredAttrs, attr)
		}
		n.Attr = filteredAttrs
	}

	child := n.FirstChild
	for child != nil {
		next := child.NextSibling
		sanitizeNode(child)
		child = next
	}
}

func removeNode(n *html.Node) {
	if n.Parent != nil {
		n.Parent.RemoveChild(n)
	}
}

// RemoveTagContent removes all occurrences of the specified HTML tag and its content.
// This function uses string-based parsing as the primary method to handle edge cases
// like unclosed tags, malformed HTML, and to preserve original character case.
func RemoveTagContent(content, tag string) string {
	if content == "" || tag == "" {
		return content
	}

	// Quick check: if tag is not present, return as-is
	openTag := "<" + strings.ToLower(tag)
	if !strings.Contains(strings.ToLower(content), openTag) {
		return content
	}

	return removeTagContentStringBased(content, tag)
}

// removeTagContentStringBased removes tags using string operations.
// This approach preserves character case and handles malformed HTML better than DOM parsing.
func removeTagContentStringBased(content, tag string) string {
	openTag := "<" + strings.ToLower(tag)
	closeTag := "</" + strings.ToLower(tag) + ">"
	lowerContent := strings.ToLower(content)

	var result strings.Builder
	result.Grow(len(content))

	pos := 0
	for pos < len(content) {
		// Find the next opening tag (case-insensitive)
		start := strings.Index(lowerContent[pos:], openTag)
		if start == -1 {
			result.WriteString(content[pos:])
			break
		}
		start += pos

		// Verify this is actually the tag we're looking for
		// Check that the character after the tag name is valid (space, >, /, or end)
		tagNameLen := len(tag) + 1 // +1 for the '<'
		if start+tagNameLen < len(content) {
			nextChar := lowerContent[start+tagNameLen]
			if nextChar != ' ' && nextChar != '>' && nextChar != '\t' && nextChar != '\n' && nextChar != '/' {
				// Not our tag, write content up to after the match and continue
				result.WriteString(content[pos : start+tagNameLen])
				pos = start + tagNameLen
				continue
			}
		}

		// Write content before the tag
		result.WriteString(content[pos:start])

		// Find the end of the opening tag
		tagEnd := strings.IndexByte(content[start:], '>')
		if tagEnd == -1 {
			// Unclosed tag, write the rest as-is
			result.WriteString(content[start:])
			break
		}
		tagEnd += start + 1

		// Look for the corresponding closing tag
		if end := strings.Index(lowerContent[tagEnd:], closeTag); end != -1 {
			// Found closing tag, skip everything between opening and closing
			pos = tagEnd + end + len(closeTag)
		} else {
			// No closing tag found, this is malformed HTML
			// Skip the opening tag and continue
			pos = tagEnd
		}
	}

	return result.String()
}

func isSafeURI(uri string) bool {
	if uri == "" {
		return true
	}

	trimmed := strings.TrimSpace(uri)
	lowerURI := strings.ToLower(trimmed)

	if strings.HasPrefix(lowerURI, "javascript:") {
		return false
	}

	if strings.HasPrefix(lowerURI, "vbscript:") {
		return false
	}

	if strings.HasPrefix(lowerURI, "file:") {
		return false
	}

	if strings.HasPrefix(lowerURI, "data:") {
		return isValidDataURL(trimmed)
	}

	return true
}

func isValidDataURL(url string) bool {
	if !strings.HasPrefix(url, "data:") {
		return false
	}

	commaIdx := strings.Index(url, ",")
	if commaIdx == -1 || commaIdx == 5 {
		return false
	}

	mediaPart := url[5:commaIdx]
	dataPart := url[commaIdx+1:]

	// Enforce maximum data URL size to prevent memory exhaustion
	if len(url) > 100000 {
		return false
	}

	if mediaPart != "" {
		var mediaType string
		if strings.HasSuffix(mediaPart, ";base64") {
			mediaType = strings.TrimSuffix(mediaPart, ";base64")
		} else if strings.Contains(mediaPart, ";") {
			semicolonIdx := strings.Index(mediaPart, ";")
			mediaType = mediaPart[:semicolonIdx]
		} else {
			mediaType = mediaPart
		}

		// Validate media type and check against whitelist of safe types
		if mediaType != "" && !isValidMediaType(mediaType) {
			return false
		}
		if mediaType != "" && !isSafeMediaType(mediaType) {
			return false
		}
	}

	isBase64 := strings.Contains(mediaPart, ";base64")
	for i := 0; i < len(dataPart); i++ {
		b := dataPart[i]
		if isBase64 {
			if !(isBase64Char(b) || b == '=' || b == '\r' || b == '\n') {
				return false
			}
		} else {
			if b < 9 || (b >= 11 && b <= 12) || (b >= 14 && b < 32) || b == 127 {
				return false
			}
		}
	}

	return true
}

// isSafeMediaType checks if the media type is in the whitelist of safe types.
// This prevents XSS through script data URIs and other dangerous content types.
func isSafeMediaType(mediaType string) bool {
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))

	// Whitelist of safe media types
	safeTypes := map[string]bool{
		// Image types (explicit whitelist for security)
		"image/gif":              true,
		"image/jpeg":             true,
		"image/jpg":              true,
		"image/png":              true,
		"image/webp":             true,
		"image/bmp":              true,
		"image/x-icon":           true,
		"image/vnd.microsoft.icon": true,
		"image/avif":             true,
		"image/apng":             true,
		// Note: image/svg+xml is NOT included because SVG can contain JavaScript
		// and other executable content that poses XSS risks
		// Font types
		"font/woff":              true,
		"font/woff2":             true,
		"font/ttf":               true,
		"font/otf":               true,
		"application/font-woff":  true,
		"application/font-woff2": true,
		// Common document formats
		"application/pdf": true,
	}

	// Check whitelist first for explicit allow-list
	if safeTypes[mediaType] {
		return true
	}

	// No wildcard matching - all types must be explicitly whitelisted
	// This prevents bypass attempts with variants like image/svg+xml
	return false
}

func isValidMediaType(mediaType string) bool {
	if mediaType == "" {
		return false
	}

	slashIdx := strings.Index(mediaType, "/")
	if slashIdx <= 0 || slashIdx == len(mediaType)-1 {
		return false
	}

	for i := 0; i < len(mediaType); i++ {
		c := mediaType[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '+' ||
			c == '/' || c == '.' || c == '_') {
			return false
		}
	}

	return true
}

func isBase64Char(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '+' || b == '/'
}
