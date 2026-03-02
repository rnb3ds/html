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

// New creates a new HTML processor with optional configuration.
// If no configuration is provided or an empty config is given, DefaultConfig() is used.
//
// Example usage:
//
//	// Simple usage with default configuration
//	processor, err := html.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer processor.Close()
//
//	// With custom configuration
//	cfg := html.DefaultConfig()
//	cfg.MaxInputSize = 10 * 1024 * 1024
//	processor, err := html.New(cfg)
//
//	// Or use preset configurations
//	processor, err := html.New(html.MarkdownConfig())
//	processor, err := html.New(html.HighSecurityConfig())
//
// To use a custom Scorer, set the Scorer field on the Config:
//
//	cfg := html.DefaultConfig()
//	cfg.Scorer = myScorer
//	processor, err := html.New(cfg)
func New(cfgs ...Config) (*Processor, error) {
	cfg := resolveConfig(cfgs...)
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

// resolveConfig resolves the configuration with smart defaults.
// It handles the following cases:
//   - No config provided: returns DefaultConfig()
//   - Empty config (all zero values): returns DefaultConfig()
//   - Valid config provided: returns the provided config
//
// This design allows both simple usage (html.New()) and custom configuration
// (html.New(cfg)) while maintaining backward compatibility.
func resolveConfig(cfgs ...Config) Config {
	// Fast path: no config provided
	if len(cfgs) == 0 {
		return DefaultConfig()
	}

	cfg := cfgs[0]

	// Check if config is empty (all zero values)
	// An empty config indicates the user wants default behavior
	if cfg.isEmpty() {
		return DefaultConfig()
	}

	return cfg
}

// isEmpty checks if the Config has all zero values.
// This is used to detect when a user wants default behavior
// but created an empty Config{} literal.
func (c Config) isEmpty() bool {
	return c.MaxInputSize == 0 &&
		c.MaxCacheEntries == 0 &&
		c.CacheTTL == 0 &&
		c.WorkerPoolSize == 0 &&
		c.MaxDepth == 0 &&
		c.ProcessingTimeout == 0 &&
		!c.EnableSanitization &&
		!c.ExtractArticle &&
		!c.PreserveImages &&
		!c.PreserveLinks &&
		!c.PreserveVideos &&
		!c.PreserveAudios &&
		c.ImageFormat == "" &&
		c.TableFormat == "" &&
		c.Encoding == "" &&
		c.Scorer == nil &&
		c.Audit.isEmpty()
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

// getExtractConfig returns the ExtractConfig from the processor's unified Config.
func (p *Processor) getExtractConfig() ExtractConfig {
	return ExtractConfig{
		ExtractArticle:    p.config.ExtractArticle,
		PreserveImages:    p.config.PreserveImages,
		PreserveLinks:     p.config.PreserveLinks,
		PreserveVideos:    p.config.PreserveVideos,
		PreserveAudios:    p.config.PreserveAudios,
		InlineImageFormat: p.config.ImageFormat,
		TableFormat:       p.config.TableFormat,
		Encoding:          p.config.Encoding,
	}
}

// getLinkExtractionConfig returns the LinkExtractionConfig from the processor's unified Config.
func (p *Processor) getLinkExtractionConfig() LinkExtractionConfig {
	return p.config.LinkExtraction
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
func resolveExtractConfig(configs ...ExtractConfig) ExtractConfig {
	if len(configs) == 0 {
		return defaultExtractConfig
	}

	cfg := configs[0]
	if cfg.isEmpty() {
		return defaultExtractConfig
	}

	// Determine boolean override behavior
	result := defaultExtractConfig
	if cfg.hasExplicitBoolean() {
		result.applyBooleanOverrides(cfg)
	}
	result.applyStringOverrides(cfg)
	return result
}

// isEmpty checks if config has all zero values
func (r ExtractConfig) isEmpty() bool {
	return !r.ExtractArticle && !r.PreserveImages && !r.PreserveLinks &&
		!r.PreserveVideos && !r.PreserveAudios &&
		r.InlineImageFormat == "" && r.TableFormat == "" && r.Encoding == ""
}

// hasExplicitBoolean checks if any boolean field is explicitly true
func (r ExtractConfig) hasExplicitBoolean() bool {
	return r.ExtractArticle || r.PreserveImages || r.PreserveLinks || r.PreserveVideos || r.PreserveAudios
}

// applyBooleanOverrides applies user's boolean values to result
func (r *ExtractConfig) applyBooleanOverrides(cfg ExtractConfig) {
	r.ExtractArticle = cfg.ExtractArticle
	r.PreserveImages = cfg.PreserveImages
	r.PreserveLinks = cfg.PreserveLinks
	r.PreserveVideos = cfg.PreserveVideos
	r.PreserveAudios = cfg.PreserveAudios
}

// applyStringOverrides applies user's string values to result
func (r *ExtractConfig) applyStringOverrides(cfg ExtractConfig) {
	if cfg.InlineImageFormat != "" {
		r.InlineImageFormat = normalizeImageFormat(cfg.InlineImageFormat)
	}
	if cfg.TableFormat != "" {
		r.TableFormat = normalizeTableFormat(cfg.TableFormat)
	}
	if cfg.Encoding != "" {
		r.Encoding = cfg.Encoding
	}
}

// normalizeTableFormat validates and normalizes table format.
// Uses inline comparison to avoid map allocations.
func normalizeTableFormat(format string) string {
	// Fast lowercase for common formats
	switch format {
	case "markdown", "Markdown", "MARKDOWN":
		return "markdown"
	case "html", "HTML", "Html":
		return "html"
	default:
		// Slow path: full normalization
		f := strings.ToLower(strings.TrimSpace(format))
		if f == "html" {
			return "html"
		}
		return "markdown"
	}
}

// normalizeImageFormat validates and normalizes image format.
// Uses inline comparison to avoid map allocations.
func normalizeImageFormat(format string) string {
	// Fast path for common formats
	switch format {
	case "none", "None", "NONE":
		return "none"
	case "markdown", "Markdown", "MARKDOWN":
		return "markdown"
	case "html", "HTML", "Html":
		return "html"
	case "placeholder", "Placeholder", "PLACEHOLDER":
		return "placeholder"
	default:
		// Slow path: full normalization
		f := strings.ToLower(strings.TrimSpace(format))
		switch f {
		case "markdown", "html", "placeholder":
			return f
		default:
			return "none"
		}
	}
}

// resolveLinkExtractionConfig resolves link extraction configuration with defaults.
// It uses the smart merge strategy similar to resolveExtractConfig:
//   - If no config is provided, returns DefaultLinkExtractionConfig()
//   - If an empty config (all zero values) is provided, returns DefaultLinkExtractionConfig()
//   - If a config with only string fields set is provided, boolean fields use defaults
//   - If a config with any boolean field set to true is provided, use user's boolean values
func resolveLinkExtractionConfig(configs ...LinkExtractionConfig) LinkExtractionConfig {
	// Fast path: no config provided
	if len(configs) == 0 {
		return defaultLinkExtractionConfig
	}

	cfg := configs[0]

	// Fast path: empty config (all zero values)
	if cfg.ResolveRelativeURLs == false && cfg.IncludeImages == false &&
		cfg.IncludeVideos == false && cfg.IncludeAudios == false &&
		cfg.IncludeCSS == false && cfg.IncludeJS == false &&
		cfg.IncludeContentLinks == false && cfg.IncludeExternalLinks == false &&
		cfg.IncludeIcons == false && cfg.BaseURL == "" {
		return defaultLinkExtractionConfig
	}

	// Check if any boolean field is explicitly set to true
	hasExplicitTrue := cfg.ResolveRelativeURLs || cfg.IncludeImages ||
		cfg.IncludeVideos || cfg.IncludeAudios || cfg.IncludeCSS ||
		cfg.IncludeJS || cfg.IncludeContentLinks || cfg.IncludeExternalLinks ||
		cfg.IncludeIcons

	// Start with defaults
	result := defaultLinkExtractionConfig

	if hasExplicitTrue {
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

	// String field - non-empty user value overrides
	if cfg.BaseURL != "" {
		result.BaseURL = cfg.BaseURL
	}

	return result
}

// collectResults collects batch processing results and returns the first error if any.
func collectResults(results []*Result, errs []error, names []string) ([]*Result, error) {
	var firstErr error
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
		}
	}

	switch failCount {
	case 0:
		return results, nil
	case len(errs):
		return results, fmt.Errorf("all %d items failed: %w", len(results), firstErr)
	default:
		return results, fmt.Errorf("partial failure (%d/%d failed): %w", failCount, len(results), firstErr)
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
