package html

import (
	"errors"
	"fmt"
)

// Sentinel errors for the `cybergodev/html` package.
// These errors can be used with errors.Is() for type checking.
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

	// ErrInternalPanic is returned when an unexpected panic occurs during processing.
	// This error indicates an internal bug and should be reported to the library maintainers.
	ErrInternalPanic = errors.New("html: internal panic recovered")
)

// InputError provides context for input-related errors.
// It supports errors.Is() checking against ErrInputTooLarge.
type InputError struct {
	Op       string // Operation that failed (e.g., "Extract", "ExtractFromFile")
	Size     int    // Actual input size
	MaxSize  int    // Maximum allowed size
	InputErr error  // Underlying error, if any
}

// Error returns a formatted error message.
func (e *InputError) Error() string {
	if e.InputErr != nil {
		return fmt.Sprintf("html: %s failed: %v (size=%d, max=%d)", e.Op, e.InputErr, e.Size, e.MaxSize)
	}
	return fmt.Sprintf("html: %s failed: input too large (size=%d, max=%d)", e.Op, e.Size, e.MaxSize)
}

// Unwrap returns the underlying error for errors.Is() support.
func (e *InputError) Unwrap() error {
	if e.InputErr != nil {
		return e.InputErr
	}
	return ErrInputTooLarge
}

// NewInputError creates a new InputError with the provided details.
func NewInputError(op string, size, maxSize int, err error) *InputError {
	return &InputError{
		Op:       op,
		Size:     size,
		MaxSize:  maxSize,
		InputErr: err,
	}
}

// ConfigError provides context for configuration errors.
// It supports errors.Is() checking against ErrInvalidConfig.
type ConfigError struct {
	Field   string // Configuration field that is invalid
	Value   any    // The invalid value
	Message string // Additional context about the error
}

// Error returns a formatted error message.
func (e *ConfigError) Error() string {
	return fmt.Sprintf("html: invalid config: %s=%v, %s", e.Field, e.Value, e.Message)
}

// Unwrap returns ErrInvalidConfig for errors.Is() support.
func (e *ConfigError) Unwrap() error {
	return ErrInvalidConfig
}

// NewConfigError creates a new ConfigError with the provided details.
func NewConfigError(field string, value any, message string) *ConfigError {
	return &ConfigError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// FileError provides context for file-related errors.
// It supports errors.Is() checking against ErrFileNotFound or ErrInvalidFilePath.
type FileError struct {
	Op      string // Operation that failed (e.g., "ExtractFromFile", "ReadFile")
	Path    string // File path that caused the error
	FileErr error  // Underlying error from file system
}

// Error returns a formatted error message.
func (e *FileError) Error() string {
	return fmt.Sprintf("html: %s %q: %v", e.Op, e.Path, e.FileErr)
}

// Unwrap returns the appropriate sentinel error for errors.Is() support.
func (e *FileError) Unwrap() error {
	if errors.Is(e.FileErr, ErrFileNotFound) {
		return ErrFileNotFound
	}
	return ErrInvalidFilePath
}

// NewFileError creates a new FileError with the provided details.
func NewFileError(op, path string, err error) *FileError {
	return &FileError{
		Op:      op,
		Path:    path,
		FileErr: err,
	}
}
