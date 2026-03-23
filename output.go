package html

import (
	"encoding/json"
)

// ExtractToMarkdown extracts content from HTML and returns it in Markdown format.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
// This method configures the extractor to use markdown format for inline images and links.
// Thread-safe: creates a config copy to avoid modifying shared state.
func (p *Processor) ExtractToMarkdown(htmlBytes []byte) (string, error) {
	result, err := p.extractWithFormats(htmlBytes, "markdown", "markdown")
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// ExtractToMarkdownFromFile extracts content from an HTML file and returns it in Markdown format.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
// This method configures the extractor to use markdown format for inline images and links.
// Thread-safe: creates a config copy to avoid modifying shared state.
func (p *Processor) ExtractToMarkdownFromFile(filePath string) (string, error) {
	result, err := p.extractFromFileWithFormats(filePath, "markdown", "markdown")
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
// This is a convenience function that uses a pooled Processor for efficiency.
//
// An optional Config can be provided to customize extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractToMarkdown(htmlBytes []byte, cfg ...Config) (string, error) {
	c, err := resolveConfig(cfg...)
	if err != nil {
		return "", err
	}
	processor, err := getProcessorWithConfig(c)
	if err != nil {
		return "", err
	}
	defer putProcessorWithConfig(processor, c)
	return processor.ExtractToMarkdown(htmlBytes)
}

// ExtractToMarkdownFromFile extracts content from an HTML file and returns it in Markdown format.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// An optional Config can be provided to customize extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractToMarkdownFromFile(filePath string, cfg ...Config) (string, error) {
	c, err := resolveConfig(cfg...)
	if err != nil {
		return "", err
	}
	processor, err := getProcessorWithConfig(c)
	if err != nil {
		return "", err
	}
	defer putProcessorWithConfig(processor, c)
	return processor.ExtractToMarkdownFromFile(filePath)
}

// ExtractToJSON extracts content from HTML and returns it as JSON.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// An optional Config can be provided to customize extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractToJSON(htmlBytes []byte, cfg ...Config) ([]byte, error) {
	c, err := resolveConfig(cfg...)
	if err != nil {
		return nil, err
	}
	processor, err := getProcessorWithConfig(c)
	if err != nil {
		return nil, err
	}
	defer putProcessorWithConfig(processor, c)
	return processor.ExtractToJSON(htmlBytes)
}

// ExtractToJSONFromFile extracts content from an HTML file and returns it as JSON.
// This is a convenience function that uses a pooled Processor for efficiency.
//
// An optional Config can be provided to customize extraction behavior.
// If no config is provided, DefaultConfig() is used.
func ExtractToJSONFromFile(filePath string, cfg ...Config) ([]byte, error) {
	c, err := resolveConfig(cfg...)
	if err != nil {
		return nil, err
	}
	processor, err := getProcessorWithConfig(c)
	if err != nil {
		return nil, err
	}
	defer putProcessorWithConfig(processor, c)
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

// extractWithFormats extracts content using temporary format settings.
// This creates a config copy to avoid race conditions when modifying format settings.
func (p *Processor) extractWithFormats(htmlBytes []byte, imageFormat, linkFormat string) (*Result, error) {
	if p == nil || p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	// Create a copy of config with modified format settings
	p.configMu.Lock()
	cfg := *p.config // Value copy to avoid race conditions
	p.configMu.Unlock()

	cfg.InlineImageFormat = imageFormat
	cfg.InlineLinkFormat = linkFormat

	// Create a temporary processor with the modified config
	tempP := &Processor{
		config: &cfg,
		cache:  p.cache,
		scorer: p.scorer,
		audit:  p.audit,
	}

	return tempP.extractCore(htmlBytes)
}

// extractFromFileWithFormats extracts content from file using temporary format settings.
func (p *Processor) extractFromFileWithFormats(filePath, imageFormat, linkFormat string) (*Result, error) {
	if p == nil || p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	data, err := p.validateAndReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return p.extractWithFormats(data, imageFormat, linkFormat)
}

// extractCore performs extraction without format modification logic.
// Used by extractWithFormats to avoid recursion through Extract.
//
// This method exists as a hook point for potential future extensions
// where internal extraction might need different behavior than the
// public Extract method. Currently it delegates directly to Extract.
func (p *Processor) extractCore(htmlBytes []byte) (*Result, error) {
	return p.Extract(htmlBytes)
}
