package internal

import (
	"strings"
)

var tagsToRemove = []string{"script", "style", "noscript"}

func SanitizeHTML(htmlContent string) string {
	if htmlContent == "" {
		return ""
	}
	for _, tag := range tagsToRemove {
		htmlContent = RemoveTagContent(htmlContent, tag)
		if htmlContent == "" {
			return ""
		}
	}
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

		if end := strings.Index(lowerContent[tagEnd:], closeTag); end != -1 {
			pos = tagEnd + end + len(closeTag)
		} else {
			pos = tagEnd
		}
	}

	return result.String()
}
