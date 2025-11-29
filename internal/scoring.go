package internal

import (
	"strings"

	"golang.org/x/net/html"
)

var (
	positiveStrongPatterns = []string{"content", "article", "main", "post", "entry", "text", "body", "story"}
	positiveMediumPatterns = []string{"blog", "news", "detail", "page"}
	negativeStrongPatterns = []string{"comment", "sidebar", "nav", "footer", "header", "menu", "ad", "advertisement"}
	negativeMediumPatterns = []string{"widget", "related", "share", "social", "meta", "tag", "category"}
	negativeWeakPatterns   = []string{"promo", "banner", "sponsor"}
)

var removePatterns = []string{
	"nav", "navigation", "menu",
	"sidebar", "side-bar",
	"footer", "header",
	"comment", "comments",
	"ad", "ads", "advertisement",
	"social", "share", "sharing",
	"related", "recommend",
	"widget", "plugin",
	"promo", "promotion",
	"banner", "sponsor",
}

var nonContentTags = map[string]bool{
	"script": true, "style": true, "noscript": true, "nav": true,
	"aside": true, "footer": true, "header": true, "form": true,
}

var blockElements = map[string]bool{
	"p": true, "div": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"article": true, "section": true, "blockquote": true, "pre": true, "ul": true, "ol": true,
	"li": true, "table": true, "tr": true, "td": true, "th": true, "br": true, "hr": true,
}

func ScoreContentNode(n *html.Node) int {
	if n == nil || n.Type != html.ElementNode || IsNonContentElement(n.Data) {
		return 0
	}

	score := 0

	switch n.Data {
	case "article":
		score = 1000
	case "main":
		score = 900
	case "section":
		score = 300
	case "div":
		score = 50
	case "body":
		score = 100
	case "p":
		return 0
	}

	score += ScoreAttributes(n)

	paragraphCount := CountChildElements(n, "p")
	if paragraphCount >= 3 {
		score += paragraphCount * 150
	} else if paragraphCount > 0 {
		score += paragraphCount * 80
	}

	headingCount := CountChildElements(n, "h1") + CountChildElements(n, "h2") +
		CountChildElements(n, "h3") + CountChildElements(n, "h4") +
		CountChildElements(n, "h5") + CountChildElements(n, "h6")
	if headingCount > 0 {
		score += headingCount * 100
	}

	textLength := GetTextLength(n)
	switch {
	case textLength > 500:
		score += 500 + (textLength-500)/10
	case textLength > 200:
		score += textLength / 2
	case textLength > 100:
		score += textLength / 3
	case textLength < 50:
		score -= 300
	}

	contentDensity := CalculateContentDensity(n)
	switch {
	case contentDensity > 0.7:
		score = int(float64(score) * 1.2)
	case contentDensity < 0.3:
		score = int(float64(score) * 0.7)
	}

	linkDensity := GetLinkDensity(n)
	switch {
	case linkDensity > 0.5:
		score = int(float64(score) * 0.2)
	case linkDensity > 0.3:
		score = int(float64(score) * 0.5)
	case linkDensity > 0.15:
		score = int(float64(score) * 0.75)
	}

	textContent := GetTextContent(n)
	if commaCount := strings.Count(textContent, ",") + strings.Count(textContent, "，"); commaCount > 5 {
		score += commaCount * 10
	}

	return score
}

func ScoreAttributes(n *html.Node) int {
	if n == nil {
		return 0
	}
	score := 0

	for _, attr := range n.Attr {
		if attr.Key == "class" || attr.Key == "id" {
			lowerVal := strings.ToLower(attr.Val)

			if MatchesPattern(lowerVal, positiveStrongPatterns) {
				score += 400
			}
			if MatchesPattern(lowerVal, positiveMediumPatterns) {
				score += 200
			}
			if MatchesPattern(lowerVal, negativeStrongPatterns) {
				score -= 400
			}
			if MatchesPattern(lowerVal, negativeMediumPatterns) {
				score -= 200
			}
			if MatchesPattern(lowerVal, negativeWeakPatterns) {
				score -= 300
			}
		}

		if attr.Key == "role" {
			lowerVal := strings.ToLower(attr.Val)
			if lowerVal == "main" || lowerVal == "article" {
				score += 500
			}
			if lowerVal == "navigation" || lowerVal == "complementary" {
				score -= 400
			}
		}
	}

	return score
}

func MatchesPattern(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(value, pattern) {
			return true
		}
	}
	return false
}

func CalculateContentDensity(n *html.Node) float64 {
	textLength := float64(GetTextLength(n))
	if textLength == 0 {
		return 0
	}

	tagCount := float64(CountTags(n))
	if tagCount == 0 {
		return 1.0
	}

	density := textLength / (tagCount * 10)
	if density > 1.0 {
		return 1.0
	}
	return density
}

func CountTags(n *html.Node) int {
	if n == nil {
		return 0
	}
	count := 0
	WalkNodes(n, func(node *html.Node) bool {
		if node.Type == html.ElementNode {
			count++
		}
		return true
	})
	return count
}

func IsNonContentElement(tag string) bool {
	return nonContentTags[tag]
}

func CountChildElements(n *html.Node, tag string) int {
	if n == nil {
		return 0
	}
	count := 0
	WalkNodes(n, func(node *html.Node) bool {
		if node != n && node.Type == html.ElementNode && node.Data == tag {
			count++
		}
		return true
	})
	return count
}

func ShouldRemoveElement(n *html.Node) bool {
	if n == nil || n.Type != html.ElementNode {
		return false
	}
	if IsNonContentElement(n.Data) {
		return true
	}

	for _, attr := range n.Attr {
		if attr.Key == "class" || attr.Key == "id" {
			lowerVal := strings.ToLower(attr.Val)
			for _, pattern := range removePatterns {
				if strings.Contains(lowerVal, pattern) {
					return true
				}
			}
		}

		if attr.Key == "style" && (strings.Contains(attr.Val, "display:none") || strings.Contains(attr.Val, "display: none")) {
			return true
		}
		if attr.Key == "hidden" {
			return true
		}
	}

	return false
}

func IsBlockElement(tag string) bool {
	return blockElements[tag]
}
