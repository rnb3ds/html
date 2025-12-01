package internal

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

func WalkNodes(node *html.Node, fn func(*html.Node) bool) {
	if node == nil || !fn(node) {
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

func GetTextLength(n *html.Node) int {
	length := 0
	WalkNodes(n, func(node *html.Node) bool {
		if node.Type == html.TextNode {
			length += len(strings.TrimSpace(node.Data))
		}
		return true
	})
	return length
}

func GetLinkDensity(n *html.Node) float64 {
	textLength := GetTextLength(n)
	if textLength == 0 {
		return 0.0
	}

	linkTextLength := 0
	WalkNodes(n, func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "a" {
			linkTextLength += GetTextLength(node)
		}
		return true
	})

	return float64(linkTextLength) / float64(textLength)
}

func CleanText(text string, whitespaceRegex *regexp.Regexp) string {
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
			if line := whitespaceRegex.ReplaceAllString(text[start:i], " "); line != "" {
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
