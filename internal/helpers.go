package internal

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var defaultWhitespaceRegex = regexp.MustCompile(`\s+`)

func CleanText(text string, whitespaceRegex *regexp.Regexp) string {
	if text == "" {
		return ""
	}
	if whitespaceRegex == nil {
		whitespaceRegex = defaultWhitespaceRegex
	}
	textLen := len(text)
	var result strings.Builder
	result.Grow(textLen >> 1)
	start := 0
	previousWasEmpty := false

	for i := 0; i <= textLen; i++ {
		if i == textLen || text[i] == '\n' {
			line := whitespaceRegex.ReplaceAllString(text[start:i], " ")
			isEmpty := true
			if line != "" {
				if line = strings.TrimSpace(line); line != "" {
					isEmpty = false
					if result.Len() > 0 {
						if previousWasEmpty {
							result.WriteByte('\n')
						}
						result.WriteByte('\n')
					}
					result.WriteString(line)
				}
			}
			previousWasEmpty = isEmpty
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
	prevEndedWithSpace := false

	WalkNodes(node, func(n *html.Node) bool {
		if n.Type == html.TextNode {
			originalData := n.Data
			// Replace internal newlines with spaces to handle multi-line text in HTML
			originalData = strings.ReplaceAll(originalData, "\n", " ")

			// Check if the original text ends/starts with whitespace (before trimming)
			endedWithSpace := len(originalData) > 0 && (originalData[len(originalData)-1] == ' ' || originalData[len(originalData)-1] == '\t')
			startedWithSpace := len(originalData) > 0 && (originalData[0] == ' ' || originalData[0] == '\t')

			text := strings.TrimSpace(originalData)
			if text != "" {
				// Add space only if previous node ended with space OR current node started with space
				needsSpace := prevEndedWithSpace
				if !needsSpace && sb.Len() > 0 {
					needsSpace = startedWithSpace
				}

				if sb.Len() > 0 && needsSpace {
					sb.WriteByte(' ')
				}
				sb.WriteString(text)
			}
			prevEndedWithSpace = endedWithSpace
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
