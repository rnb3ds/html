package internal

import (
	"strings"

	"golang.org/x/net/html"
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
		"navigation": strongNegativeScore, "footer": strongNegativeScore, "header": strongNegativeScore,
		"menu": strongNegativeScore, "ad": strongNegativeScore, "advertisement": strongNegativeScore,
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
	// HTML5 inline elements - elements that should NOT add newlines or paragraph spacing
	// These elements flow with text on the same line
	inlineElements = map[string]bool{
		// Text formatting (presentational)
		"font": true, "b": true, "i": true, "u": true, "s": true, "strike": true,
		"del": true, "ins": true, "strong": true, "em": true,
		"mark": true, "small": true, "sub": true, "sup": true,
		"big": true, "tt": true,

		// Semantic inline
		"span": true, "a": true, "code": true, "kbd": true, "samp": true,
		"var": true, "abbr": true, "cite": true, "q": true, "dfn": true,
		"time": true, "data": true, "ruby": true, "rt": true, "rp": true,
		"bdi": true, "wbr": true,

		// Media and embedded
		"img": true, "svg": true, "picture": true,
		"video": true, "audio": true, "canvas": true,
		"object": true, "embed": true, "iframe": true,
		"map": true,

		// Form controls
		"input": true, "button": true, "select": true,
		"textarea": true, "label": true, "output": true,

		// Line break (special inline)
		"br": true,

		// Metadata (should not affect layout)
		"script": true, "style": true, "link": true, "meta": true, "title": true,
	}
	// HTML5 block elements - elements that should add newlines and paragraph spacing
	// Organized by category for better maintainability
	blockElements = map[string]bool{
		// Text containers
		"p": true, "div": true, "pre": true, "blockquote": true,

		// Headings
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,

		// Semantic HTML5 sections (high priority)
		"article": true, "section": true, "main": true, "nav": true, "aside": true,
		"header": true, "footer": true, "figure": true, "figcaption": true,

		// Lists
		"ul": true, "ol": true, "li": true, "dl": true, "dt": true, "dd": true,

		// Tables
		"table": true, "thead": true, "tbody": true, "tfoot": true, "tr": true, "td": true, "th": true,

		// Forms
		"form": true, "fieldset": true,

		// Interactive elements
		"details": true, "summary": true, "dialog": true,

		// Other block elements
		"hr": true, "address": true,

		// Structural elements (low priority, rarely appear in content extraction)
		"body": true, "html": true, "head": true,

		// Deprecated elements
		"center": true,

		// Media/Interactive elements
		"canvas": true,
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

// ScoreContentNode calculates a relevance score for content extraction.
// Higher scores indicate more likely main content. Negative scores suggest non-content elements.
// This function has been optimized to reduce DOM traversals by combining multiple metrics.
func ScoreContentNode(node *html.Node) int {
	if node == nil || node.Type != html.ElementNode || IsNonContentElement(node.Data) || node.Data == "p" {
		return 0
	}

	score := getTagScore(node.Data) + ScoreAttributes(node)

	// Collect all metrics in a single traversal
	metrics := collectContentMetrics(node)

	// Score based on paragraph count
	if metrics.paragraphCount >= minParagraphsForBonus {
		score += metrics.paragraphCount * manyParagraphsMultiplier
	} else if metrics.paragraphCount > 0 {
		score += metrics.paragraphCount * fewParagraphsMultiplier
	}

	// Score based on heading count
	if metrics.headingCount > 0 {
		score += metrics.headingCount * headingMultiplier
	}

	// Score based on text length
	textLength := metrics.textLength
	switch {
	case textLength > veryLongTextThreshold:
		score += veryLongTextThreshold + (textLength-veryLongTextThreshold)/veryLongTextBonusMultiplier
	case textLength > longTextThreshold:
		score += textLength / longTextBonusDivider
	case textLength > mediumTextThreshold:
		score += textLength / mediumTextBonusDivider
	case textLength < shortTextThreshold:
		score += shortTextPenalty
	}

	// Apply content density multiplier
	contentDensity := calculateDensityFromMetrics(metrics)
	if contentDensity > 0.7 {
		score = int(float64(score) * highDensityMultiplier)
	} else if contentDensity < 0.3 {
		score = int(float64(score) * lowDensityMultiplier)
	}

	// Penalize high link density (likely navigation/spam)
	linkDensity := calculateLinkDensityFromMetrics(metrics)
	if linkDensity > highLinkDensityThreshold {
		score = int(float64(score) * highLinkDensityPenalty)
	} else if linkDensity > mediumLinkDensityThreshold {
		score = int(float64(score) * mediumLinkDensityPenalty)
	} else if linkDensity > lowLinkDensityThreshold {
		score = int(float64(score) * lowLinkDensityPenalty)
	}

	// Bonus for comma-rich content (likely prose)
	if metrics.commaCount > commaBonusThreshold {
		score += metrics.commaCount * commaBonusMultiplier
	}

	return score
}

// contentMetrics holds all metrics collected during a single DOM traversal.
type contentMetrics struct {
	paragraphCount  int
	headingCount    int
	textLength      int
	linkTextLength  int
	totalTextLength int
	tagCount        int
	commaCount      int
}

// collectContentMetrics collects all scoring metrics in a single DOM traversal.
// This is more efficient than calling separate functions for each metric.
func collectContentMetrics(node *html.Node) contentMetrics {
	var metrics contentMetrics

	WalkNodes(node, func(n *html.Node) bool {
		if n.Type == html.ElementNode {
			metrics.tagCount++
			switch n.Data {
			case "p":
				metrics.paragraphCount++
			case "h1", "h2", "h3", "h4", "h5", "h6":
				metrics.headingCount++
			}
		} else if n.Type == html.TextNode {
			textData := normalizeNonBreakingSpaces(n.Data)
			text := strings.TrimSpace(textData)
			if text != "" {
				metrics.textLength += len(text)
				metrics.totalTextLength += len(text)
				metrics.commaCount += strings.Count(text, ",") + strings.Count(text, "，")

				// Check if this text is inside a link
				for parent := n.Parent; parent != nil; parent = parent.Parent {
					if parent.Type == html.ElementNode && parent.Data == "a" {
						metrics.linkTextLength += len(text)
						break
					}
				}
			}
		}
		return true
	})

	return metrics
}

// calculateDensityFromMetrics calculates content density from collected metrics.
func calculateDensityFromMetrics(m contentMetrics) float64 {
	if m.textLength == 0 {
		return 0
	}
	if m.tagCount == 0 {
		return 1.0
	}
	density := float64(m.textLength) / (float64(m.tagCount) * 10)
	if density > 1.0 {
		return 1.0
	}
	return density
}

// calculateLinkDensityFromMetrics calculates link density from collected metrics.
func calculateLinkDensityFromMetrics(m contentMetrics) float64 {
	if m.totalTextLength == 0 {
		return 0.0
	}
	return float64(m.linkTextLength) / float64(m.totalTextLength)
}

func getTagScore(tag string) int {
	return tagScores[tag]
}

func ScoreAttributes(n *html.Node) int {
	if n == nil {
		return 0
	}

	score := 0
	for _, attr := range n.Attr {
		switch attr.Key {
		case "class", "id":
			lowerVal := strings.ToLower(attr.Val)
			score += calculatePatternScore(lowerVal, positiveStrongPatterns)
			score += calculatePatternScore(lowerVal, positiveMediumPatterns)
			score += calculatePatternScore(lowerVal, negativeStrongPatterns)
			score += calculatePatternScore(lowerVal, negativeMediumPatterns)
			score += calculatePatternScore(lowerVal, negativeWeakPatterns)
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

func calculatePatternScore(value string, patterns map[string]int) int {
	score := 0
	for pattern, patternScore := range patterns {
		if hasWordBoundary(value, pattern, boundaryStandard) {
			score += patternScore
		}
	}
	return score
}

// matchesPattern checks if value contains any pattern from the map with word boundaries.
// This is an internal helper used by ShouldRemoveElement.
func matchesPattern(value string, patterns map[string]bool) bool {
	for pattern := range patterns {
		if hasWordBoundary(value, pattern, boundaryStandard) {
			return true
		}
	}
	return false
}

// MatchesPattern is the exported version of matchesPattern for testing purposes.
// It checks if value contains any pattern from the map with word boundaries.
func MatchesPattern(value string, patterns map[string]bool) bool {
	return matchesPattern(value, patterns)
}

// CalculateContentDensity calculates text-to-tag ratio.
// This is the exported version that uses the internal calculateDensityFromMetrics.
func CalculateContentDensity(n *html.Node) float64 {
	if n == nil {
		return 0
	}
	metrics := collectContentMetrics(n)
	return calculateDensityFromMetrics(metrics)
}

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
			for pattern := range removePatterns {
				if hasWordBoundary(lowerVal, pattern, boundaryStandard) {
					return true
				}
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

func IsBlockElement(tag string) bool {
	return blockElements[tag]
}

// IsInlineElement returns true if the tag is a known inline element.
// Inline elements should not add newlines or paragraph spacing.
func IsInlineElement(tag string) bool {
	return inlineElements[tag]
}
