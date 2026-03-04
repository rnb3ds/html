package html

import (
	"encoding/json"
)

// ExtractToMarkdown extracts content from HTML and returns it in Markdown format.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
// This method configures the extractor to use markdown format for inline images.
// It uses the processor's configuration (cache, timeout, etc.) for extraction.
func (p *Processor) ExtractToMarkdown(htmlBytes []byte) (string, error) {
	config := p.getExtractConfig()
	config.InlineImageFormat = "markdown"
	result, err := p.extractInternal(htmlBytes, config)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// ExtractToMarkdownFromFile extracts content from an HTML file and returns it in Markdown format.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
// This method configures the extractor to use markdown format for inline images.
// It uses the processor's configuration (cache, timeout, etc.) for extraction.
func (p *Processor) ExtractToMarkdownFromFile(filePath string) (string, error) {
	config := p.getExtractConfig()
	config.InlineImageFormat = "markdown"
	result, err := p.extractFromFileWithConfig(filePath, config)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// ExtractToJSON extracts content from HTML and returns it as JSON.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
// This method uses the processor's configuration (cache, timeout, etc.) for extraction.
func (p *Processor) ExtractToJSON(htmlBytes []byte) ([]byte, error) {
	result, err := p.Extract(htmlBytes)
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

// ExtractToJSONFromFile extracts content from an HTML file and returns it as JSON.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
// This method uses the processor's configuration (cache, timeout, etc.) for extraction.
func (p *Processor) ExtractToJSONFromFile(filePath string) ([]byte, error) {
	result, err := p.ExtractFromFile(filePath)
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

// ============================================================================
// Package-level Convenience Functions
// ============================================================================

// ExtractToMarkdown extracts content from HTML and returns it in Markdown format.
// This is a convenience function that creates a temporary Processor with the given configuration.
// If no configuration is provided, DefaultConfig() is used.
//
// Example usage:
//
//	// Simple usage with default configuration
//	markdown, err := html.ExtractToMarkdown(htmlBytes)
//
//	// With custom configuration
//	cfg := html.DefaultConfig()
//	cfg.MaxInputSize = 10 * 1024 * 1024
//	markdown, err := html.ExtractToMarkdown(htmlBytes, cfg)
//
//	// Using preset configuration optimized for Markdown output
//	markdown, err := html.ExtractToMarkdown(htmlBytes, html.MarkdownConfig())
func ExtractToMarkdown(htmlBytes []byte, cfg ...Config) (string, error) {
	processor, err := New(resolveConfig(cfg...))
	if err != nil {
		return "", err
	}
	defer processor.Close()
	return processor.ExtractToMarkdown(htmlBytes)
}

// ExtractToMarkdownFromFile extracts content from an HTML file and returns it in Markdown format.
// This is a convenience function that creates a temporary Processor with the given configuration.
// If no configuration is provided, DefaultConfig() is used.
//
// Example usage:
//
//	// Simple usage with default configuration
//	markdown, err := html.ExtractToMarkdownFromFile("page.html")
//
//	// With custom configuration
//	cfg := html.DefaultConfig()
//	cfg.MaxInputSize = 10 * 1024 * 1024
//	markdown, err := html.ExtractToMarkdownFromFile("page.html", cfg)
//
//	// Using preset configuration optimized for Markdown output
//	markdown, err := html.ExtractToMarkdownFromFile("page.html", html.MarkdownConfig())
func ExtractToMarkdownFromFile(filePath string, cfg ...Config) (string, error) {
	processor, err := New(resolveConfig(cfg...))
	if err != nil {
		return "", err
	}
	defer processor.Close()
	return processor.ExtractToMarkdownFromFile(filePath)
}

// ExtractToJSON extracts content from HTML and returns it as JSON.
// This is a convenience function that creates a temporary Processor with the given configuration.
// If no configuration is provided, DefaultConfig() is used.
//
// Example usage:
//
//	// Simple usage with default configuration
//	jsonData, err := html.ExtractToJSON(htmlBytes)
//
//	// With custom configuration
//	cfg := html.DefaultConfig()
//	cfg.MaxInputSize = 10 * 1024 * 1024
//	jsonData, err := html.ExtractToJSON(htmlBytes, cfg)
//
//	// Using preset configuration
//	jsonData, err := html.ExtractToJSON(htmlBytes, html.TextOnlyConfig())
func ExtractToJSON(htmlBytes []byte, cfg ...Config) ([]byte, error) {
	processor, err := New(resolveConfig(cfg...))
	if err != nil {
		return nil, err
	}
	defer processor.Close()
	return processor.ExtractToJSON(htmlBytes)
}

// ExtractToJSONFromFile extracts content from an HTML file and returns it as JSON.
// This is a convenience function that creates a temporary Processor with the given configuration.
// If no configuration is provided, DefaultConfig() is used.
//
// Example usage:
//
//	// Simple usage with default configuration
//	jsonData, err := html.ExtractToJSONFromFile("page.html")
//
//	// With custom configuration
//	cfg := html.DefaultConfig()
//	cfg.MaxInputSize = 10 * 1024 * 1024
//	jsonData, err := html.ExtractToJSONFromFile("page.html", cfg)
//
//	// Using preset configuration
//	jsonData, err := html.ExtractToJSONFromFile("page.html", html.TextOnlyConfig())
func ExtractToJSONFromFile(filePath string, cfg ...Config) ([]byte, error) {
	processor, err := New(resolveConfig(cfg...))
	if err != nil {
		return nil, err
	}
	defer processor.Close()
	return processor.ExtractToJSONFromFile(filePath)
}

// jsonResult wraps Result for custom JSON marshaling with duration formatting.
type jsonResult struct {
	Text             string      `json:"text"`
	Title            string      `json:"title"`
	Images           []ImageInfo `json:"images,omitempty"`
	Links            []LinkInfo  `json:"links,omitempty"`
	Videos           []VideoInfo `json:"videos,omitempty"`
	Audios           []AudioInfo `json:"audios,omitempty"`
	ProcessingTimeMS int64       `json:"processing_time_ms"`
	WordCount        int         `json:"word_count"`
	ReadingTimeMS    int64       `json:"reading_time_ms"`
}

// MarshalJSON implements custom JSON marshaling for Result.
// It converts time.Duration fields to milliseconds for better JSON interoperability.
func (r *Result) MarshalJSON() ([]byte, error) {
	jr := jsonResult{
		Text:             r.Text,
		Title:            r.Title,
		Images:           r.Images,
		Links:            r.Links,
		Videos:           r.Videos,
		Audios:           r.Audios,
		ProcessingTimeMS: r.ProcessingTime.Milliseconds(),
		WordCount:        r.WordCount,
		ReadingTimeMS:    r.ReadingTime.Milliseconds(),
	}
	return json.Marshal(jr)
}
