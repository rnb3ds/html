package internal

import (
	"strings"

	"golang.org/x/net/html"
)

var tagsToRemove = []string{
	// Script and style containers
	"script", "style", "noscript",
	// Embedded content (potential XSS vectors)
	"iframe", "embed", "object",
	// Form elements (potential CSRF/UI redress)
	"form", "input", "button",
	// SVG can contain JavaScript and event handlers
	"svg",
	// MathML can be abused for XSS in some browsers
	"math",
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
	"href":   true,
	"src":    true,
	"cite":   true,
	"action": true,
	"data":   true,
	// Note: "formaction" is not included here as it's already in dangerousAttributes
	// which blocks it completely. Having it here would be redundant.
	"poster":     true,
	"background": true,
	"longdesc":   true,
	"usemap":     true,
	"profile":    true,
	// SVG attack vectors - xlink:href can execute JavaScript
	"xlink:href": true,
}

func SanitizeHTML(htmlContent string) string {
	return SanitizeHTMLWithAudit(htmlContent, NoOpAuditRecorder{})
}

// SanitizeHTMLWithAudit sanitizes HTML content and records security events.
// The audit recorder receives events for blocked tags, attributes, and URLs.
func SanitizeHTMLWithAudit(htmlContent string, audit AuditRecorder) string {
	if htmlContent == "" {
		return ""
	}

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return ""
	}

	sanitizeNodeWithAudit(doc, audit)

	// Find body element and extract its content properly
	body := findBodyElement(doc)
	if body == nil {
		// No body element found, render the entire document (fragment case)
		buf := GetBuffer()
		defer PutBuffer(buf)

		if err := html.Render(buf, doc); err != nil {
			return ""
		}
		result := buf.String()
		// Remove the automatic html/head/body wrapper for fragments
		result = strings.ReplaceAll(result, "<html><head></head><body>", "")
		result = strings.ReplaceAll(result, "</body></html>", "")
		return result
	}

	buf := GetBuffer()
	defer PutBuffer(buf)

	for child := body.FirstChild; child != nil; child = child.NextSibling {
		if err := html.Render(buf, child); err != nil {
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

func sanitizeNodeWithAudit(n *html.Node, audit AuditRecorder) {
	if n.Type == html.ElementNode {
		for _, tag := range tagsToRemove {
			if strings.EqualFold(n.Data, tag) {
				audit.RecordBlockedTag(n.Data)
				removeNode(n)
				return
			}
		}

		// Pre-allocate filtered attributes slice with capacity
		// Use the original length as capacity to avoid reallocation in most cases
		attrLen := len(n.Attr)
		if attrLen > 0 {
			filteredAttrs := make([]html.Attribute, 0, attrLen)
			for _, attr := range n.Attr {
				attrKey := strings.ToLower(attr.Key)
				if dangerousAttributes[attrKey] {
					audit.RecordBlockedAttr(attr.Key, attr.Val)
					continue
				}
				if uriAttributes[attrKey] {
					if !isSafeURIWithAudit(attr.Val, audit) {
						continue
					}
				}
				filteredAttrs = append(filteredAttrs, attr)
			}
			n.Attr = filteredAttrs
		}
	}

	child := n.FirstChild
	for child != nil {
		next := child.NextSibling
		sanitizeNodeWithAudit(child, audit)
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
// Uses BuilderPool for memory efficiency to reduce allocations during string building.
func removeTagContentStringBased(content, tag string) string {
	openTag := "<" + strings.ToLower(tag)
	closeTag := "</" + strings.ToLower(tag) + ">"
	lowerContent := strings.ToLower(content)

	// Use pooled builder for better memory efficiency
	sb := GetBuilder()
	defer PutBuilder(sb)

	sb.Grow(len(content))

	pos := 0
	for pos < len(content) {
		// Find the next opening tag (case-insensitive)
		start := strings.Index(lowerContent[pos:], openTag)
		if start == -1 {
			sb.WriteString(content[pos:])
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
				sb.WriteString(content[pos : start+tagNameLen])
				pos = start + tagNameLen
				continue
			}
		}

		// Write content before the tag
		sb.WriteString(content[pos:start])

		// Find the end of the opening tag
		tagEnd := strings.IndexByte(content[start:], '>')
		if tagEnd == -1 {
			// Unclosed tag, write the rest as-is
			sb.WriteString(content[start:])
			break
		}
		tagEnd += start + 1
		// Ensure tagEnd is within bounds
		if tagEnd > len(content) {
			tagEnd = len(content)
		}

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

	return sb.String()
}

func isSafeURI(uri string) bool {
	return isSafeURIWithAudit(uri, NoOpAuditRecorder{})
}

func isSafeURIWithAudit(uri string, audit AuditRecorder) bool {
	if uri == "" {
		return true
	}

	trimmed := strings.TrimSpace(uri)
	lowerURI := strings.ToLower(trimmed)

	if strings.HasPrefix(lowerURI, "javascript:") {
		audit.RecordBlockedURL(uri, "javascript scheme")
		return false
	}

	if strings.HasPrefix(lowerURI, "vbscript:") {
		audit.RecordBlockedURL(uri, "vbscript scheme")
		return false
	}

	if strings.HasPrefix(lowerURI, "file:") {
		audit.RecordBlockedURL(uri, "file scheme")
		return false
	}

	if strings.HasPrefix(lowerURI, "data:") {
		// Explicitly block SVG data URLs - they can contain JavaScript
		// This provides defense-in-depth in case SVG tag removal is bypassed
		if strings.Contains(lowerURI, "image/svg+xml") {
			audit.RecordBlockedURL(uri, "svg data url")
			return false
		}
		if !isValidDataURLWithAudit(trimmed, audit) {
			return false
		}
	}

	return true
}

func isValidDataURL(url string) bool {
	return isValidDataURLWithAudit(url, NoOpAuditRecorder{})
}

func isValidDataURLWithAudit(url string, audit AuditRecorder) bool {
	if !strings.HasPrefix(url, "data:") {
		return false
	}

	commaIdx := strings.Index(url, ",")
	if commaIdx == -1 || commaIdx == 5 {
		audit.RecordBlockedURL(url, "malformed data URL")
		return false
	}

	mediaPart := url[5:commaIdx]
	dataPart := url[commaIdx+1:]

	// Enforce maximum data URL size to prevent memory exhaustion
	// Uses the same limit as IsValidURL for consistency
	if len(url) > MaxDataURILength {
		audit.RecordBlockedURL(url, "data URL exceeds size limit")
		return false
	}

	if mediaPart != "" {
		var mediaType string
		if strings.HasSuffix(mediaPart, ";base64") {
			mediaType = strings.TrimSuffix(mediaPart, ";base64")
		} else if strings.Contains(mediaPart, ";") {
			semicolonIdx := strings.Index(mediaPart, ";")
			if semicolonIdx > 0 {
				mediaType = mediaPart[:semicolonIdx]
			}
			// If semicolonIdx == 0, mediaType remains empty (will be handled by validation below)
		} else {
			mediaType = mediaPart
		}

		// Validate media type and check against whitelist of safe types
		if mediaType != "" && !isValidMediaType(mediaType) {
			audit.RecordBlockedURL(url, "invalid media type in data URL")
			return false
		}
		if mediaType != "" && !isSafeMediaType(mediaType) {
			audit.RecordBlockedURL(url, "unsafe media type in data URL: "+mediaType)
			return false
		}
	}

	isBase64 := strings.Contains(mediaPart, ";base64")
	for i := 0; i < len(dataPart); i++ {
		b := dataPart[i]
		if isBase64 {
			if !(isBase64Char(b) || b == '=' || b == '\r' || b == '\n') {
				audit.RecordBlockedURL(url, "invalid base64 in data URL")
				return false
			}
		} else {
			if b < 9 || (b >= 11 && b <= 12) || (b >= 14 && b < 32) || b == 127 {
				audit.RecordBlockedURL(url, "invalid character in data URL")
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
		"image/gif":                true,
		"image/jpeg":               true,
		"image/jpg":                true,
		"image/png":                true,
		"image/webp":               true,
		"image/bmp":                true,
		"image/x-icon":             true,
		"image/vnd.microsoft.icon": true,
		"image/avif":               true,
		"image/apng":               true,
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
