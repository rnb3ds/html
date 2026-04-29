package html

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cybergodev/html/internal"
	stdxhtml "golang.org/x/net/html"
)

// Scorer defines the interface for content scoring algorithms.
// Implementations can provide custom scoring logic for content extraction.
// If no custom scorer is provided, the DefaultScorer is used.
//
// The ContentNode interface abstracts away the internal HTML parser types,
// allowing users to implement custom scorers without importing golang.org/x/net/html.
//
// # Architecture Notes
//
// This public interface uses ContentNode abstraction to hide the internal
// golang.org/x/net/html dependency. Internally, the scorerAdapter converts
// between this interface and internal.Scorer which uses *html.Node directly
// for performance. This dual-interface design provides:
//   - Clean public API (no external parser types exposed)
//   - High performance internally (direct node access)
//   - Flexibility for users to implement custom scoring
type Scorer interface {
	// Score calculates a relevance score for a content node.
	// Higher scores indicate more likely main content.
	Score(node ContentNode) int
	// ShouldRemove determines if a node should be removed from the content tree.
	ShouldRemove(node ContentNode) bool
}

// scorerAdapter adapts the public Scorer interface to the internal Scorer interface.
type scorerAdapter struct {
	external Scorer
}

func (a *scorerAdapter) Score(node *stdxhtml.Node) int {
	if a.external == nil || node == nil {
		return 0
	}
	return a.external.Score(contentNodeAdapter{node})
}

func (a *scorerAdapter) ShouldRemove(node *stdxhtml.Node) bool {
	if a.external == nil || node == nil {
		return false
	}
	return a.external.ShouldRemove(contentNodeAdapter{node})
}

// contentNodeAdapter adapts *stdxhtml.Node to ContentNode interface.
type contentNodeAdapter struct {
	*stdxhtml.Node
}

func (n contentNodeAdapter) Type() string {
	if n.Node == nil {
		return ""
	}
	switch n.Node.Type {
	case stdxhtml.ErrorNode:
		return "error"
	case stdxhtml.TextNode:
		return "text"
	case stdxhtml.DocumentNode:
		return "document"
	case stdxhtml.ElementNode:
		return "element"
	case stdxhtml.CommentNode:
		return "comment"
	case stdxhtml.DoctypeNode:
		return "doctype"
	case stdxhtml.RawNode:
		return "raw"
	default:
		return "unknown"
	}
}

func (n contentNodeAdapter) Data() string {
	if n.Node == nil {
		return ""
	}
	return n.Node.Data
}

func (n contentNodeAdapter) AttrValue(key string) string {
	if n.Node == nil {
		return ""
	}
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func (n contentNodeAdapter) Attrs() []NodeAttr {
	if n.Node == nil {
		return nil
	}
	attrs := make([]NodeAttr, len(n.Attr))
	for i, attr := range n.Attr {
		attrs[i] = NodeAttr{Key: attr.Key, Value: attr.Val}
	}
	return attrs
}

func (n contentNodeAdapter) FirstChild() ContentNode {
	if n.Node == nil || n.Node.FirstChild == nil {
		return nil
	}
	return contentNodeAdapter{n.Node.FirstChild}
}

func (n contentNodeAdapter) NextSibling() ContentNode {
	if n.Node == nil || n.Node.NextSibling == nil {
		return nil
	}
	return contentNodeAdapter{n.Node.NextSibling}
}

func (n contentNodeAdapter) Parent() ContentNode {
	if n.Node == nil || n.Node.Parent == nil {
		return nil
	}
	return contentNodeAdapter{n.Node.Parent}
}

// Processor is the main HTML processing engine.
// It provides methods for extracting content, links, and media from HTML documents
// with automatic encoding detection and caching support.
type Processor struct {
	config   *Config
	configMu sync.Mutex // Protects config fields during temporary modifications
	cache    *internal.Cache
	scorer   internal.Scorer
	audit    *auditCollector
	closed   atomic.Bool
	stats    *processorStats

	// Pre-computed format strings to avoid repeated strings.ToLower in hot path
	imageFormat string
	linkFormat  string
	// Cached audit adapter to avoid per-call allocation
	auditAdapter *auditRecorderAdapter
}

// processorStats holds thread-safe statistics counters shared between processors.
type processorStats struct {
	totalProcessed   atomic.Int64
	cacheHits        atomic.Int64
	cacheMisses      atomic.Int64
	errorCount       atomic.Int64
	totalProcessTime atomic.Int64
}

// New creates a new HTML processor with optional configuration.
// If no configuration is provided, DefaultConfig() is used.
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
//	cfg.InlineImageFormat = "markdown"
//	processor, err := html.New(cfg)
//
//	// Or use preset configurations
//	processor, err := html.New(html.HighSecurityConfig())
//
// To use a custom Scorer, set the Scorer field:
//
//	cfg := html.DefaultConfig()
//	cfg.Scorer = myScorer
//	processor, err := html.New(cfg)
func New(cfg ...Config) (*Processor, error) {
	c, _, err := resolveConfig(cfg...)
	if err != nil {
		return nil, err
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	p := &Processor{
		config: &c,
		cache:  internal.NewCache(c.MaxCacheEntries, c.CacheTTL),
		audit:  newAuditCollector(c.Audit),
		stats:  &processorStats{},
	}

	// Pre-compute normalized format strings to avoid repeated strings.ToLower in hot path
	p.imageFormat = strings.ToLower(strings.TrimSpace(c.InlineImageFormat))
	if p.imageFormat == "" {
		p.imageFormat = "none"
	}
	p.linkFormat = strings.ToLower(strings.TrimSpace(c.InlineLinkFormat))
	if p.linkFormat == "" {
		p.linkFormat = "none"
	}

	// Cache audit adapter to avoid per-call allocation
	p.auditAdapter = &auditRecorderAdapter{collector: p.audit}

	// Set up scorer from config
	// Note: Scorer interface uses ContentNode abstraction; adapter converts to internal.Scorer
	if c.Scorer != nil {
		p.scorer = &scorerAdapter{external: c.Scorer}
	} else {
		p.scorer = internal.SharedDefaultScorer()
	}

	// Start background cache cleanup if TTL and cleanup interval are configured
	if c.CacheTTL > 0 && c.CacheCleanup > 0 {
		p.cache.StartCleanup(c.CacheCleanup)
	}

	return p, nil
}

// GetStatistics returns current processing statistics.
func (p *Processor) GetStatistics() Statistics {
	if p == nil || p.stats == nil {
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
	if p == nil || p.stats == nil {
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
		_ = p.audit.Close() // best-effort cleanup
	}
	return nil
}

// validateInput performs common validation for HTML input.
// It checks for nil/closed processor and input size limits.
// Returns an error if validation fails, nil otherwise.
func (p *Processor) validateInput(htmlBytes []byte) error {
	if p == nil || p.closed.Load() {
		return ErrProcessorClosed
	}
	if len(htmlBytes) > p.config.MaxInputSize {
		if p.audit != nil {
			p.audit.RecordInputViolation(len(htmlBytes), p.config.MaxInputSize, "input_too_large")
		}
		p.stats.errorCount.Add(1)
		return newInputError("Extract", len(htmlBytes), p.config.MaxInputSize, nil)
	}
	return nil
}

// detectEncoding detects the character encoding and converts HTML bytes to UTF-8.
// This is a helper method used by multiple extraction methods to avoid code duplication.
// It records encoding issues to the audit log if enabled.
func (p *Processor) detectEncoding(htmlBytes []byte) (string, error) {
	utf8String, _, convErr := internal.DetectAndConvertToUTF8String(htmlBytes, p.config.Encoding)
	if convErr != nil {
		if p.audit != nil {
			p.audit.RecordEncodingIssue(p.config.Encoding, convErr.Error())
		}
		p.stats.errorCount.Add(1)
		return "", fmt.Errorf("encoding detection failed: %w", convErr)
	}
	return utf8String, nil
}

// validateAndReadFile validates the file path and reads the file contents.
// It performs security checks including path traversal detection.
// Returns the file contents or an appropriate error.
func (p *Processor) validateAndReadFile(filePath string) ([]byte, error) {
	// Validate file path
	if filePath == "" {
		return nil, newFileError("ReadFile", filePath, ErrInvalidFilePath)
	}

	// Clean the file path to resolve any "." or ".." components
	cleanPath := filepath.Clean(filePath)

	// After cleaning, check if the path contains parent directory references
	// This catches path traversal attempts like "../file", "subdir/../../file", etc.
	if strings.Contains(cleanPath, "..") {
		if p.audit != nil {
			p.audit.RecordPathTraversal(filePath)
		}
		return nil, newFileError("ReadFile", filePath, fmt.Errorf("path traversal detected: %s", cleanPath))
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, newFileError("ReadFile", cleanPath, ErrFileNotFound)
		}
		return nil, newFileError("ReadFile", cleanPath, err)
	}

	return data, nil
}
