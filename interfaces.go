package html

import "context"

// ============================================================================
// Core Interfaces
// ============================================================================

// Closer defines the interface for resource cleanup.
type Closer interface {
	// Close releases resources used by the extractor.
	Close() error
}

// ContentExtractor defines the interface for basic HTML content extraction.
// Use this interface when you only need the core extraction functionality.
type ContentExtractor interface {
	// Extract extracts content from HTML bytes with automatic encoding detection.
	Extract(htmlBytes []byte) (*Result, error)
	// ExtractWithContext extracts content with context support for cancellation.
	ExtractWithContext(ctx context.Context, htmlBytes []byte) (*Result, error)
}

// FileExtractor defines the interface for file-based HTML extraction.
// Use this interface when you only need file-based operations.
type FileExtractor interface {
	// ExtractFromFile extracts content from an HTML file.
	ExtractFromFile(filePath string) (*Result, error)
	// ExtractFromFileWithContext extracts content from a file with context support.
	ExtractFromFileWithContext(ctx context.Context, filePath string) (*Result, error)
}

// TextExtractor defines the interface for plain text extraction.
// Use this interface when you only need plain text output.
type TextExtractor interface {
	// ExtractText extracts plain text from HTML bytes.
	ExtractText(htmlBytes []byte) (string, error)
	// ExtractTextFromFile extracts plain text from an HTML file.
	ExtractTextFromFile(filePath string) (string, error)
}

// OutputExtractor defines the interface for formatted output extraction.
// Use this interface when you need Markdown or JSON output formats.
type OutputExtractor interface {
	// ExtractToMarkdown extracts content and returns it in Markdown format.
	ExtractToMarkdown(htmlBytes []byte) (string, error)
	// ExtractToMarkdownFromFile extracts content from a file and returns it in Markdown format.
	ExtractToMarkdownFromFile(filePath string) (string, error)
	// ExtractToJSON extracts content and returns it as JSON.
	ExtractToJSON(htmlBytes []byte) ([]byte, error)
	// ExtractToJSONFromFile extracts content from a file and returns it as JSON.
	ExtractToJSONFromFile(filePath string) ([]byte, error)
}

// BatchExtractor defines the interface for batch processing operations.
// Use this interface when you need to process multiple documents concurrently.
type BatchExtractor interface {
	// ExtractBatch extracts content from multiple HTML byte slices concurrently.
	ExtractBatch(htmlContents [][]byte) ([]*Result, error)
	// ExtractBatchWithContext extracts content with context support for cancellation.
	ExtractBatchWithContext(ctx context.Context, htmlContents [][]byte) *BatchResult
	// ExtractBatchFiles extracts content from multiple HTML files concurrently.
	ExtractBatchFiles(filePaths []string) ([]*Result, error)
	// ExtractBatchFilesWithContext extracts content from files with context support.
	ExtractBatchFilesWithContext(ctx context.Context, filePaths []string) *BatchResult
}

// Extractor defines the complete interface for HTML content extraction.
// This is the primary interface for mocking and testing.
// It combines all specialized interfaces for full functionality.
//
// For more focused interfaces, see:
//   - ContentExtractor: basic extraction operations
//   - FileExtractor: file-based operations
//   - TextExtractor: plain text output
//   - OutputExtractor: Markdown/JSON output
//   - BatchExtractor: batch processing
type Extractor interface {
	ContentExtractor
	FileExtractor
	TextExtractor
	OutputExtractor
	BatchExtractor
	Closer
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

// ============================================================================
// Compile-time Interface Assertions
// ============================================================================

// Compile-time assertions to ensure Processor implements all interfaces.
var (
	_ Closer           = (*Processor)(nil)
	_ ContentExtractor = (*Processor)(nil)
	_ FileExtractor    = (*Processor)(nil)
	_ TextExtractor    = (*Processor)(nil)
	_ OutputExtractor  = (*Processor)(nil)
	_ BatchExtractor   = (*Processor)(nil)
	_ Extractor        = (*Processor)(nil)
	_ LinkExtractor    = (*Processor)(nil)
	_ StatsProvider    = (*Processor)(nil)
)
