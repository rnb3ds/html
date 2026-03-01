// Package html provides HTML content extraction with automatic encoding detection.
// This file contains compatibility helpers for cross-platform support and testability.
//
// These wrapper functions exist primarily to enable mocking in unit tests.
// They provide thin wrappers around standard library functions, allowing
// tests to inject alternative implementations when needed.
package html

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// filepathClean is a cross-platform filepath.Clean wrapper.
func filepathClean(path string) string {
	return filepath.Clean(path)
}

// stringsContains is a wrapper for strings.Contains.
func stringsContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// osIsNotExist is a wrapper for os.IsNotExist.
func osIsNotExist(err error) bool {
	return os.IsNotExist(err)
}

// readFile reads a file and returns its contents.
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// now returns the current time.
func now() time.Time {
	return time.Now()
}

// since returns the time elapsed since t.
func since(t time.Time) time.Duration {
	return time.Since(t)
}
