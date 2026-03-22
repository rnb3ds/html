// Package html_test provides fuzz tests for the html package.
// These tests verify that the parser handles arbitrary input without panicking.
package html_test

import (
	"strings"
	"testing"

	"github.com/cybergodev/html"
)

// FuzzExtract tests that Extract handles arbitrary input without panicking.
func FuzzExtract(f *testing.F) {
	// Seed corpus with valid and edge-case inputs
	seeds := []string{
		`<html><body><p>Valid HTML</p></body></html>`,
		"",
		"<>",
		"<html><body>",
		"<div>Unclosed",
		"<script>alert(1)</script>",
		strings.Repeat("x", 1024*1024),   // 1MB of data
		"\x00\x01\x02\x03",               // Binary data
		string([]byte{0xFF, 0xFE, 0xFD}), // Invalid UTF-8
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		p, err := html.New()
		if err != nil {
			t.Fatalf("Failed to create processor: %v", err)
		}
		defer p.Close()

		// Should never panic
		result, err := p.Extract([]byte(input))

		// If no error, result should be valid
		if err == nil && result == nil {
			t.Error("Nil result with no error")
		}
	})
}

// FuzzEncodingDetection tests encoding detection robustness.
func FuzzEncodingDetection(f *testing.F) {
	seeds := [][]byte{
		[]byte(`<meta charset="utf-8"><body>Test</body>`),
		[]byte(`<meta http-equiv="Content-Type" content="text/html; charset=iso-8859-1">`),
		{0xEF, 0xBB, 0xBF, '<', 'h', 't', 'm', 'l', '>'}, // UTF-8 BOM
		{0xFF, 0xFE, '<', 0, 'h', 0},                     // UTF-16 LE BOM
		[]byte(`<html><body>Simple content</body></html>`),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		p, err := html.New()
		if err != nil {
			t.Fatalf("Failed to create processor: %v", err)
		}
		defer p.Close()

		// Should handle any byte sequence without panic
		_, _ = p.Extract(input)
	})
}

// FuzzTableParsing tests table extraction robustness.
func FuzzTableParsing(f *testing.F) {
	seeds := []string{
		`<table><tr><td>Cell</td></tr></table>`,
		`<table><tr><td colspan="2">Span</td></tr></table>`,
		`<table><tr><td rowspan="999">Row</td></tr></table>`,
		`<table><tr><td width="1000000px">Wide</td></tr></table>`,
		`<table><tr></tr></table>`,
		`<table></table>`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, tableHTML string) {
		cfg := html.DefaultConfig()
		cfg.TableFormat = "markdown"
		p, err := html.New(cfg)
		if err != nil {
			t.Fatalf("Failed to create processor: %v", err)
		}
		defer p.Close()

		htmlContent := `<html><body>` + tableHTML + `</body></html>`
		result, _ := p.Extract([]byte(htmlContent))

		// Result text should not contain unescaped HTML tags if table was processed
		if result != nil && strings.Contains(result.Text, "<table>") {
			t.Errorf("Markdown output contains HTML tags")
		}
	})
}

// FuzzExtractAllLinks tests link extraction robustness.
func FuzzExtractAllLinks(f *testing.F) {
	seeds := []string{
		`<a href="https://example.com">Link</a>`,
		`<img src="image.jpg">`,
		`<script src="script.js"></script>`,
		`<link rel="stylesheet" href="style.css">`,
		`<a href="javascript:alert(1)">XSS</a>`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		p, err := html.New()
		if err != nil {
			t.Fatalf("Failed to create processor: %v", err)
		}
		defer p.Close()

		htmlContent := `<html><body>` + input + `</body></html>`
		_, err = p.ExtractAllLinks([]byte(htmlContent))

		// Should handle any input without panic
		if err != nil {
			// Error is acceptable for malformed input
			t.Logf("Got error (acceptable): %v", err)
		}
	})
}
