package html

import "context"

// Extractor defines the complete interface for HTML content extraction.
// This is the primary interface for mocking and testing.
//
// Package-level convenience functions are available for all Extractor methods,
// accepting an optional Config variadic parameter (cfg ...Config).
// When no config is provided, DefaultConfig() is used.
type Extractor interface {
	// Core extraction
	Extract(htmlBytes []byte) (*Result, error)
	ExtractWithContext(ctx context.Context, htmlBytes []byte) (*Result, error)
	ExtractFromFile(filePath string) (*Result, error)
	ExtractFromFileWithContext(ctx context.Context, filePath string) (*Result, error)

	// Text extraction
	ExtractText(htmlBytes []byte) (string, error)
	ExtractTextFromFile(filePath string) (string, error)
	ExtractTextWithContext(ctx context.Context, htmlBytes []byte) (string, error)
	ExtractTextFromFileWithContext(ctx context.Context, filePath string) (string, error)

	// Formatted output
	ExtractToMarkdown(htmlBytes []byte) (string, error)
	ExtractToMarkdownFromFile(filePath string) (string, error)
	ExtractToJSON(htmlBytes []byte) ([]byte, error)
	ExtractToJSONFromFile(filePath string) ([]byte, error)
	ExtractToMarkdownWithContext(ctx context.Context, htmlBytes []byte) (string, error)
	ExtractToMarkdownFromFileWithContext(ctx context.Context, filePath string) (string, error)
	ExtractToJSONWithContext(ctx context.Context, htmlBytes []byte) ([]byte, error)
	ExtractToJSONFromFileWithContext(ctx context.Context, filePath string) ([]byte, error)

	// Batch processing
	ExtractBatch(htmlContents [][]byte) *BatchResult
	ExtractBatchWithContext(ctx context.Context, htmlContents [][]byte) *BatchResult
	ExtractBatchFiles(filePaths []string) *BatchResult
	ExtractBatchFilesWithContext(ctx context.Context, filePaths []string) *BatchResult

	// Link extraction
	ExtractAllLinks(htmlBytes []byte) ([]LinkResource, error)
	ExtractAllLinksFromFile(filePath string) ([]LinkResource, error)
	ExtractAllLinksWithContext(ctx context.Context, htmlBytes []byte) ([]LinkResource, error)
	ExtractAllLinksFromFileWithContext(ctx context.Context, filePath string) ([]LinkResource, error)

	// Resource cleanup
	Close() error
}

// StatsProvider defines the interface for statistics and cache management.
type StatsProvider interface {
	// GetStatistics returns current processing statistics.
	GetStatistics() Statistics
	// ClearCache clears the cache contents but preserves cumulative statistics.
	ClearCache()
	// ResetStatistics resets all statistics counters to zero.
	ResetStatistics()
}

// Compile-time assertions to ensure Processor implements all interfaces.
var (
	_ Extractor     = (*Processor)(nil)
	_ StatsProvider = (*Processor)(nil)
)
