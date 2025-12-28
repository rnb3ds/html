package html

import "errors"

// Error definitions for the html package.
var (
	// ErrInputTooLarge is returned when input exceeds MaxInputSize.
	ErrInputTooLarge = errors.New("html: input size exceeds maximum")

	// ErrInvalidHTML is returned when HTML parsing fails.
	ErrInvalidHTML = errors.New("html: invalid HTML")

	// ErrProcessorClosed is returned when operations are attempted on a closed processor.
	ErrProcessorClosed = errors.New("html: processor closed")

	// ErrMaxDepthExceeded is returned when HTML nesting exceeds MaxDepth.
	ErrMaxDepthExceeded = errors.New("html: max depth exceeded")

	// ErrInvalidConfig is returned when configuration validation fails.
	ErrInvalidConfig = errors.New("html: invalid config")

	// ErrProcessingTimeout is returned when processing exceeds ProcessingTimeout.
	ErrProcessingTimeout = errors.New("html: processing timeout exceeded")

	// ErrEmptyInput is returned when input is empty or whitespace-only.
	ErrEmptyInput = errors.New("html: empty input")

	// ErrFileNotFound is returned when specified file cannot be read.
	ErrFileNotFound = errors.New("html: file not found")

	// ErrInvalidURL is returned when URL validation fails.
	ErrInvalidURL = errors.New("html: invalid URL")

	// ErrInvalidBaseURL is returned when base URL for relative link resolution is invalid.
	ErrInvalidBaseURL = errors.New("html: invalid base URL")
)
