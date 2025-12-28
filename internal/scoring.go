package internal

import (
	"strings"

	"golang.org/x/net/html"
)

// Content scoring patterns for article detection
const (
	// Strong positive indicators
	strongPositiveScore = 400
	mediumPositiveScore = 200

	// Strong negative indicators
	strongNegativeScore = -400
	mediumNegativeScore = -200
	weakNegativeScore   = -300
)

// Pattern registry for content scoring
var contentPatterns = struct {
	positiveStrong []string
	positiveMedium []string
	negativeStrong []string
	negativeMedium []string
	negativeWeak   []string
	removePatterns []string
}{
	positiveStrong: []string{"content", "article", "main", "post", "entry", "text", "body", "story"},
	positiveMedium: []string{"blog", "news", "detail", "page"},
	negativeStrong: []string{"comment", "sidebar", "nav", "footer", "header", "menu", "ad", "advertisement"},
	negativeMedium: []string{"widget", "related", "share", "social", "meta", "tag", "category"},
	negativeWeak:   []string{"promo", "banner", "sponsor"},
	removePatterns: []string{
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
	},
}

// Non-content elements that should be excluded
var nonContentTags = map[string]bool{
	"script": true, "style": true, "noscript": true, "nav": true,
	"aside": true, "footer": true, "header": true, "form": true,
}

// Block-level elements for structure detection
var blockElements = map[string]bool{
	"p": true, "div": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"article": true, "section": true, "blockquote": true, "pre": true, "ul": true, "ol": true,
	"li": true, "table": true, "tr": true, "td": true, "th": true, "br": true, "hr": true,
}

// Tag-based scoring weights
var tagScores = map[string]int{
	"article": 1000,
	"main":    900,
	"section": 300,
	"body":    100,
	"div":     50,
	"p":       0,
}

// ScoreContentNode calculates content relevance score for a node.
func ScoreContentNode(n *html.Node) int {
	if n == nil || n.Type != html.ElementNode || IsNonContentElement(n.Data) || n.Data == "p" {
		return 0
	}

	score := tagScores[n.Data] + ScoreAttributes(n)

	// Score based on paragraph count
	paragraphCount := CountChildElements(n, "p")
	if paragraphCount >= 3 {
		score += paragraphCount * 150
	} else if paragraphCount > 0 {
		score += paragraphCount * 80
	}

	// Score based on heading count
	headingCount := CountChildElements(n, "h1") + CountChildElements(n, "h2") +
		CountChildElements(n, "h3") + CountChildElements(n, "h4") +
		CountChildElements(n, "h5") + CountChildElements(n, "h6")
	if headingCount > 0 {
		score += headingCount * 100
	}

	// Score based on text length
	textLength := GetTextLength(n)
	switch {
	case textLength > 500:
		score += 500 + (textLength-500)/10
	case textLength > 200:
		score += textLength >> 1
	case textLength > 100:
		score += textLength / 3
	case textLength < 50:
		score -= 300
	}

	// Apply content density multiplier
	contentDensity := CalculateContentDensity(n)
	if contentDensity > 0.7 {
		score = int(float64(score) * 1.2)
	} else if contentDensity < 0.3 {
		score = int(float64(score) * 0.7)
	}

	// Penalize high link density (likely navigation/spam)
	linkDensity := GetLinkDensity(n)
	if linkDensity > 0.5 {
		score = int(float64(score) * 0.2)
	} else if linkDensity > 0.3 {
		score = int(float64(score) * 0.5)
	} else if linkDensity > 0.15 {
		score = int(float64(score) * 0.75)
	}

	// Bonus for comma-rich content (likely prose)
	textContent := GetTextContent(n)
	if commaCount := strings.Count(textContent, ",") + strings.Count(textContent, "，"); commaCount > 5 {
		score += commaCount * 10
	}

	return score
}

// ScoreAttributes scores node based on class/id attributes.
func ScoreAttributes(n *html.Node) int {
	if n == nil {
		return 0
	}

	score := 0
	for _, attr := range n.Attr {
		switch attr.Key {
		case "class", "id":
			lowerVal := strings.ToLower(attr.Val)
			if MatchesPattern(lowerVal, contentPatterns.positiveStrong) {
				score += strongPositiveScore
			}
			if MatchesPattern(lowerVal, contentPatterns.positiveMedium) {
				score += mediumPositiveScore
			}
			if MatchesPattern(lowerVal, contentPatterns.negativeStrong) {
				score += strongNegativeScore
			}
			if MatchesPattern(lowerVal, contentPatterns.negativeMedium) {
				score += mediumNegativeScore
			}
			if MatchesPattern(lowerVal, contentPatterns.negativeWeak) {
				score += weakNegativeScore
			}
		case "role":
			lowerVal := strings.ToLower(attr.Val)
			switch lowerVal {
			case "main", "article":
				score += 500
			case "navigation", "complementary":
				score -= 400
			}
		}
	}
	return score
}

// MatchesPattern checks if value contains any of the patterns.
func MatchesPattern(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(value, pattern) {
			return true
		}
	}
	return false
}

// CalculateContentDensity calculates text-to-tag ratio.
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

// CountTags counts HTML elements in node tree.
func CountTags(n *html.Node) int {
	count := 0
	WalkNodes(n, func(node *html.Node) bool {
		if node.Type == html.ElementNode {
			count++
		}
		return true
	})
	return count
}

// IsNonContentElement checks if tag should be excluded from content.
func IsNonContentElement(tag string) bool {
	return nonContentTags[tag]
}

// CountChildElements counts child elements of specific tag type.
func CountChildElements(n *html.Node, tag string) int {
	count := 0
	WalkNodes(n, func(node *html.Node) bool {
		if node != n && node.Type == html.ElementNode && node.Data == tag {
			count++
		}
		return true
	})
	return count
}

// ShouldRemoveElement determines if element should be removed during cleaning.
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
			for _, pattern := range contentPatterns.removePatterns {
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

// IsBlockElement checks if tag is a block-level element.
func IsBlockElement(tag string) bool {
	return blockElements[tag]
}
