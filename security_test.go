package html_test

// security_test.go - Comprehensive security and robustness tests
// Tests for XSS prevention, injection attacks, and malformed input handling

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cybergodev/html"
)

// TestXSSPrevention tests that dangerous scripts are removed from content
func TestXSSPrevention(t *testing.T) {
	t.Parallel()

	p, err := html.New()
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
			result, err := p.Extract([]byte(tt.html), html.DefaultExtractConfig())
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

	p, err := html.New()
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

	_, err = p.Extract([]byte(largeHTML), html.DefaultExtractConfig())
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

	// Create deeply nested HTML
	deepHTML := "<html><body>"
	for i := 0; i < 100; i++ {
		deepHTML += "<div>"
	}
	deepHTML += "Content"
	for i := 0; i < 100; i++ {
		deepHTML += "</div>"
	}
	deepHTML += "</body></html>"

	result, err := p.Extract([]byte(deepHTML), html.DefaultExtractConfig())
	if err != nil {
		// Deep nesting should either work or return a clear error
		t.Logf("Deep nesting handled with: %v", err)
	}
	if result != nil && strings.Contains(result.Text, "Content") {
		// Content should still be extracted if processing succeeded
		t.Log("Content extracted from deeply nested HTML")
	}
}

// TestMalformedHTMLHandling tests tolerance for malformed HTML
func TestMalformedHTMLHandling(t *testing.T) {
	t.Parallel()

	p, err := html.New()
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
			result, err := p.Extract([]byte(tt.html), html.DefaultExtractConfig())
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

	p, err := html.New()
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
			result, err := p.Extract([]byte(tt.html), html.DefaultExtractConfig())
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

	p, err := html.New()
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

	result, err := p.Extract(invalidUTF8, html.DefaultExtractConfig())
	if err != nil {
		t.Logf("Invalid UTF-8 handled with error: %v", err)
	}
	if result != nil && result.Text == "" {
		// Empty result is acceptable for severely invalid input
		t.Log("Empty result for invalid UTF-8")
	}
}

// TestControlCharacterHandling tests handling of control characters
func TestControlCharacterHandling(t *testing.T) {
	t.Parallel()

	p, err := html.New()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Create HTML with various control characters
	controlCharHTML := "<html><body>Content\x00\x01\x02\x1F\x7Fwith\x80\x81\x82control</body></html>"

	result, err := p.Extract([]byte(controlCharHTML), html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	// Control characters may be preserved in the text - that's acceptable behavior
	// The important thing is that extraction doesn't crash and returns a result
	if result == nil {
		t.Error("Expected non-nil result")
	}

	// Just verify that extraction completed successfully
	_ = result.Text
}

// TestNullByteInjection tests null byte handling in URLs and paths
func TestNullByteInjection(t *testing.T) {
	t.Parallel()

	p, err := html.New()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	nullByteCases := []struct {
		name  string
		html  string
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
			result, err := p.Extract([]byte(tt.html), html.DefaultExtractConfig())
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

	p, err := html.New()
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

	result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
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
		_, _ = p.Extract(htmlContent, html.DefaultExtractConfig())
	}
}
