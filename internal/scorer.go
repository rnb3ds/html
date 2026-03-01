// Package internal provides implementation details for the cybergodev/html library.
// This file contains the Scorer interface and default implementation for content scoring.
package internal

import (
	"strings"

	"golang.org/x/net/html"
)

// Scorer defines the interface for content scoring algorithms.
// Implementations can provide custom scoring logic for content extraction.
type Scorer interface {
	// Score calculates a relevance score for a content node.
	// Higher scores indicate more likely main content.
	Score(node *html.Node) int
	// ShouldRemove determines if a node should be removed from the content tree.
	ShouldRemove(node *html.Node) bool
}

// ScoringConfig holds the configuration for the default scorer.
type ScoringConfig struct {
	// PositiveStrongPatterns maps pattern strings to their strong positive scores.
	PositiveStrongPatterns map[string]int
	// PositiveMediumPatterns maps pattern strings to their medium positive scores.
	PositiveMediumPatterns map[string]int
	// NegativeStrongPatterns maps pattern strings to their strong negative scores.
	NegativeStrongPatterns map[string]int
	// NegativeMediumPatterns maps pattern strings to their medium negative scores.
	NegativeMediumPatterns map[string]int
	// NegativeWeakPatterns maps pattern strings to their weak negative scores.
	NegativeWeakPatterns map[string]int
	// RemovePatterns maps pattern strings to a boolean indicating removal.
	RemovePatterns map[string]bool
	// TagScores maps tag names to their base scores.
	TagScores map[string]int
}

// DefaultScoringConfig returns the default scoring configuration.
func DefaultScoringConfig() *ScoringConfig {
	return &ScoringConfig{
		PositiveStrongPatterns: map[string]int{
			"content": strongPositiveScore, "article": strongPositiveScore, "main": strongPositiveScore,
			"post": strongPositiveScore, "entry": strongPositiveScore, "text": strongPositiveScore,
			"body": strongPositiveScore, "story": strongPositiveScore,
		},
		PositiveMediumPatterns: map[string]int{
			"blog": mediumPositiveScore, "news": mediumPositiveScore,
			"detail": mediumPositiveScore, "page": mediumPositiveScore,
		},
		NegativeStrongPatterns: map[string]int{
			"comment": strongNegativeScore, "sidebar": strongNegativeScore, "nav": strongNegativeScore,
			"navigation": strongNegativeScore, "footer": strongNegativeScore, "header": strongNegativeScore,
			"menu": strongNegativeScore, "ad": strongNegativeScore, "advertisement": strongNegativeScore,
		},
		NegativeMediumPatterns: map[string]int{
			"widget": mediumNegativeScore, "related": mediumNegativeScore, "share": mediumNegativeScore,
			"social": mediumNegativeScore, "meta": mediumNegativeScore, "tag": mediumNegativeScore,
			"category": mediumNegativeScore,
		},
		NegativeWeakPatterns: map[string]int{
			"promo": weakNegativeScore, "banner": weakNegativeScore, "sponsor": weakNegativeScore,
		},
		RemovePatterns: map[string]bool{
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
		},
		TagScores: map[string]int{
			"article": 1000,
			"main":    900,
			"section": 300,
			"body":    100,
			"div":     50,
			"p":       0,
		},
	}
}

// DefaultScorer is the default implementation of the Scorer interface.
type DefaultScorer struct {
	config *ScoringConfig
}

// NewDefaultScorer creates a new DefaultScorer with the default configuration.
func NewDefaultScorer() *DefaultScorer {
	return &DefaultScorer{
		config: DefaultScoringConfig(),
	}
}

// NewDefaultScorerWithConfig creates a new DefaultScorer with custom configuration.
func NewDefaultScorerWithConfig(config *ScoringConfig) *DefaultScorer {
	return &DefaultScorer{
		config: config,
	}
}

// Score calculates a relevance score for a content node.
func (s *DefaultScorer) Score(node *html.Node) int {
	if node == nil || node.Type != html.ElementNode || IsNonContentElement(node.Data) || node.Data == "p" {
		return 0
	}

	score := s.getTagScore(node.Data) + s.scoreAttributes(node)

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

// ShouldRemove determines if a node should be removed from the content tree.
func (s *DefaultScorer) ShouldRemove(node *html.Node) bool {
	if node == nil || node.Type != html.ElementNode {
		return false
	}

	if IsNonContentElement(node.Data) {
		return true
	}

	for _, attr := range node.Attr {
		switch attr.Key {
		case "class", "id":
			lowerVal := strings.ToLower(attr.Val)
			for pattern := range s.config.RemovePatterns {
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

// getTagScore returns the base score for a tag name.
func (s *DefaultScorer) getTagScore(tag string) int {
	if score, ok := s.config.TagScores[tag]; ok {
		return score
	}
	return 0
}

// ScoreAttributes calculates a score based on element attributes.
// This is the public version for external use.
func (s *DefaultScorer) ScoreAttributes(n *html.Node) int {
	return s.scoreAttributes(n)
}

// scoreAttributes calculates a score based on element attributes.
func (s *DefaultScorer) scoreAttributes(n *html.Node) int {
	if n == nil {
		return 0
	}

	score := 0
	for _, attr := range n.Attr {
		switch attr.Key {
		case "class", "id":
			lowerVal := strings.ToLower(attr.Val)
			score += s.calculatePatternScore(lowerVal, s.config.PositiveStrongPatterns)
			score += s.calculatePatternScore(lowerVal, s.config.PositiveMediumPatterns)
			score += s.calculatePatternScore(lowerVal, s.config.NegativeStrongPatterns)
			score += s.calculatePatternScore(lowerVal, s.config.NegativeMediumPatterns)
			score += s.calculatePatternScore(lowerVal, s.config.NegativeWeakPatterns)
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

// calculatePatternScore calculates score based on pattern matching.
func (s *DefaultScorer) calculatePatternScore(value string, patterns map[string]int) int {
	score := 0
	for pattern, patternScore := range patterns {
		if hasWordBoundary(value, pattern, boundaryStandard) {
			score += patternScore
		}
	}
	return score
}

// defaultScorer is the global default scorer instance.
var defaultScorer = NewDefaultScorer()
