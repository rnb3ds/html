package internal

import (
	"strings"
)

var tagsToRemove = []string{"script", "style", "noscript"}

func SanitizeHTML(htmlContent string) string {
	if htmlContent == "" {
		return ""
	}
	// Process all tags in sequence
	result := htmlContent
	for _, tag := range tagsToRemove {
		result = RemoveTagContent(result, tag)
		// Early exit if content becomes empty
		if result == "" {
			return ""
		}
	}
	return result
}

func RemoveTagContent(content, tag string) string {
	if content == "" || tag == "" {
		return content
	}

	openTag := "<" + tag
	closeTag := "</" + tag + ">"
	lowerContent := strings.ToLower(content)

	// Early exit: tag not present
	if !strings.Contains(lowerContent, openTag) {
		return content
	}

	// Pre-allocate with estimated capacity
	var result strings.Builder
	result.Grow(len(content))

	pos := 0
	for pos < len(content) {
		// Find next opening tag
		start := strings.Index(lowerContent[pos:], openTag)
		if start == -1 {
			// No more tags, append remaining content
			result.WriteString(content[pos:])
			break
		}
		start += pos

		// Append content before tag
		result.WriteString(content[pos:start])

		// Find end of opening tag
		tagEnd := strings.IndexByte(content[start:], '>')
		if tagEnd == -1 {
			// Malformed tag, append rest and exit
			result.WriteString(content[start:])
			break
		}
		tagEnd += start + 1

		// Find closing tag
		if end := strings.Index(lowerContent[tagEnd:], closeTag); end != -1 {
			// Skip entire tag and its content
			pos = tagEnd + end + len(closeTag)
		} else {
			// No closing tag, skip opening tag only
			pos = tagEnd
		}
	}

	return result.String()
}
