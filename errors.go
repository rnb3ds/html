package html

import "errors"

// Error definitions for the `cybergodev/html` package.
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

	// ErrFileNotFound is returned when specified file cannot be read.
	ErrFileNotFound = errors.New("html: file not found")

	// ErrInvalidFilePath is returned when file path validation fails.
	ErrInvalidFilePath = errors.New("html: invalid file path")
)
