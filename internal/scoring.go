package internal

import (
	"strings"

	"golang.org/x/net/html"
)

// Scoring constants for content relevance evaluation.
const (
	strongPositiveScore = 400  // Strong positive indicators (article, main, content)
	mediumPositiveScore = 200  // Medium positive indicators (blog, news, detail)
	strongNegativeScore = -400 // Strong negative indicators (comment, sidebar, nav)
	mediumNegativeScore = -200 // Medium negative indicators (widget, related, share)
	weakNegativeScore   = -100 // Weak negative indicators (promo, banner, sponsor)
)

var (
	positiveStrongPatterns = map[string]int{
		"content": strongPositiveScore, "article": strongPositiveScore, "main": strongPositiveScore,
		"post": strongPositiveScore, "entry": strongPositiveScore, "text": strongPositiveScore,
		"body": strongPositiveScore, "story": strongPositiveScore,
	}
	positiveMediumPatterns = map[string]int{
		"blog": mediumPositiveScore, "news": mediumPositiveScore,
		"detail": mediumPositiveScore, "page": mediumPositiveScore,
	}
	negativeStrongPatterns = map[string]int{
		"comment": strongNegativeScore, "sidebar": strongNegativeScore, "nav": strongNegativeScore,
		"footer": strongNegativeScore, "header": strongNegativeScore, "menu": strongNegativeScore,
		"ad": strongNegativeScore, "advertisement": strongNegativeScore,
	}
	negativeMediumPatterns = map[string]int{
		"widget": mediumNegativeScore, "related": mediumNegativeScore, "share": mediumNegativeScore,
		"social": mediumNegativeScore, "meta": mediumNegativeScore, "tag": mediumNegativeScore,
		"category": mediumNegativeScore,
	}
	negativeWeakPatterns = map[string]int{
		"promo": weakNegativeScore, "banner": weakNegativeScore, "sponsor": weakNegativeScore,
	}

	removePatterns = map[string]bool{
		"nav": true, "navigation": true, "menu": true,
		"sidebar": true, "side-bar": true,
		"footer": true, "header": true,
		"comment": true, "comments": true,
		"ad": true, "ads": true, "advertisement": true,
		"social": true, "share": true, "sharing": true,
		"related": true, "recommend": true,
		"widget": true, "plugin": true,
		"promo": true, "promotion": true,
		"banner": true, "sponsor": true,
	}

	nonContentTags = map[string]bool{
		"script": true, "style": true, "noscript": true, "nav": true,
		"aside": true, "footer": true, "header": true, "form": true,
	}

	blockElements = map[string]bool{
		"p": true, "div": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"article": true, "section": true, "blockquote": true, "pre": true, "ul": true, "ol": true,
		"li": true, "table": true, "tr": true, "td": true, "th": true, "br": true, "hr": true,
	}

	tagScores = map[string]int{
		"article": 1000,
		"main":    900,
		"section": 300,
		"body":    100,
		"div":     50,
		"p":       0,
	}
)

// ScoreContentNode calculates content relevance score for a node.
func ScoreContentNode(node *html.Node) int {
	if node == nil || node.Type != html.ElementNode || IsNonContentElement(node.Data) || node.Data == "p" {
		return 0
	}

	score := tagScores[node.Data] + ScoreAttributes(node)

	// Score based on paragraph count
	paragraphCount := CountChildElements(node, "p")
	if paragraphCount >= 3 {
		score += paragraphCount * 150
	} else if paragraphCount > 0 {
		score += paragraphCount * 80
	}

	// Score based on heading count
	headingCount := CountChildElements(node, "h1") + CountChildElements(node, "h2") +
		CountChildElements(node, "h3") + CountChildElements(node, "h4") +
		CountChildElements(node, "h5") + CountChildElements(node, "h6")
	if headingCount > 0 {
		score += headingCount * 100
	}

	// Score based on text length
	textLength := GetTextLength(node)
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
	contentDensity := CalculateContentDensity(node)
	if contentDensity > 0.7 {
		score = int(float64(score) * 1.2)
	} else if contentDensity < 0.3 {
		score = int(float64(score) * 0.7)
	}

	// Penalize high link density (likely navigation/spam)
	linkDensity := GetLinkDensity(node)
	if linkDensity > 0.5 {
		score = int(float64(score) * 0.2)
	} else if linkDensity > 0.3 {
		score = int(float64(score) * 0.5)
	} else if linkDensity > 0.15 {
		score = int(float64(score) * 0.75)
	}

	// Bonus for comma-rich content (likely prose)
	commaCount := countCommas(node)
	if commaCount > 5 {
		score += commaCount * 10
	}

	return score
}

// countCommas efficiently counts commas without building full text string.
func countCommas(node *html.Node) int {
	count := 0
	WalkNodes(node, func(n *html.Node) bool {
		if n.Type == html.TextNode {
			count += strings.Count(n.Data, ",") + strings.Count(n.Data, "，")
		}
		return true
	})
	return count
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
			score += checkPatterns(lowerVal, positiveStrongPatterns)
			score += checkPatterns(lowerVal, positiveMediumPatterns)
			score += checkPatterns(lowerVal, negativeStrongPatterns)
			score += checkPatterns(lowerVal, negativeMediumPatterns)
			score += checkPatterns(lowerVal, negativeWeakPatterns)
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

// checkPatterns returns the sum of all pattern scores that match the value.
func checkPatterns(value string, patterns map[string]int) int {
	score := 0
	for pattern, patternScore := range patterns {
		if strings.Contains(value, pattern) {
			score += patternScore
		}
	}
	return score
}

// MatchesPattern checks if value contains any of the patterns.
func MatchesPattern(value string, patterns map[string]bool) bool {
	for pattern := range patterns {
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
		switch attr.Key {
		case "class", "id":
			lowerVal := strings.ToLower(attr.Val)
			if MatchesPattern(lowerVal, removePatterns) {
				return true
			}
		case "style":
			lowerStyle := strings.ToLower(attr.Val)
			if strings.Contains(lowerStyle, "display:none") ||
				strings.Contains(lowerStyle, "display: none") ||
				strings.Contains(lowerStyle, "visibility:hidden") ||
				strings.Contains(lowerStyle, "visibility: hidden") {
				return true
			}
		case "hidden":
			return true
		}
	}
	return false
}

// IsBlockElement checks if tag is a block-level element.
func IsBlockElement(tag string) bool {
	return blockElements[tag]
}
