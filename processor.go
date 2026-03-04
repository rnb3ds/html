package html

import (
	"fmt"
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
func New(cfg ...Config) (*Processor, error) {
	c := resolveConfig(cfg...)
	if err := c.Validate(); err != nil {
		return nil, err
	}

	p := &Processor{
		config: &c,
		cache:  internal.NewCache(c.MaxCacheEntries, c.CacheTTL),
		audit:  NewAuditCollector(c.Audit),
	}

	// Set up scorer from config
	if c.Scorer != nil {
		p.scorer = &internalScorerWrapper{scorer: c.Scorer}
	} else {
		p.scorer = internal.NewDefaultScorer()
	}

	// Start background cache cleanup if TTL and cleanup interval are configured
	if c.CacheTTL > 0 && c.CacheCleanup > 0 {
		p.cache.StartCleanup(c.CacheCleanup)
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
func resolveConfig(cfg ...Config) Config {
	// Fast path: no config provided
	if len(cfg) == 0 {
		return DefaultConfig()
	}

	c := cfg[0]

	// Check if config is empty (all zero values)
	// An empty config indicates the user wants default behavior
	if c.isEmpty() {
		return DefaultConfig()
	}

	return c
}

// isEmpty checks if the Config has all zero values.
// This is used to detect when a user wants default behavior
// but created an empty Config{} literal.
func (c Config) isEmpty() bool {
	return c.MaxInputSize == 0 &&
		c.MaxCacheEntries == 0 &&
		c.CacheTTL == 0 &&
		c.CacheCleanup == 0 &&
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
		c.LinkFormat == "" &&
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
	// Stop background cleanup goroutine if running
	p.cache.StopCleanup()
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
		InlineLinkFormat:  p.config.LinkFormat,
		TableFormat:       p.config.TableFormat,
		Encoding:          p.config.Encoding,
	}
}

// getLinkExtractionConfig returns the LinkExtractionConfig from the processor's unified Config.
func (p *Processor) getLinkExtractionConfig() LinkExtractionConfig {
	return p.config.LinkExtraction
}

// validateAndReadFile validates the file path and reads the file contents.
// It performs security checks including path traversal detection.
// Returns the file contents or an appropriate error.
func (p *Processor) validateAndReadFile(filePath string) ([]byte, error) {
	// Validate file path
	if filePath == "" {
		return nil, fmt.Errorf("%w: empty file path", ErrInvalidFilePath)
	}

	// Clean the file path to resolve any "." or ".." components
	cleanPath := filepathClean(filePath)

	// After cleaning, check if the path contains parent directory references
	// This catches path traversal attempts like "../file", "subdir/../../file", etc.
	if stringsContains(cleanPath, "..") {
		if p.audit != nil {
			p.audit.RecordPathTraversal(filePath)
		}
		return nil, fmt.Errorf("%w: path traversal detected: %s", ErrInvalidFilePath, cleanPath)
	}

	data, err := readFile(cleanPath)
	if err != nil {
		if osIsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, cleanPath)
		}
		return nil, fmt.Errorf("read file %q: %w", cleanPath, err)
	}

	return data, nil
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
