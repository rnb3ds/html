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
	"onclick":     true,
	"onerror":     true,
	"onload":      true,
	"onmouseover": true,
	"onmouseout":  true,
	"onfocus":     true,
	"onblur":      true,
	"onchange":    true,
	"onsubmit":    true,
	"onreset":     true,
	"ondblclick":  true,
}

var uriAttributes = map[string]bool{
	"href":        true,
	"src":         true,
	"cite":        true,
	"action":      true,
	"data":        true,
	"formaction":  true,
	"poster":      true,
	"background":  true,
	"longdesc":    true,
	"usemap":      true,
	"profile":     true,
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

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return ""
	}

	result := buf.String()

	result = strings.ReplaceAll(result, "<html><head></head><body>", "")
	result = strings.ReplaceAll(result, "</body></html>", "")

	return result
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

func RemoveTagContent(content, tag string) string {
	if content == "" || tag == "" {
		return content
	}

	openTag := "<" + tag
	closeTag := "</" + tag + ">"
	lowerContent := strings.ToLower(content)

	if !strings.Contains(lowerContent, openTag) {
		return content
	}

	var result strings.Builder
	result.Grow(len(content))

	pos := 0
	for pos < len(content) {
		start := strings.Index(lowerContent[pos:], openTag)
		if start == -1 {
			result.WriteString(content[pos:])
			break
		}
		start += pos

		if start+len(openTag) < len(content) {
			nextChar := content[start+len(openTag)]
			if nextChar != ' ' && nextChar != '>' && nextChar != '\t' && nextChar != '\n' && nextChar != '/' {
				result.WriteString(content[pos : start+len(openTag)])
				pos = start + len(openTag)
				continue
			}
		}

		result.WriteString(content[pos:start])

		tagEnd := strings.IndexByte(content[start:], '>')
		if tagEnd == -1 {
			result.WriteString(content[start:])
			break
		}
		tagEnd += start + 1

		if end := strings.Index(lowerContent[tagEnd:], closeTag); end != -1 {
			pos = tagEnd + end + len(closeTag)
		} else {
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

	if mediaPart != "" {
		if strings.HasSuffix(mediaPart, ";base64") {
			mediaType := strings.TrimSuffix(mediaPart, ";base64")
			if mediaType != "" && !isValidMediaType(mediaType) {
				return false
			}
		} else if strings.Contains(mediaPart, ";") {
			semicolonIdx := strings.Index(mediaPart, ";")
			mediaType := mediaPart[:semicolonIdx]
			if mediaType != "" && !isValidMediaType(mediaType) {
				return false
			}
		} else {
			if !isValidMediaType(mediaPart) {
				return false
			}
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
