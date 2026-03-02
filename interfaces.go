package html

import "context"

// Extractor defines the interface for HTML content extraction.
// This interface enables mocking and testing of extraction functionality
// without requiring a real Processor instance.
type Extractor interface {
	// Extract extracts content from HTML bytes with automatic encoding detection.
	Extract(htmlBytes []byte) (*Result, error)
	// ExtractFromFile extracts content from an HTML file with automatic encoding detection.
	ExtractFromFile(filePath string) (*Result, error)
	// ExtractText extracts plain text from HTML bytes with automatic encoding detection.
	ExtractText(htmlBytes []byte) (string, error)
	// ExtractTextFromFile extracts plain text from an HTML file with automatic encoding detection.
	ExtractTextFromFile(filePath string) (string, error)
	// ExtractToMarkdown extracts content from HTML and returns it in Markdown format.
	ExtractToMarkdown(htmlBytes []byte) (string, error)
	// ExtractToMarkdownFromFile extracts content from an HTML file and returns it in Markdown format.
	ExtractToMarkdownFromFile(filePath string) (string, error)
	// ExtractToJSON extracts content from HTML and returns it as JSON.
	ExtractToJSON(htmlBytes []byte) ([]byte, error)
	// ExtractToJSONFromFile extracts content from an HTML file and returns it as JSON.
	ExtractToJSONFromFile(filePath string) ([]byte, error)
	// ExtractBatch extracts content from multiple HTML byte slices concurrently.
	ExtractBatch(htmlContents [][]byte) ([]*Result, error)
	// ExtractBatchWithContext extracts content from multiple HTML byte slices concurrently with context support.
	ExtractBatchWithContext(ctx context.Context, htmlContents [][]byte) *BatchResult
	// ExtractBatchFiles extracts content from multiple HTML files concurrently.
	ExtractBatchFiles(filePaths []string) ([]*Result, error)
	// ExtractBatchFilesWithContext extracts content from multiple HTML files concurrently with context support.
	ExtractBatchFilesWithContext(ctx context.Context, filePaths []string) *BatchResult
	// Close releases resources used by the extractor.
	Close() error
}

// LinkExtractor defines the interface for link extraction from HTML content.
type LinkExtractor interface {
	// ExtractAllLinks extracts all links from HTML bytes with automatic encoding detection.
	ExtractAllLinks(htmlBytes []byte) ([]LinkResource, error)
	// ExtractAllLinksFromFile extracts all links from an HTML file with automatic encoding detection.
	ExtractAllLinksFromFile(filePath string) ([]LinkResource, error)
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

// ProcessorInterface combines all interfaces for complete Processor functionality.
// This can be used for dependency injection and comprehensive mocking.
type ProcessorInterface interface {
	Extractor
	LinkExtractor
	StatsProvider
}

// Compile-time assertions to ensure Processor implements all interfaces.
var (
	_ Extractor          = (*Processor)(nil)
	_ LinkExtractor      = (*Processor)(nil)
	_ StatsProvider      = (*Processor)(nil)
	_ ProcessorInterface = (*Processor)(nil)
)
