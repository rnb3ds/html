package internal

import (
	"strings"
)

func SanitizeHTML(htmlContent string) string {
	if htmlContent == "" {
		return htmlContent
	}
	htmlContent = RemoveTagContent(htmlContent, "script")
	htmlContent = RemoveTagContent(htmlContent, "style")
	htmlContent = RemoveTagContent(htmlContent, "noscript")
	return htmlContent
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

		result.WriteString(content[pos:start])

		tagEnd := strings.IndexByte(content[start:], '>')
		if tagEnd == -1 {
			result.WriteString(content[start:])
			break
		}
		tagEnd += start + 1

		end := strings.Index(lowerContent[tagEnd:], closeTag)
		if end == -1 {
			pos = tagEnd
			continue
		}
		pos = tagEnd + end + len(closeTag)
	}

	return result.String()
}
