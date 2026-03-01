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
// The function supports multiple configuration patterns:
//
//	processor, err := html.New()                    // Uses DefaultConfig()
//	processor, err := html.New(config)              // Uses custom Config struct
//	processor, err := html.New(html.WithCache(1000, time.Hour))  // Uses functional options
//	processor, err := html.New(config, html.WithMaxDepth(200))   // Mix of both
//
// The returned processor must be closed when no longer needed:
//
//	processor, err := html.New()
//	defer processor.Close()
//
// With custom scorer:
//
//	processor, err := html.New(html.WithScorer(myScorer))
func New(opts ...any) (*Processor, error) {
	config := DefaultConfig()
	var customScorer Scorer

	for _, opt := range opts {
		switch v := opt.(type) {
		case Config:
			// Full Config struct provided
			config = v
		case Option:
			// Functional option provided
			if err := v(&config); err != nil {
				return nil, err
			}
		case Scorer:
			// Custom scorer provided
			customScorer = v
		default:
			return nil, fmt.Errorf("%w: unsupported option type %T", ErrInvalidConfig, opt)
		}
	}

	if err := validateConfig(config); err != nil {
		return nil, err
	}

	p := &Processor{
		config: &config,
		cache:  internal.NewCache(config.MaxCacheEntries, config.CacheTTL),
	}

	// Set up scorer
	if customScorer != nil {
		p.scorer = &internalScorerWrapper{scorer: customScorer}
	} else {
		p.scorer = internal.NewDefaultScorer()
	}

	return p, nil
}

// validateConfig validates the processor configuration.
// Deprecated: Use Config.Validate() directly.
func validateConfig(c Config) error {
	return c.Validate()
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
	return nil
}

// resolveExtractConfig resolves extraction configuration with defaults.
func resolveExtractConfig(configs ...ExtractConfig) ExtractConfig {
	if len(configs) > 0 {
		cfg := configs[0]
		// Validate and normalize TableFormat
		format := strings.ToLower(strings.TrimSpace(cfg.TableFormat))
		if format != "markdown" && format != "html" {
			format = "markdown"
		}
		cfg.TableFormat = format
		return cfg
	}
	return DefaultExtractConfig()
}

// resolveLinkExtractionConfig resolves link extraction configuration with defaults.
func resolveLinkExtractionConfig(configs ...LinkExtractionConfig) LinkExtractionConfig {
	if len(configs) > 0 {
		return configs[0]
	}
	return DefaultLinkExtractionConfig()
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
