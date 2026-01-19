package internal

import (
	"strings"
)

var tagsToRemove = []string{
	"script", "style", "noscript", "iframe",
	"embed", "object", "form", "input", "button",
}

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
	return removeDangerousAttributes(htmlContent)
}

func removeDangerousAttributes(htmlContent string) string {
	if htmlContent == "" {
		return ""
	}

	var result strings.Builder
	result.Grow(len(htmlContent))

	lowerContent := strings.ToLower(htmlContent)
	pos := 0

	dangerousPrefixes := []string{
		" on", "javascript:", "vbscript:", "data:",
	}

	for pos < len(htmlContent) {
		earliestDangerous := -1

		for _, prefix := range dangerousPrefixes {
			if idx := strings.Index(lowerContent[pos:], prefix); idx != -1 {
				actualPos := pos + idx
				if earliestDangerous == -1 || actualPos < earliestDangerous {
					earliestDangerous = actualPos
				}
			}
		}

		if earliestDangerous == -1 {
			result.WriteString(htmlContent[pos:])
			break
		}

		result.WriteString(htmlContent[pos:earliestDangerous])

		endPos := earliestDangerous + 1
		for endPos < len(htmlContent) {
			c := htmlContent[endPos]
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '>' || c == '=' {
				break
			}
			endPos++
		}
		pos = endPos
	}

	return result.String()
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
