package html

import (
	"encoding/json"
	"fmt"
)

// ExtractToMarkdown extracts content from HTML and returns it in Markdown format.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration (cache, timeout, etc.), use
// Processor.ExtractToMarkdown instead.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
func ExtractToMarkdown(htmlBytes []byte, configs ...ExtractConfig) (string, error) {
	processor, err := New()
	if err != nil {
		return "", fmt.Errorf("create processor: %w", err)
	}
	defer processor.Close()
	return processor.ExtractToMarkdown(htmlBytes, configs...)
}

// ExtractToMarkdown extracts content from HTML and returns it in Markdown format.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
// This method configures the extractor to use markdown format for inline images.
// It uses the processor's configuration (cache, timeout, etc.) for extraction.
func (p *Processor) ExtractToMarkdown(htmlBytes []byte, configs ...ExtractConfig) (string, error) {
	config := resolveExtractConfig(configs...)
	config.InlineImageFormat = "markdown"
	result, err := p.Extract(htmlBytes, config)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// ExtractToMarkdownFromFile extracts content from an HTML file and returns it in Markdown format.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration (cache, timeout, etc.), use
// Processor.ExtractToMarkdownFromFile instead.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
func ExtractToMarkdownFromFile(filePath string, configs ...ExtractConfig) (string, error) {
	processor, err := New()
	if err != nil {
		return "", fmt.Errorf("create processor: %w", err)
	}
	defer processor.Close()
	return processor.ExtractToMarkdownFromFile(filePath, configs...)
}

// ExtractToMarkdownFromFile extracts content from an HTML file and returns it in Markdown format.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
// This method configures the extractor to use markdown format for inline images.
// It uses the processor's configuration (cache, timeout, etc.) for extraction.
func (p *Processor) ExtractToMarkdownFromFile(filePath string, configs ...ExtractConfig) (string, error) {
	config := resolveExtractConfig(configs...)
	config.InlineImageFormat = "markdown"
	result, err := p.ExtractFromFile(filePath, config)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// ExtractToJSON extracts content from HTML and returns it as JSON.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration (cache, timeout, etc.), use
// Processor.ExtractToJSON instead.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
func ExtractToJSON(htmlBytes []byte, configs ...ExtractConfig) ([]byte, error) {
	processor, err := New()
	if err != nil {
		return nil, fmt.Errorf("create processor: %w", err)
	}
	defer processor.Close()
	return processor.ExtractToJSON(htmlBytes, configs...)
}

// ExtractToJSON extracts content from HTML and returns it as JSON.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML bytes and converts it to UTF-8 before processing.
// This method uses the processor's configuration (cache, timeout, etc.) for extraction.
func (p *Processor) ExtractToJSON(htmlBytes []byte, configs ...ExtractConfig) ([]byte, error) {
	result, err := p.Extract(htmlBytes, configs...)
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

// ExtractToJSONFromFile extracts content from an HTML file and returns it as JSON.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration (cache, timeout, etc.), use
// Processor.ExtractToJSONFromFile instead.
//
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
func ExtractToJSONFromFile(filePath string, configs ...ExtractConfig) ([]byte, error) {
	processor, err := New()
	if err != nil {
		return nil, fmt.Errorf("create processor: %w", err)
	}
	defer processor.Close()
	return processor.ExtractToJSONFromFile(filePath, configs...)
}

// ExtractToJSONFromFile extracts content from an HTML file and returns it as JSON.
// The method automatically detects the character encoding (Windows-1252, UTF-8, GBK, Shift_JIS, etc.)
// from the HTML file and converts it to UTF-8 before processing.
// This method uses the processor's configuration (cache, timeout, etc.) for extraction.
func (p *Processor) ExtractToJSONFromFile(filePath string, configs ...ExtractConfig) ([]byte, error) {
	result, err := p.ExtractFromFile(filePath, configs...)
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
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
