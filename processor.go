package html

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cybergodev/html/internal"
	stdxhtml "golang.org/x/net/html"
)

// Scorer defines the interface for content scoring algorithms.
// Implementations can provide custom scoring logic for content extraction.
// If no custom scorer is provided, the DefaultScorer is used.
type Scorer interface {
	// Score calculates a relevance score for a content node.
	// Higher scores indicate more likely main content.
	Score(node *Node) int
	// ShouldRemove determines if a node should be removed from the content tree.
	ShouldRemove(node *Node) bool
}

// internalScorer wraps a public Scorer to implement internal.Scorer.
type internalScorerWrapper struct {
	scorer Scorer
}

func (w *internalScorerWrapper) Score(node *stdxhtml.Node) int {
	return w.scorer.Score(node)
}

func (w *internalScorerWrapper) ShouldRemove(node *stdxhtml.Node) bool {
	return w.scorer.ShouldRemove(node)
}

// Processor is the main HTML processing engine.
// It provides methods for extracting content, links, and media from HTML documents
// with automatic encoding detection and caching support.
type Processor struct {
	config *Config
	cache  *internal.Cache
	scorer internal.Scorer
	audit  *AuditCollector
	closed atomic.Bool
	stats  struct {
		totalProcessed   atomic.Int64
		cacheHits        atomic.Int64
		cacheMisses      atomic.Int64
		errorCount       atomic.Int64
		totalProcessTime atomic.Int64
	}
}

// New creates a new HTML processor with the given configuration.
// If no configuration is provided, it uses DefaultConfig().
//
// The function supports the following usage patterns:
//
//	processor, err := html.New()          // Uses DefaultConfig()
//	processor, err := html.New(config)    // Uses custom Config struct
//
// To use a custom Scorer, set the Scorer field on the Config:
//
//	config := html.DefaultConfig()
//	config.Scorer = myScorer
//	processor, err := html.New(config)
//
// The returned processor must be closed when no longer needed:
//
//	processor, err := html.New()
//	defer processor.Close()
func New(config ...Config) (*Processor, error) {
	cfg := DefaultConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	p := &Processor{
		config: &cfg,
		cache:  internal.NewCache(cfg.MaxCacheEntries, cfg.CacheTTL),
		audit:  NewAuditCollector(cfg.Audit),
	}

	// Set up scorer from config
	if cfg.Scorer != nil {
		p.scorer = &internalScorerWrapper{scorer: cfg.Scorer}
	} else {
		p.scorer = internal.NewDefaultScorer()
	}

	return p, nil
}

// GetStatistics returns current processing statistics.
func (p *Processor) GetStatistics() Statistics {
	if p == nil {
		return Statistics{}
	}
	totalProcessed := p.stats.totalProcessed.Load()
	totalTime := time.Duration(p.stats.totalProcessTime.Load())
	var avgTime time.Duration
	if totalProcessed > 0 {
		avgTime = totalTime / time.Duration(totalProcessed)
	}
	return Statistics{
		TotalProcessed:     totalProcessed,
		CacheHits:          p.stats.cacheHits.Load(),
		CacheMisses:        p.stats.cacheMisses.Load(),
		ErrorCount:         p.stats.errorCount.Load(),
		AverageProcessTime: avgTime,
	}
}

// GetAuditLog returns the audit log entries collected during processing.
// Returns nil if audit logging is not enabled.
func (p *Processor) GetAuditLog() []AuditEntry {
	if p == nil || p.audit == nil {
		return nil
	}
	return p.audit.GetEntries()
}

// ClearAuditLog clears all collected audit log entries.
func (p *Processor) ClearAuditLog() {
	if p == nil || p.audit == nil {
		return
	}
	p.audit.Clear()
}

// ClearCache clears the cache contents but preserves cumulative statistics.
// Use ResetStatistics to reset statistics counters.
func (p *Processor) ClearCache() {
	if p == nil {
		return
	}
	p.cache.Clear()
}

// ResetStatistics resets all statistics counters to zero.
// This preserves cache entries while clearing the accumulated metrics.
func (p *Processor) ResetStatistics() {
	if p == nil {
		return
	}
	p.stats.cacheHits.Store(0)
	p.stats.cacheMisses.Store(0)
	p.stats.errorCount.Store(0)
	p.stats.totalProcessed.Store(0)
	p.stats.totalProcessTime.Store(0)
}

// Close releases resources used by the processor.
// After calling Close, the processor should not be used.
func (p *Processor) Close() error {
	if p == nil {
		return nil
	}
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	p.cache.Clear()
	if p.audit != nil {
		p.audit.Close()
	}
	return nil
}

// resolveExtractConfig resolves extraction configuration with defaults.
// It merges user-provided configuration with default values:
//   - If no config is provided, returns DefaultExtractConfig()
//   - If an empty config (all zero values) is provided, returns DefaultExtractConfig()
//   - If a config with only string fields set is provided, boolean fields use defaults
//   - If a config with any boolean field set to true is provided, use user's boolean values
//
// This design ensures that:
//   - Users can set individual string options without setting all booleans
//   - TextOnlyExtractConfig() works correctly (all booleans explicitly false)
//   - Empty config behaves the same as no config
//
// Example usage:
//
//	// This works - ExtractArticle defaults to true
//	result, err := p.Extract(data, html.ExtractConfig{
//	    InlineImageFormat: "html",
//	})
//
//	// TextOnlyExtractConfig works - explicitly disables media
//	result, err := p.Extract(data, html.TextOnlyExtractConfig())
func resolveExtractConfig(configs ...ExtractConfig) ExtractConfig {
	defaults := DefaultExtractConfig()

	if len(configs) == 0 {
		return defaults
	}

	cfg := configs[0]

	// Check if user passed an empty config (all zero values)
	// In this case, treat it as "use defaults"
	if cfg == (ExtractConfig{}) {
		return defaults
	}

	// Check if any boolean field is explicitly set to true
	// This indicates user intentionally modified the config
	hasExplicitTrue := cfg.ExtractArticle || cfg.PreserveImages ||
		cfg.PreserveLinks || cfg.PreserveVideos || cfg.PreserveAudios

	// Start with defaults
	result := defaults

	if hasExplicitTrue {
		// User explicitly set at least one boolean to true
		// Use all their boolean values (true or false)
		result.ExtractArticle = cfg.ExtractArticle
		result.PreserveImages = cfg.PreserveImages
		result.PreserveLinks = cfg.PreserveLinks
		result.PreserveVideos = cfg.PreserveVideos
		result.PreserveAudios = cfg.PreserveAudios
	}
	// else: all booleans are false, which means user only wanted to set string fields
	// Keep default true values for booleans

	// String fields - non-empty user value overrides defaults
	if cfg.InlineImageFormat != "" {
		result.InlineImageFormat = cfg.InlineImageFormat
	}
	if cfg.TableFormat != "" {
		result.TableFormat = cfg.TableFormat
	}
	if cfg.Encoding != "" {
		result.Encoding = cfg.Encoding
	}

	// Validate and normalize TableFormat
	format := strings.ToLower(strings.TrimSpace(result.TableFormat))
	if format != "markdown" && format != "html" {
		format = "markdown"
	}
	result.TableFormat = format

	// Validate and normalize InlineImageFormat
	imageFormat := strings.ToLower(strings.TrimSpace(result.InlineImageFormat))
	validFormats := map[string]bool{
		"none":        true,
		"markdown":    true,
		"html":        true,
		"placeholder": true,
	}
	if !validFormats[imageFormat] {
		imageFormat = "none"
	}
	result.InlineImageFormat = imageFormat

	return result
}

// resolveLinkExtractionConfig resolves link extraction configuration with defaults.
// It uses the smart merge strategy similar to resolveExtractConfig:
//   - If no config is provided, returns DefaultLinkExtractionConfig()
//   - If an empty config (all zero values) is provided, returns DefaultLinkExtractionConfig()
//   - If a config with only string fields set is provided, boolean fields use defaults
//   - If a config with any boolean field set to true is provided, use user's boolean values
func resolveLinkExtractionConfig(configs ...LinkExtractionConfig) LinkExtractionConfig {
	defaults := DefaultLinkExtractionConfig()

	if len(configs) == 0 {
		return defaults
	}

	cfg := configs[0]

	// Check if user passed an empty config (all zero values)
	if cfg == (LinkExtractionConfig{}) {
		return defaults
	}

	// Check if any boolean field is explicitly set to true
	hasExplicitTrue := cfg.ResolveRelativeURLs || cfg.IncludeImages ||
		cfg.IncludeVideos || cfg.IncludeAudios || cfg.IncludeCSS ||
		cfg.IncludeJS || cfg.IncludeContentLinks || cfg.IncludeExternalLinks ||
		cfg.IncludeIcons

	// Start with defaults
	result := defaults

	if hasExplicitTrue {
		// User explicitly set at least one boolean to true
		// Use all their boolean values
		result.ResolveRelativeURLs = cfg.ResolveRelativeURLs
		result.IncludeImages = cfg.IncludeImages
		result.IncludeVideos = cfg.IncludeVideos
		result.IncludeAudios = cfg.IncludeAudios
		result.IncludeCSS = cfg.IncludeCSS
		result.IncludeJS = cfg.IncludeJS
		result.IncludeContentLinks = cfg.IncludeContentLinks
		result.IncludeExternalLinks = cfg.IncludeExternalLinks
		result.IncludeIcons = cfg.IncludeIcons
	}
	// else: all booleans are false, which means user only wanted to set string fields
	// Keep default true values for booleans

	// String field - non-empty user value overrides
	if cfg.BaseURL != "" {
		result.BaseURL = cfg.BaseURL
	}

	return result
}

// collectResults collects batch processing results and returns the first error if any.
func collectResults(results []*Result, errs []error, names []string) ([]*Result, error) {
	var firstErr error
	successCount := 0
	failCount := 0

	for i, err := range errs {
		if err != nil {
			failCount++
			if firstErr == nil {
				if names != nil {
					firstErr = fmt.Errorf("%s: %w", names[i], err)
				} else {
					firstErr = fmt.Errorf("item %d: %w", i, err)
				}
			}
		} else {
			successCount++
		}
	}

	switch {
	case successCount == 0:
		return results, fmt.Errorf("all %d items failed: %w", len(results), firstErr)
	case failCount > 0:
		return results, fmt.Errorf("partial failure (%d/%d succeeded): %w", successCount, len(results), firstErr)
	default:
		return results, nil
	}
}

// GroupLinksByType groups links by their type.
func GroupLinksByType(links []LinkResource) map[string][]LinkResource {
	if len(links) == 0 {
		return make(map[string][]LinkResource)
	}

	grouped := make(map[string][]LinkResource, 8)
	for _, link := range links {
		if link.Type != "" {
			grouped[link.Type] = append(grouped[link.Type], link)
		} else {
			grouped["unknown"] = append(grouped["unknown"], link)
		}
	}
	return grouped
}
