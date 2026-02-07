// Package internal provides centralized constant definitions for internal use.
package internal

const (
	// Cache configuration
	initialColWidthsCap = 12 // Initial capacity for table column widths

	// Content analysis thresholds
	cleanTextGrowthFactor = 2   // Factor for estimating cleaned text size
	builderInitialSize    = 256 // Initial capacity for strings.Builder

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
