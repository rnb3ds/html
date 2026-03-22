package html_test

// errors_test.go - Tests for error types and constructors
// This file tests all error types, their constructors, and errors.Is() support.

import (
	"errors"
	"strings"
	"testing"

	"github.com/cybergodev/html"
)

func TestInputError(t *testing.T) {
	t.Parallel()

	t.Run("error message format", func(t *testing.T) {
		err := html.NewInputError("Extract", 10000, 5000, nil)
		msg := err.Error()

		if msg == "" {
			t.Error("Error message should not be empty")
		}
		if !strings.Contains(msg, "Extract") {
			t.Errorf("Error message should contain operation: %s", msg)
		}
		if !strings.Contains(msg, "10000") {
			t.Errorf("Error message should contain size: %s", msg)
		}
		if !strings.Contains(msg, "5000") {
			t.Errorf("Error message should contain max: %s", msg)
		}
	})

	t.Run("error message with underlying error", func(t *testing.T) {
		underlying := errors.New("underlying error")
		err := html.NewInputError("Extract", 10000, 5000, underlying)
		msg := err.Error()

		if !strings.Contains(msg, "underlying error") {
			t.Errorf("Error message should contain underlying error: %s", msg)
		}
	})

	t.Run("field values", func(t *testing.T) {
		err := html.NewInputError("Op", 100, 50, nil)

		if err.Op != "Op" {
			t.Errorf("Op = %q, want 'Op'", err.Op)
		}
		if err.Size != 100 {
			t.Errorf("Size = %d, want 100", err.Size)
		}
		if err.MaxSize != 50 {
			t.Errorf("MaxSize = %d, want 50", err.MaxSize)
		}
	})
}

func TestInputErrorUnwrap(t *testing.T) {
	t.Parallel()

	t.Run("unwrap to ErrInputTooLarge", func(t *testing.T) {
		err := html.NewInputError("Extract", 10000, 5000, nil)
		if !errors.Is(err, html.ErrInputTooLarge) {
			t.Error("InputError should unwrap to ErrInputTooLarge")
		}
	})

	t.Run("unwrap with underlying error", func(t *testing.T) {
		underlying := errors.New("underlying")
		err := html.NewInputError("Extract", 10000, 5000, underlying)
		if !errors.Is(err, underlying) {
			t.Error("InputError should unwrap to underlying error")
		}
	})
}

func TestConfigError(t *testing.T) {
	t.Parallel()

	t.Run("error message format", func(t *testing.T) {
		err := html.NewConfigError("MaxDepth", 0, "must be positive")
		msg := err.Error()

		if msg == "" {
			t.Error("Error message should not be empty")
		}
		if !strings.Contains(msg, "MaxDepth") {
			t.Errorf("Error message should contain field: %s", msg)
		}
		if !strings.Contains(msg, "must be positive") {
			t.Errorf("Error message should contain message: %s", msg)
		}
	})

	t.Run("field values", func(t *testing.T) {
		err := html.NewConfigError("Field", "value", "test message")

		if err.Field != "Field" {
			t.Errorf("Field = %q, want 'Field'", err.Field)
		}
		if err.Value != "value" {
			t.Errorf("Value = %v, want 'value'", err.Value)
		}
		if err.Message != "test message" {
			t.Errorf("Message = %q, want 'test message'", err.Message)
		}
	})

	t.Run("different value types", func(t *testing.T) {
		tests := []struct {
			name  string
			field string
			value interface{}
		}{
			{"int value", "MaxDepth", 100},
			{"string value", "Encoding", "utf-8"},
			{"negative int", "MaxInputSize", -1},
			{"zero int", "WorkerPoolSize", 0},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := html.NewConfigError(tt.field, tt.value, "test")
				if err.Field != tt.field {
					t.Errorf("Field = %q, want %q", err.Field, tt.field)
				}
				if err.Value != tt.value {
					t.Errorf("Value = %v, want %v", err.Value, tt.value)
				}
			})
		}
	})
}

func TestConfigErrorUnwrap(t *testing.T) {
	t.Parallel()

	t.Run("unwrap to ErrInvalidConfig", func(t *testing.T) {
		err := html.NewConfigError("MaxDepth", 0, "must be positive")
		if !errors.Is(err, html.ErrInvalidConfig) {
			t.Error("ConfigError should unwrap to ErrInvalidConfig")
		}
	})
}

func TestFileError(t *testing.T) {
	t.Parallel()

	t.Run("error message format", func(t *testing.T) {
		err := html.NewFileError("ExtractFromFile", "/path/to/file.html", errors.New("not found"))
		msg := err.Error()

		if msg == "" {
			t.Error("Error message should not be empty")
		}
		if !strings.Contains(msg, "ExtractFromFile") {
			t.Errorf("Error message should contain operation: %s", msg)
		}
		if !strings.Contains(msg, "/path/to/file.html") {
			t.Errorf("Error message should contain path: %s", msg)
		}
	})

	t.Run("field values", func(t *testing.T) {
		underlying := errors.New("test error")
		err := html.NewFileError("Read", "/test/path", underlying)

		if err.Op != "Read" {
			t.Errorf("Op = %q, want 'Read'", err.Op)
		}
		if err.Path != "/test/path" {
			t.Errorf("Path = %q, want '/test/path'", err.Path)
		}
		if err.FileErr != underlying {
			t.Error("FileErr should be the underlying error")
		}
	})
}

func TestFileErrorUnwrap(t *testing.T) {
	t.Parallel()

	t.Run("unwrap to ErrInvalidFilePath", func(t *testing.T) {
		err := html.NewFileError("ExtractFromFile", "../traversal", errors.New("path error"))
		if !errors.Is(err, html.ErrInvalidFilePath) {
			t.Error("FileError should unwrap to ErrInvalidFilePath")
		}
	})

	t.Run("unwrap to ErrFileNotFound", func(t *testing.T) {
		err := html.NewFileError("ExtractFromFile", "missing.html", html.ErrFileNotFound)
		if !errors.Is(err, html.ErrFileNotFound) {
			t.Error("FileError should unwrap to ErrFileNotFound")
		}
	})
}

func TestNewFileErrorPathVariants(t *testing.T) {
	t.Parallel()

	paths := []struct {
		name string
		path string
	}{
		{"empty path", ""},
		{"relative path", "relative/path.html"},
		{"absolute path", "/absolute/path.html"},
		{"path with spaces", "/path with spaces/file.html"},
		{"windows path", `C:\Users\test\file.html`},
		{"traversal attempt", "../../../etc/passwd"},
	}

	for _, tt := range paths {
		t.Run(tt.name, func(t *testing.T) {
			err := html.NewFileError("Test", tt.path, errors.New("test"))
			if err.Path != tt.path {
				t.Errorf("Path = %q, want %q", err.Path, tt.path)
			}
		})
	}
}

// TestSentinelErrors verifies all sentinel errors exist and have meaningful messages
func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	sentinelErrors := []struct {
		name string
		err  error
		msg  string // Expected substring in error message
	}{
		{"ErrInputTooLarge", html.ErrInputTooLarge, "input"},
		{"ErrInvalidHTML", html.ErrInvalidHTML, "invalid"},
		{"ErrProcessorClosed", html.ErrProcessorClosed, "closed"},
		{"ErrMaxDepthExceeded", html.ErrMaxDepthExceeded, "depth"},
		{"ErrInvalidConfig", html.ErrInvalidConfig, "config"},
		{"ErrProcessingTimeout", html.ErrProcessingTimeout, "timeout"},
		{"ErrFileNotFound", html.ErrFileNotFound, "not found"},
		{"ErrInvalidFilePath", html.ErrInvalidFilePath, "path"},
	}

	for _, tt := range sentinelErrors {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("%s should have a message", tt.name)
			}
			if !strings.Contains(tt.err.Error(), tt.msg) {
				t.Errorf("%s message should contain %q, got %q", tt.name, tt.msg, tt.err.Error())
			}
		})
	}
}

func TestErrorIsUsage(t *testing.T) {
	t.Parallel()

	t.Run("errors.Is with InputError", func(t *testing.T) {
		err := html.NewInputError("Extract", 10000, 5000, nil)
		if !errors.Is(err, html.ErrInputTooLarge) {
			t.Error("errors.Is should match ErrInputTooLarge")
		}
	})

	t.Run("errors.Is with ConfigError", func(t *testing.T) {
		err := html.NewConfigError("Field", "value", "message")
		if !errors.Is(err, html.ErrInvalidConfig) {
			t.Error("errors.Is should match ErrInvalidConfig")
		}
	})

	t.Run("errors.Is with FileError", func(t *testing.T) {
		err := html.NewFileError("Op", "path", errors.New("test"))
		if !errors.Is(err, html.ErrInvalidFilePath) {
			t.Error("errors.Is should match ErrInvalidFilePath")
		}
	})

	t.Run("errors.Is negative case", func(t *testing.T) {
		err := html.NewConfigError("Field", "value", "message")
		if errors.Is(err, html.ErrInputTooLarge) {
			t.Error("ConfigError should not match ErrInputTooLarge")
		}
	})
}

func TestErrInternalPanic(t *testing.T) {
	t.Parallel()

	t.Run("sentinel error exists", func(t *testing.T) {
		if html.ErrInternalPanic == nil {
			t.Error("ErrInternalPanic should not be nil")
		}
		if html.ErrInternalPanic.Error() == "" {
			t.Error("ErrInternalPanic should have a message")
		}
	})

	t.Run("error message contains expected text", func(t *testing.T) {
		msg := html.ErrInternalPanic.Error()
		if !strings.Contains(msg, "panic") {
			t.Errorf("ErrInternalPanic message should contain 'panic': %s", msg)
		}
	})
}
