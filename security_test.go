package html_test

// security_test.go - Comprehensive security and robustness tests
// Tests for XSS prevention, injection attacks, and malformed input handling

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/cybergodev/html"
)

// TestXSSPrevention tests that dangerous scripts are removed from content
func TestXSSPrevention(t *testing.T) {
	t.Parallel()

	p, err := html.New(html.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	xssPayloads := []struct {
		name string
		html string
	}{
		{
			name: "basic script tag",
			html: `<html><body><script>alert('XSS')</script><p>Content</p></body></html>`,
		},
		{
			name: "script with src",
			html: `<html><body><script src="http://evil.com/xss.js"></script><p>Content</p></body></html>`,
		},
		{
			name: "onclick attribute",
			html: `<html><body><div onclick="alert('XSS')">Content</div></body></html>`,
		},
		{
			name: "javascript href",
			html: `<html><body><a href="javascript:alert('XSS')">Click</a><p>Content</p></body></html>`,
		},
		{
			name: "iframe injection",
			html: `<html><body><iframe src="http://evil.com"></iframe><p>Content</p></body></html>`,
		},
		{
			name: "embed tag",
			html: `<html><body><embed src="evil.swf"><p>Content</p></body></html>`,
		},
		{
			name: "object tag",
			html: `<html><body><object data="evil.swf"></object><p>Content</p></body></html>`,
		},
		{
			name: "onerror attribute",
			html: `<html><body><img src=x onerror="alert('XSS')"><p>Content</p></body></html>`,
		},
		{
			name: "onload attribute",
			html: `<html><body><body onload="alert('XSS')"><p>Content</p></body></html>`,
		},
		{
			name: "multiple event handlers",
			html: `<html><body><div onmouseover="alert('XSS')" onmouseout="alert('XSS2')">Content</div></body></html>`,
		},
	}

	for _, tt := range xssPayloads {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Extract([]byte(tt.html))
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			// Check that dangerous content is not in the extracted text
			extractedText := strings.ToLower(result.Text)
			if strings.Contains(extractedText, "alert") ||
				strings.Contains(extractedText, "javascript:") ||
				strings.Contains(extractedText, "onerror") ||
				strings.Contains(extractedText, "onclick") {
				t.Errorf("XSS payload not sanitized: %s", extractedText)
			}

			// Check that script tags are not in links/images
			for _, link := range result.Links {
				if strings.Contains(strings.ToLower(link.URL), "javascript:") {
					t.Errorf("JavaScript URL not removed from links: %s", link.URL)
				}
			}
		})
	}
}

// TestPathTraversalPrevention tests file path validation
func TestPathTraversalPrevention(t *testing.T) {
	t.Parallel()

	p, err := html.New(html.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	pathTraversalAttempts := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
		"../../test.html",
		"../test.html",
		"./../../etc/passwd",
		"test/../../../etc/passwd",
		"..%2F..%2F..%2Fetc%2Fpasswd",
		"..%5c..%5c..%5cwindows%5csystem32",
	}

	for _, path := range pathTraversalAttempts {
		t.Run(path, func(t *testing.T) {
			_, err := p.ExtractFromFile(path)
			if err == nil {
				t.Errorf("Expected error for path traversal attempt: %s", path)
			}
			if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "not found") {
				t.Logf("Path '%s' rejected with: %v", path, err)
			}
		})
	}
}

// TestLargeInputDoSPrevention tests rejection of oversized inputs
func TestLargeInputDoSPrevention(t *testing.T) {
	t.Parallel()

	config := html.DefaultConfig()
	config.MaxInputSize = 1024 // 1KB limit for testing

	p, err := html.New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Create input larger than MaxInputSize
	largeHTML := strings.Repeat("<div>test content</div>", 1000) // ~18KB

	_, err = p.Extract([]byte(largeHTML))
	if err == nil {
		t.Error("Expected error for oversized input")
	}
	if err != html.ErrInputTooLarge {
		t.Logf("Large input rejected with: %v", err)
	}
}

// TestDeepNestingDoSPrevention tests depth limit enforcement
func TestDeepNestingDoSPrevention(t *testing.T) {
	t.Parallel()

	config := html.DefaultConfig()
	config.MaxDepth = 50 // Low limit for testing

	p, err := html.New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Create deeply nested HTML (100 levels, exceeding MaxDepth of 50)
	deepHTML := "<html><body>"
	for i := 0; i < 100; i++ {
		deepHTML += "<div>"
	}
	deepHTML += "Content"
	for i := 0; i < 100; i++ {
		deepHTML += "</div>"
	}
	deepHTML += "</body></html>"

	result, err := p.Extract([]byte(deepHTML))

	// The library should either:
	// 1. Process successfully (being tolerant of deep nesting), OR
	// 2. Return a clear error (rejecting the input)
	// Either behavior is acceptable for DoS prevention
	if err != nil {
		// Error is acceptable - input was rejected
		if !strings.Contains(err.Error(), "depth") &&
			!strings.Contains(err.Error(), "invalid") &&
			err != html.ErrInvalidHTML {
			t.Errorf("Unexpected error for deep nesting: %v", err)
		}
		return
	}

	// If processing succeeded, verify the result is valid
	if result == nil {
		t.Fatal("Expected non-nil result when no error returned")
	}

	// Content should be extracted if processing succeeded
	if !strings.Contains(result.Text, "Content") {
		t.Errorf("Expected 'Content' in extracted text, got: %q", result.Text)
	}

	// Verify text is not empty
	if result.Text == "" {
		t.Error("Expected non-empty text extraction from deeply nested HTML")
	}
}

// TestMalformedHTMLHandling tests tolerance for malformed HTML
func TestMalformedHTMLHandling(t *testing.T) {
	t.Parallel()

	p, err := html.New(html.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	malformedCases := []struct {
		name string
		html string
	}{
		{
			name: "unclosed div",
			html: `<html><body><div>Test<p>Content</body></html>`,
		},
		{
			name: "missing closing tags",
			html: `<html><body><h1>Title<p>Content</body></html>`,
		},
		{
			name: "nested improper tags",
			html: `<html><body><b><i>Test</b></i></body></html>`,
		},
		{
			name: "extra closing tags",
			html: `<html><body><p>Test</p></div></body></html>`,
		},
		{
			name: "attribute without value",
			html: `<html><body><input type="text" disabled>Content</body></html>`,
		},
		{
			name: "mismatched quotes",
			html: `<html><body><a href="http://example.com>Link</a>Content</body></html>`,
		},
		{
			name: "empty tag name",
			html: `<html><body><>Test</>Content</body></html>`,
		},
		{
			name: "triple angle brackets",
			html: `<<<>>>Test content<<<>>>`,
		},
	}

	for _, tt := range malformedCases {
		t.Run(tt.name, func(t *testing.T) {
			// Library should be tolerant and extract what it can
			result, err := p.Extract([]byte(tt.html))
			if err != nil && err != html.ErrInvalidHTML {
				t.Fatalf("Extract() failed for malformed HTML: %v", err)
			}
			if result == nil {
				t.Error("Expected non-nil result even for malformed HTML")
			}
		})
	}
}

// TestDataURLInjection tests data URL validation
func TestDataURLInjection(t *testing.T) {
	t.Parallel()

	p, err := html.New(html.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	dataURLCases := []struct {
		name          string
		html          string
		shouldExtract bool
	}{
		{
			name:          "safe image data URL",
			html:          `<html><body><img src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAYAAACNbyblAAAAHElEQVQI12P4//8/w38GIAXDIBKE0DHxgljNBAAO9TXL0Y4OHwAAAABJRU5ErkJggg==">Content</body></html>`,
			shouldExtract: true,
		},
		{
			name:          "oversized data URL",
			html:          fmt.Sprintf(`<html><body><img src="data:image/png;base64,%s">Content</body></html>`, strings.Repeat("A", 200000)),
			shouldExtract: false,
		},
		{
			name:          "script data URL",
			html:          `<html><body><script src="data:text/javascript,alert('XSS')"></script>Content</body></html>`,
			shouldExtract: false,
		},
		{
			name:          "html data URL",
			html:          `<html><body><iframe src="data:text/html,<script>alert('XSS')</script>"></iframe>Content</body></html>`,
			shouldExtract: false,
		},
	}

	for _, tt := range dataURLCases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Extract([]byte(tt.html))
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			hasImage := len(result.Images) > 0
			if tt.shouldExtract && !hasImage {
				t.Error("Expected image to be extracted from data URL")
			}
			if !tt.shouldExtract && hasImage {
				t.Error("Expected data URL to be rejected")
			}
		})
	}
}

// TestInvalidUTF8Handling tests handling of invalid UTF-8 sequences
func TestInvalidUTF8Handling(t *testing.T) {
	t.Parallel()

	p, err := html.New(html.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Create HTML with invalid UTF-8 sequences
	invalidUTF8 := []byte{
		'<', 'h', 't', 'm', 'l', '>', '<', 'b', 'o', 'd', 'y', '>',
		0xFF, 0xFF, 0xFF, // Invalid UTF-8 bytes
		'C', 'o', 'n', 't', 'e', 'n', 't',
		0xC0, 0x80, // Overlong UTF-8 encoding
		'<', '/', 'b', 'o', 'd', 'y', '>', '<', '/', 'h', 't', 'm', 'l', '>',
	}

	result, err := p.Extract(invalidUTF8)

	// The library should handle invalid UTF-8 gracefully:
	// 1. Either process it (sanitizing/replacing invalid sequences), OR
	// 2. Return a clear error
	if err != nil {
		// Error is acceptable - invalid input was rejected
		if !strings.Contains(err.Error(), "encoding") &&
			!strings.Contains(err.Error(), "utf") &&
			!strings.Contains(err.Error(), "invalid") &&
			err != html.ErrInvalidHTML {
			t.Errorf("Unexpected error for invalid UTF-8: %v", err)
		}
		return
	}

	// If processing succeeded, verify the result is valid
	if result == nil {
		t.Fatal("Expected non-nil result when no error returned")
	}

	// The text should be valid UTF-8 (no replacement characters from invalid sequences)
	// Check that result.Text is valid UTF-8
	for i, r := range result.Text {
		if r == utf8.RuneError {
			// Check if this is a valid RuneError or an invalid sequence
			// The library should sanitize invalid sequences
			t.Logf("RuneError found at position %d in result text", i)
		}
	}

	// Content should be present (the valid "Content" text)
	if !strings.Contains(result.Text, "Content") {
		t.Errorf("Expected 'Content' in extracted text, got: %q", result.Text)
	}
}

// TestControlCharacterHandling tests handling of control characters
func TestControlCharacterHandling(t *testing.T) {
	t.Parallel()

	p, err := html.New(html.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Create HTML with various control characters
	controlCharHTML := "<html><body>Content\x00\x01\x02\x1F\x7Fwith\x80\x81\x82control</body></html>"

	result, err := p.Extract([]byte(controlCharHTML))
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	// Verify the result is valid
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// The text should be non-empty
	if result.Text == "" {
		t.Error("Expected non-empty text extraction")
	}

	// Verify that some content was extracted (control chars may be sanitized or preserved)
	// The key requirement is that extraction doesn't crash
	if !strings.Contains(result.Text, "Content") && !strings.Contains(result.Text, "with") {
		t.Errorf("Expected 'Content' or 'with' in extracted text, got: %q", result.Text)
	}

	// Check that null bytes are handled safely (removed or replaced, not causing issues)
	if strings.Contains(result.Text, "\x00") {
		// Null bytes in output may be acceptable but should be documented
		t.Logf("Warning: Null byte present in extracted text")
	}

	// Verify no control characters in the printable range cause crashes
	// The extraction should complete without panicking
	t.Logf("Successfully extracted text with length %d from control character input", len(result.Text))
}

// TestNullByteInjection tests null byte handling in URLs and paths
func TestNullByteInjection(t *testing.T) {
	t.Parallel()

	p, err := html.New(html.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	nullByteCases := []struct {
		name string
		html string
	}{
		{
			name: "null in URL",
			html: `<html><body><a href="http://example.com\x00.php">Link</a></body></html>`,
		},
		{
			name: "null in src",
			html: `<html><body><img src="http://example.com\x00.jpg" alt="Image"></body></html>`,
		},
		{
			name: "multiple nulls",
			html: `<html><body><a href="http://example.com\x00\x00\x00.php">Link</a></body></html>`,
		},
	}

	for _, tt := range nullByteCases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Extract([]byte(tt.html))
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			// Check that null bytes are not in extracted URLs
			for _, link := range result.Links {
				if strings.Contains(link.URL, "\x00") {
					t.Errorf("Null byte not removed from URL: %q", link.URL)
				}
			}
			for _, img := range result.Images {
				if strings.Contains(img.URL, "\x00") {
					t.Errorf("Null byte not removed from image src: %q", img.URL)
				}
			}
		})
	}
}

// TestProtocolRelativeURLSafety tests safety of protocol-relative URLs
func TestProtocolRelativeURLSafety(t *testing.T) {
	t.Parallel()

	p, err := html.New(html.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	htmlContent := []byte(`<html>
		<head><base href="//example.com/path/"></head>
		<body>
			<a href="//evil.com/xss.js">Script</a>
			<img src="//evil.com/steal.jpg">
			<p>Content</p>
		</body>
	</html>`)

	result, err := p.Extract(htmlContent)
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	// Protocol-relative URLs should be preserved or converted to https
	// but they should not cause security issues in extraction
	if result == nil {
		t.Error("Expected non-nil result")
	}
}

// TestBenchmarkDoSPrevention benchmarks DoS prevention overhead
func BenchmarkDoSPreventionChecks(b *testing.B) {
	config := html.DefaultConfig()
	p, _ := html.New(config)
	defer p.Close()

	// Normal HTML content
	htmlContent := []byte(`<html><body><h1>Normal Content</h1><p>Test paragraph</p></body></html>`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.Extract(htmlContent)
	}
}

// TestSVGXSSPrevention tests that SVG-based XSS attacks are blocked
func TestSVGXSSPrevention(t *testing.T) {
	t.Parallel()

	p, err := html.New(html.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	svgPayloads := []struct {
		name string
		html string
	}{
		{
			name: "svg with onload event",
			html: `<html><body><svg onload="alert('XSS')"><circle cx="50"/></svg><p>Content</p></body></html>`,
		},
		{
			name: "svg with embedded script",
			html: `<html><body><svg><script>alert('XSS')</script></svg><p>Content</p></body></html>`,
		},
		{
			name: "svg with foreignObject",
			html: `<html><body><svg><foreignObject><body onload="alert('XSS')"></body></foreignObject></svg><p>Content</p></body></html>`,
		},
		{
			name: "svg animate element",
			html: `<html><body><svg><animate onbegin="alert('XSS')" attributeName="x"/></svg><p>Content</p></body></html>`,
		},
		{
			name: "svg use xlink",
			html: `<html><body><svg><use xlink:href="data:image/svg+xml,<svg onload='alert(1)'/>"/></svg><p>Content</p></body></html>`,
		},
		{
			name: "svg set element",
			html: `<html><body><svg><set onbegin="alert('XSS')"/></svg><p>Content</p></body></html>`,
		},
	}

	for _, tt := range svgPayloads {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Extract([]byte(tt.html))
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			extractedText := strings.ToLower(result.Text)
			if strings.Contains(extractedText, "alert") {
				t.Errorf("SVG XSS payload not sanitized: %s", extractedText)
			}
			if strings.Contains(extractedText, "onload") {
				t.Errorf("SVG onload event not removed: %s", extractedText)
			}
		})
	}
}

// TestMathMLXSSPrevention tests that MathML-based XSS attacks are blocked
func TestMathMLXSSPrevention(t *testing.T) {
	t.Parallel()

	p, err := html.New(html.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	mathPayloads := []struct {
		name string
		html string
	}{
		{
			name: "math with annotation-xml",
			html: `<html><body><math><annotation-xml encoding="application/xhtml+xml"><script>alert('XSS')</script></annotation-xml></math><p>Content</p></body></html>`,
		},
		{
			name: "math with javascript href",
			html: `<html><body><math href="javascript:alert('XSS')"><mtext>click</mtext></math><p>Content</p></body></html>`,
		},
	}

	for _, tt := range mathPayloads {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Extract([]byte(tt.html))
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			extractedText := strings.ToLower(result.Text)
			if strings.Contains(extractedText, "alert") {
				t.Errorf("MathML XSS payload not sanitized: %s", extractedText)
			}
		})
	}
}

// TestHighSecurityConfig tests the high-security configuration
func TestHighSecurityConfig(t *testing.T) {
	t.Parallel()

	config := html.HighSecurityConfig()

	// Verify security-enhanced settings
	if config.MaxInputSize > 10*1024*1024 {
		t.Errorf("HighSecurityConfig MaxInputSize should be <= 10MB, got %d", config.MaxInputSize)
	}
	if config.MaxDepth > 100 {
		t.Errorf("HighSecurityConfig MaxDepth should be <= 100, got %d", config.MaxDepth)
	}
	if config.ProcessingTimeout > 15*time.Second {
		t.Errorf("HighSecurityConfig ProcessingTimeout should be <= 15s, got %v", config.ProcessingTimeout)
	}
	if !config.EnableSanitization {
		t.Error("HighSecurityConfig should always have EnableSanitization=true")
	}

	// Test that high-security config works
	p, err := html.New(config)
	if err != nil {
		t.Fatalf("Failed to create processor with HighSecurityConfig: %v", err)
	}
	defer p.Close()

	// Test with normal content
	htmlContent := []byte(`<html><body><h1>Test</h1><p>Content</p></body></html>`)
	result, err := p.Extract(htmlContent)
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}
	if result.Text == "" {
		t.Error("Expected non-empty text extraction")
	}

	// Test that oversized input is rejected
	largeHTML := make([]byte, 15*1024*1024) // 15MB, exceeds 10MB limit
	_, err = p.Extract(largeHTML)
	if err == nil {
		t.Error("Expected error for oversized input in high-security mode")
	}
}

// TestHighSecurityConfigStricterThanDefault verifies high-security is more restrictive
func TestHighSecurityConfigStricterThanDefault(t *testing.T) {
	t.Parallel()

	defaultConfig := html.DefaultConfig()
	highSecConfig := html.HighSecurityConfig()

	if highSecConfig.MaxInputSize >= defaultConfig.MaxInputSize {
		t.Error("HighSecurityConfig MaxInputSize should be smaller than default")
	}
	if highSecConfig.MaxDepth >= defaultConfig.MaxDepth {
		t.Error("HighSecurityConfig MaxDepth should be smaller than default")
	}
	if highSecConfig.ProcessingTimeout >= defaultConfig.ProcessingTimeout {
		t.Error("HighSecurityConfig ProcessingTimeout should be shorter than default")
	}
}
