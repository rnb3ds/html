package internal

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var (
	// Package-level regex for whitespace (compiled once, thread-safe in Go 1.6+)
	whitespaceRegex = regexp.MustCompile(`\s+`)
)

// CleanTextWithRegex cleans text using provided regex.
// Note: The regex parameter should be a compiled regex pattern.
// In Go 1.6+, regexp.Regexp is safe for concurrent use.
func CleanTextWithRegex(text string, whitespaceRegex *regexp.Regexp) string {
	if text == "" {
		return ""
	}
	if whitespaceRegex == nil {
		return ReplaceHTMLEntities(strings.TrimSpace(text))
	}
	textLen := len(text)
	var result strings.Builder
	result.Grow(textLen >> 1)
	start := 0
	for i := 0; i <= textLen; i++ {
		if i == textLen || text[i] == '\n' {
			line := whitespaceRegex.ReplaceAllString(text[start:i], " ")
			if line != "" {
				if line = strings.TrimSpace(line); line != "" {
					if result.Len() > 0 {
						result.WriteByte('\n')
					}
					result.WriteString(line)
				}
			}
			start = i + 1
		}
	}
	return ReplaceHTMLEntities(result.String())
}

func WalkNodes(node *html.Node, fn func(*html.Node) bool) {
	if node == nil || fn == nil || !fn(node) {
		return
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		WalkNodes(child, fn)
	}
}

func FindElementByTag(doc *html.Node, tagName string) *html.Node {
	var result *html.Node
	WalkNodes(doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == tagName {
			result = n
			return false
		}
		return true
	})
	return result
}

func GetTextContent(node *html.Node) string {
	var sb strings.Builder
	sb.Grow(256)
	WalkNodes(node, func(n *html.Node) bool {
		if n.Type == html.TextNode {
			if text := strings.TrimSpace(n.Data); text != "" {
				if sb.Len() > 0 {
					sb.WriteByte(' ')
				}
				sb.WriteString(text)
			}
		}
		return true
	})
	return sb.String()
}

func GetTextLength(node *html.Node) int {
	length := 0
	WalkNodes(node, func(n *html.Node) bool {
		if n.Type == html.TextNode {
			length += len(strings.TrimSpace(n.Data))
		}
		return true
	})
	return length
}

func GetLinkDensity(node *html.Node) float64 {
	if node == nil {
		return 0.0
	}

	textLength := 0
	linkTextLength := 0

	WalkNodes(node, func(n *html.Node) bool {
		if n.Type == html.TextNode {
			length := len(strings.TrimSpace(n.Data))
			textLength += length
			// Check if any ancestor is an <a> tag
			for parent := n.Parent; parent != nil; parent = parent.Parent {
				if parent.Type == html.ElementNode && parent.Data == "a" {
					linkTextLength += length
					break
				}
			}
		}
		return true
	})

	if textLength == 0 {
		return 0.0
	}
	return float64(linkTextLength) / float64(textLength)
}

// CleanText cleans text with optional whitespace regex replacement.
// For backward compatibility, it accepts an optional regex parameter.
func CleanText(text string, whitespaceRegex *regexp.Regexp) string {
	return CleanTextWithRegex(text, whitespaceRegex)
}

var entityReplacer = strings.NewReplacer(
	"&nbsp;", " ",
	"&amp;", "&",
	"&lt;", "<",
	"&gt;", ">",
	"&quot;", "\"",
	"&apos;", "'",
	"&mdash;", "-",
	"&ndash;", "-",
)

func ReplaceHTMLEntities(text string) string {
	if !strings.ContainsRune(text, '&') {
		return text
	}
	return entityReplacer.Replace(text)
}

func IsExternalURL(url string) bool {
	return strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "//")
}

func SelectBestCandidate(candidates map[*html.Node]int) *html.Node {
	var bestNode *html.Node
	bestScore := -1

	for node, score := range candidates {
		if score > bestScore {
			bestScore = score
			bestNode = node
		}
	}

	return bestNode
}
