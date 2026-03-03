// Package internal provides centralized constant definitions for internal use.
package internal

const (
	// Content analysis thresholds
	// cleanTextGrowthFactor is the multiplier used to estimate the buffer size needed for
	// cleaned text output. A value of 2 provides adequate headroom for text expansion during
	// processing (e.g., entity replacement, newline normalization) while avoiding over-allocation.
	cleanTextGrowthFactor = 2
	builderInitialSize    = 256 // Initial capacity for strings.Builder

	// URL validation limits
	MaxURLLength     = 2000   // Maximum URL length
	MaxDataURILength = 100000 // Maximum data URL length (100KB)

	// Scoring constants
	strongPositiveScore = 400
	mediumPositiveScore = 200
	strongNegativeScore = -400
	mediumNegativeScore = -200
	weakNegativeScore   = -100

	minParagraphsForBonus       = 3
	manyParagraphsMultiplier    = 150
	fewParagraphsMultiplier     = 80
	headingMultiplier           = 100
	veryLongTextThreshold       = 500
	longTextThreshold           = 200
	mediumTextThreshold         = 100
	shortTextThreshold          = 50
	veryLongTextBonusMultiplier = 10
	longTextBonusDivider        = 2
	mediumTextBonusDivider      = 3
	shortTextPenalty            = -300
	highLinkDensityThreshold    = 0.5
	mediumLinkDensityThreshold  = 0.3
	lowLinkDensityThreshold     = 0.15
	highDensityMultiplier       = 1.2
	lowDensityMultiplier        = 0.7
	highLinkDensityPenalty      = 0.2
	mediumLinkDensityPenalty    = 0.5
	lowLinkDensityPenalty       = 0.75
	commaBonusThreshold         = 5
	commaBonusMultiplier        = 10
)
