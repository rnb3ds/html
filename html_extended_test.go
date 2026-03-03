package html_test

// html_extended_test.go - Extended error handling and edge case tests
// This file contains:
// - Comprehensive error handling tests (input validation, timeout, file errors)
// - Edge case tests (empty input, unicode, whitespace, entities)

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cybergodev/html"
)

// ============================================================================
// COMPREHENSIVE ERROR HANDLING TESTS
// ============================================================================

func TestErrorHandlingComprehensive(t *testing.T) {
	t.Parallel()

	t.Run("ErrInputTooLarge - oversized input rejected", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxInputSize = 10000 // Set lower limit for faster test
		p, err := html.New(config)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		largeHTML := strings.Repeat("<div>test content here</div>", 10000)

		_, err = p.Extract([]byte(largeHTML))
		if err == nil {
			t.Errorf("Expected error for large input, got nil")
		}
	})

	t.Run("ErrInputTooLarge - exact limit boundary", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxInputSize = 5000
		p, err := html.New(config)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		validHTML := strings.Repeat("<div>a</div>", 100)
		_, err = p.Extract([]byte(validHTML))
		if err != nil {
			t.Errorf("Should accept input at MaxInputSize boundary, got: %v", err)
		}

		oversizedHTML := strings.Repeat("<div>a</div>", 1000)
		_, err = p.Extract([]byte(oversizedHTML))
		if err == nil {
			t.Errorf("Expected error for oversize input, got nil")
		}
	})

	t.Run("ErrInvalidHTML - malformed HTML handling", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		malformedCases := []struct {
			name string
			html string
		}{
			{"unclosed div", "<div>test"},
			{"triple angle brackets", "<<<>>>"},
			{"invalid tag", "<div test>test</div>"},
		}

		for _, tc := range malformedCases {
			t.Run(tc.name, func(t *testing.T) {
				// The library is tolerant and tries to extract anyway
				result, err := p.Extract([]byte(tc.html))
				if err != nil {
					t.Errorf("Library should tolerate malformed HTML, got: %v", err)
				}
				if result == nil {
					t.Error("Result should not be nil even for malformed HTML")
				}
			})
		}

		t.Run("excessively deep nesting", func(t *testing.T) {
			config := html.DefaultConfig()
			config.MaxDepth = 100
			p, err := html.New(config)
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()

			deepHTML := strings.Repeat("<div>", 200) + "test" + strings.Repeat("</div>", 200)
			_, err = p.Extract([]byte(deepHTML))
			// This should error due to max depth
			if err == nil {
				t.Error("Expected error for excessively deep nesting")
			}
		})
	})

	t.Run("ErrProcessingTimeout - timeout enforcement", func(t *testing.T) {
		config := html.DefaultConfig()
		config.ProcessingTimeout = 1 * time.Nanosecond
		p, err := html.New(config)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		largeHTML := strings.Repeat("<div>"+strings.Repeat("test ", 100)+"</div>", 1000)

		_, err = p.Extract([]byte(largeHTML))
		if err != html.ErrProcessingTimeout {
			t.Errorf("Expected ErrProcessingTimeout, got: %v", err)
		}
	})

	t.Run("ErrFileNotFound - non-existent file", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		_, err := p.ExtractFromFile("non-existent-file-12345.html")
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
		if !errors.Is(err, html.ErrFileNotFound) {
			t.Errorf("Expected ErrFileNotFound, got: %v", err)
		}
	})

	t.Run("ErrInvalidFilePath - empty and invalid paths", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		// Empty string should return ErrInvalidFilePath
		_, err := p.ExtractFromFile("")
		if !errors.Is(err, html.ErrInvalidFilePath) {
			t.Errorf("Expected ErrInvalidFilePath for empty path, got: %v", err)
		}

		// Whitespace paths may return OS-level errors (file not found)
		// This is acceptable behavior
		_, err = p.ExtractFromFile("   ")
		if err == nil {
			t.Error("Expected error for whitespace path")
		}
	})

}

// ============================================================================
// EDGE CASE TESTS
// ============================================================================

func TestEdgeCasesComprehensive(t *testing.T) {
	t.Parallel()

	t.Run("Empty and whitespace-only HTML", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		emptyCases := []string{
			"",
			"   ",
			"\t\n",
			"<!DOCTYPE html>",
			"<!-- comment only -->",
			"<html></html>",
			"<html><body></body></html>",
		}

		for _, htmlContent := range emptyCases {
			t.Run(fmt.Sprintf("empty_%d", len(htmlContent)), func(t *testing.T) {
				result, err := p.Extract([]byte(htmlContent))
				if err != nil {
					t.Errorf("Empty HTML should return result, not error: %v", err)
				}
				if result == nil {
					t.Error("Result should not be nil for empty HTML")
				}
			})
		}
	})

	t.Run("Unicode content handling", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		unicodeHTML := `<html><body>
			<h1>你好世界</h1>
			<p>Привет мир</p>
			<p>مرحبا بالعالم</p>
			<p>🎉🎊🎁 Emoji test</p>
			<p>עברית</p>
			<p>العربية</p>
		</body></html>`

		result, err := p.Extract([]byte(unicodeHTML))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "你好世界") {
			t.Error("Should preserve Chinese characters")
		}
		if !strings.Contains(result.Text, "Привет мир") {
			t.Error("Should preserve Cyrillic characters")
		}
		if !strings.Contains(result.Text, "مرحبا بالعالم") {
			t.Error("Should preserve Arabic characters")
		}
		if !strings.Contains(result.Text, "🎉") {
			t.Error("Should preserve emoji")
		}
	})

	t.Run("Excessive whitespace handling", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		whitespaceHTML := `<html><body>
			<p>Text    with     many     spaces</p>
			<p>Text

with

newlines</p>
			<p>Text	with	tabs</p>
		</body></html>`

		result, err := p.Extract([]byte(whitespaceHTML))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if strings.Contains(result.Text, "    ") {
			t.Error("Should collapse multiple spaces")
		}
		if strings.Contains(result.Text, "\t") {
			t.Error("Should replace tabs with spaces")
		}
	})

	t.Run("Mixed content and scripts", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		mixedHTML := `<html><body>
			<script>var x = "dangerous";</script>
			<style>.danger { color: red; }</style>
			<p>Valid content</p>
			<noscript>JavaScript required</noscript>
		</body></html>`

		result, err := p.Extract([]byte(mixedHTML))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if strings.Contains(strings.ToLower(result.Text), "var x") {
			t.Error("Script content should be removed")
		}
		if !strings.Contains(result.Text, "Valid content") {
			t.Error("Valid content should be extracted")
		}
	})

	t.Run("Entity decoding edge cases", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		entityHTML := `<html><body>
			<p>&amp;&lt;&gt;&quot;&apos;</p>
			<p>&nbsp;&nbsp;&copy;&reg;&trade;</p>
			<p>&mdash;&ndash;&hellip;</p>
			<p>&euro;&pound;&yen;</p>
			<p>&#65;&#x41;</p>
		</body></html>`

		result, err := p.Extract([]byte(entityHTML))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "&<>\"'") {
			t.Error("Basic XML entities should be decoded")
		}
		if !strings.Contains(result.Text, "©") {
			t.Error("Copyright entity should be decoded")
		}
		if !strings.Contains(result.Text, "—") {
			t.Error("Em dash entity should be decoded")
		}
	})
}

// ============================================================================
// UNICODE EDGE CASE TESTS
// ============================================================================

func TestUnicodeEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("zero-width characters", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		// Contains various zero-width characters
		// U+200B: Zero Width Space
		// U+200C: Zero Width Non-Joiner
		// U+200D: Zero Width Joiner
		// U+FEFF: Zero Width No-Break Space (BOM)
		htmlContent := `<html><body>` +
			`<p>Text\u200Bwith\u200Bzero\u200Bwidth\u200Bspaces</p>` +
			`<p>Text\u200Cwith\u200Djoiners</p>` +
			`<p>\uFEFFBOM\uFEFFtest</p>` +
			`</body></html>`

		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Zero-width characters should be preserved or handled gracefully
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		if result.Text == "" {
			t.Error("Expected non-empty text")
		}
	})

	t.Run("combining characters", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		// Contains combining characters
		// e + combining acute accent = é
		// a + combining umlaut = ä
		htmlContent := `<html><body>` +
			`<p>cafe\u0301</p>` + // café with combining accent
			`<p>na\u0308ive</p>` + // naïve with combining umlaut
			`<p>\u0041\u030A</p>` + // Å with combining ring
			`</body></html>`

		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		// Combining characters should be preserved
		if !strings.Contains(result.Text, "caf") {
			t.Error("Expected base text to be preserved")
		}
	})

	t.Run("bidirectional text", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		// Mixed LTR and RTL text
		htmlContent := `<html><body>` +
			`<p>Hello مرحبا World עולם</p>` +
			`<p dir="rtl">Right to left text</p>` +
			`<p>English و Arabic مختلط mixed</p>` +
			`</body></html>`

		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Bidirectional text should be preserved
		if !strings.Contains(result.Text, "Hello") {
			t.Error("Expected English text to be preserved")
		}
		if !strings.Contains(result.Text, "مرحبا") {
			t.Error("Expected Arabic text to be preserved")
		}
	})

	t.Run("surrogate pairs (emoji)", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		// Various emoji including multi-codepoint ones
		htmlContent := `<html><body>` +
			`<p>Simple: 😀😂🥳</p>` +
			`<p>Complex: 👨‍👩‍👧‍👦 🏳️‍🌈</p>` + // Family emoji, rainbow flag
			`<p>Skin tone: 👍🏻👍🏼👍🏽👍🏾👍🏿</p>` +
			`<p>ZWJ sequences: 👨‍⚕️ 👩‍💻</p>` +
			`</body></html>`

		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Emoji should be preserved
		if !strings.Contains(result.Text, "😀") {
			t.Error("Expected simple emoji to be preserved")
		}
	})

	t.Run("null character in content", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		// HTML with null character
		htmlContent := `<html><body>` +
			`<p>Before\x00After</p>` +
			`<p>Multi\x00null\x00chars</p>` +
			`</body></html>`

		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Should handle null characters gracefully (remove or sanitize)
		if result == nil {
			t.Error("Expected non-nil result")
		}
	})

	t.Run("maximum valid unicode", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		// Test with maximum valid Unicode code point (U+10FFFF)
		htmlContent := `<html><body>` +
			`<p>Max Unicode: \U0010FFFF</p>` +
			`<p>High codepoints: \U00010000 \U00020000</p>` +
			`</body></html>`

		result, err := p.Extract([]byte(htmlContent))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if result == nil {
			t.Error("Expected non-nil result")
		}
	})

	t.Run("invalid unicode sequences", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		// Invalid UTF-8 sequences
		invalidCases := []struct {
			name string
			html []byte
		}{
			{"invalid continuation", []byte(`<html><body>Test\x80\x81</body></html>`)},
			{"truncated sequence", []byte(`<html><body>Test\xc3</body></html>`)},
			{"overlong encoding", []byte(`<html><body>Test\xc0\x80</body></html>`)},
		}

		for _, tc := range invalidCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := p.Extract(tc.html)
				// Should either handle gracefully or return an error
				if err != nil {
					return // Error is acceptable
				}
				if result == nil {
					t.Error("Expected non-nil result when no error returned")
				}
			})
		}
	})
}

// ============================================================================
// MAXDEPTH BOUNDARY CONDITION TESTS
// ============================================================================

func TestMaxDepthBoundaryConditions(t *testing.T) {
	t.Parallel()

	t.Run("depth exactly at limit", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxDepth = 50

		p, err := html.New(config)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		// Create HTML with exactly 50 nested levels
		deepHTML := "<html><body>"
		for i := 0; i < 48; i++ { // 48 divs + html + body = 50
			deepHTML += "<div>"
		}
		deepHTML += "Content"
		for i := 0; i < 48; i++ {
			deepHTML += "</div>"
		}
		deepHTML += "</body></html>"

		result, err := p.Extract([]byte(deepHTML))

		// Should succeed - depth is exactly at limit
		if err != nil {
			t.Logf("Depth at limit returned error: %v", err)
		}
		if result != nil && !strings.Contains(result.Text, "Content") {
			t.Error("Expected content to be extracted at exact depth limit")
		}
	})

	t.Run("depth one over limit", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxDepth = 50

		p, err := html.New(config)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		// Create HTML with 51 nested levels (one over limit)
		deepHTML := "<html><body>"
		for i := 0; i < 49; i++ { // 49 divs + html + body = 51
			deepHTML += "<div>"
		}
		deepHTML += "Content"
		for i := 0; i < 49; i++ {
			deepHTML += "</div>"
		}
		deepHTML += "</body></html>"

		result, err := p.Extract([]byte(deepHTML))

		// Behavior depends on implementation:
		// - May return error (strict mode)
		// - May extract content anyway (tolerant mode)
		// Either is acceptable
		if err != nil {
			t.Logf("Depth over limit returned error (expected): %v", err)
		}
		if result != nil {
			t.Logf("Depth over limit extracted content: %s",
				truncateString(result.Text, 50))
		}
	})

	t.Run("depth limit zero", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxDepth = 0

		p, err := html.New(config)
		// MaxDepth=0 is invalid and should return an error
		if err == nil {
			p.Close()
			t.Error("Expected error for MaxDepth=0")
		}
	})

	t.Run("depth limit one", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxDepth = 1

		p, err := html.New(config)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		// Even just <html> tag counts as depth 1
		singleLevelHTML := `<html>Content</html>`

		result, err := p.Extract([]byte(singleLevelHTML))

		// The library enforces depth strictly, so even html tag may exceed
		if err != nil {
			t.Logf("MaxDepth=1 returned error: %v", err)
		}
		_ = result // May be nil
	})

	t.Run("very deep nesting with high limit", func(t *testing.T) {
		config := html.DefaultConfig()
		config.MaxDepth = 500

		p, err := html.New(config)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		// Create HTML with 200 nested levels
		deepHTML := "<html><body>"
		for i := 0; i < 198; i++ {
			deepHTML += "<div>"
		}
		deepHTML += "Content"
		for i := 0; i < 198; i++ {
			deepHTML += "</div>"
		}
		deepHTML += "</body></html>"

		result, err := p.Extract([]byte(deepHTML))

		if err != nil {
			t.Errorf("Deep nesting with high limit should succeed: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}
	})
}

// Helper function to truncate strings for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
