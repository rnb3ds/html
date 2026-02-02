package internal

import (
	"strings"

	"golang.org/x/net/html"
)

const (
	strongPositiveScore = 400
	mediumPositiveScore = 200
	strongNegativeScore = -400
	mediumNegativeScore = -200
	weakNegativeScore   = -100

	minParagraphsForBonus        = 3
	manyParagraphsMultiplier     = 150
	fewParagraphsMultiplier      = 80
	headingMultiplier            = 100
	veryLongTextThreshold        = 500
	longTextThreshold            = 200
	mediumTextThreshold          = 100
	shortTextThreshold           = 50
	veryLongTextBonusMultiplier  = 10
	longTextBonusDivider         = 2
	mediumTextBonusDivider       = 3
	shortTextPenalty              = -300
	highLinkDensityThreshold      = 0.5
	mediumLinkDensityThreshold   = 0.3
	lowLinkDensityThreshold      = 0.15
	highDensityMultiplier        = 1.2
	lowDensityMultiplier         = 0.7
	highLinkDensityPenalty       = 0.2
	mediumLinkDensityPenalty     = 0.5
	lowLinkDensityPenalty        = 0.75
	commaBonusThreshold          = 5
	commaBonusMultiplier         = 10
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

// ScoreContentNode calculates a relevance score for content extraction.
// Higher scores indicate more likely main content. Negative scores suggest non-content elements.
func ScoreContentNode(node *html.Node) int {
	if node == nil || node.Type != html.ElementNode || IsNonContentElement(node.Data) || node.Data == "p" {
		return 0
	}

	score := getTagScore(node.Data) + ScoreAttributes(node)

	// Score based on paragraph count
	paragraphCount := CountChildElements(node, "p")
	if paragraphCount >= minParagraphsForBonus {
		score += paragraphCount * manyParagraphsMultiplier
	} else if paragraphCount > 0 {
		score += paragraphCount * fewParagraphsMultiplier
	}

	// Score based on heading count
	headingCount := CountChildElements(node, "h1") + CountChildElements(node, "h2") +
		CountChildElements(node, "h3") + CountChildElements(node, "h4") +
		CountChildElements(node, "h5") + CountChildElements(node, "h6")
	if headingCount > 0 {
		score += headingCount * headingMultiplier
	}

	// Score based on text length
	textLength := GetTextLength(node)
	switch {
	case textLength > veryLongTextThreshold:
		score += veryLongTextThreshold + (textLength-veryLongTextThreshold)/veryLongTextBonusMultiplier
	case textLength > longTextThreshold:
		score += textLength >> 1
	case textLength > mediumTextThreshold:
		score += textLength / mediumTextBonusDivider
	case textLength < shortTextThreshold:
		score += shortTextPenalty
	}

	// Apply content density multiplier
	contentDensity := CalculateContentDensity(node)
	if contentDensity > 0.7 {
		score = int(float64(score) * highDensityMultiplier)
	} else if contentDensity < 0.3 {
		score = int(float64(score) * lowDensityMultiplier)
	}

	// Penalize high link density (likely navigation/spam)
	linkDensity := GetLinkDensity(node)
	if linkDensity > highLinkDensityThreshold {
		score = int(float64(score) * highLinkDensityPenalty)
	} else if linkDensity > mediumLinkDensityThreshold {
		score = int(float64(score) * mediumLinkDensityPenalty)
	} else if linkDensity > lowLinkDensityThreshold {
		score = int(float64(score) * lowLinkDensityPenalty)
	}

	// Bonus for comma-rich content (likely prose)
	commaCount := countCommas(node)
	if commaCount > commaBonusThreshold {
		score += commaCount * commaBonusMultiplier
	}

	return score
}

func getTagScore(tag string) int {
	return tagScores[tag]
}

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
		if patternMatches(value, pattern) {
			score += patternScore
		}
	}
	return score
}

func MatchesPattern(value string, patterns map[string]bool) bool {
	for pattern := range patterns {
		if patternMatches(value, pattern) {
			return true
		}
	}
	return false
}

// patternMatches checks if value matches pattern with word boundary detection
func patternMatches(value, pattern string) bool {
	idx := strings.Index(value, pattern)
	if idx == -1 {
		return false
	}

	// Check character before the match
	if idx > 0 {
		before := value[idx-1]
		if before != '-' && before != '_' && before != ' ' && before != '\t' {
			return false
		}
	}

	// Check character after the match
	endIdx := idx + len(pattern)
	if endIdx < len(value) {
		after := value[endIdx]
		if after != '-' && after != '_' && after != ' ' && after != '\t' {
			return false
		}
	}

	return true
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

func IsBlockElement(tag string) bool {
	return blockElements[tag]
}
