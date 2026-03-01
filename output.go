package html

import (
	"encoding/json"
)

// ExtractToMarkdown extracts content from HTML and returns it in Markdown format.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration, use Processor.ExtractToMarkdown instead.
func ExtractToMarkdown(htmlBytes []byte, configs ...ExtractConfig) (string, error) {
	processor, _ := New()
	defer processor.Close()
	return processor.ExtractToMarkdown(htmlBytes, configs...)
}

// ExtractToMarkdown extracts content from HTML and returns it in Markdown format.
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

// ExtractToJSON extracts content from HTML and returns it as JSON.
// This is a convenience function that creates a temporary Processor with default settings.
// For repeated extractions or custom configuration, use Processor.ExtractToJSON instead.
func ExtractToJSON(htmlBytes []byte, configs ...ExtractConfig) ([]byte, error) {
	processor, _ := New()
	defer processor.Close()
	return processor.ExtractToJSON(htmlBytes, configs...)
}

// ExtractToJSON extracts content from HTML and returns it as JSON.
// This method uses the processor's configuration (cache, timeout, etc.) for extraction.
func (p *Processor) ExtractToJSON(htmlBytes []byte, configs ...ExtractConfig) ([]byte, error) {
	result, err := p.Extract(htmlBytes, configs...)
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
