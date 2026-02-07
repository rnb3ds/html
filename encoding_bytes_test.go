package html_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/cybergodev/html"
)

// TestExtractWithEncodingDetection tests automatic encoding detection in Extract
func TestExtractWithEncodingDetection(t *testing.T) {
	t.Run("Windows-1252 encoded HTML", func(t *testing.T) {
		// Read actual Windows-1252 file
		testFile := "dev_test/Source Files/a2025q310-qexx311.html"
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Skip("Test file not found")
		}

		// Read file bytes
		htmlBytes, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read test file: %v", err)
		}

		// Extract using bytes method (with encoding detection)
		result, err := html.Extract(htmlBytes)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}

		// Verify UTF-8 output
		if !strings.Contains(result.Text, "registrant's") {
			t.Errorf("Expected UTF-8 apostrophes, got: %s", result.Text)
		}

		// Verify no garbled characters
		if strings.Contains(result.Text, "\ufffd") {
			t.Error("Found replacement character, encoding may not be detected correctly")
		}
	})

	t.Run("UTF-8 encoded HTML", func(t *testing.T) {
		htmlContent := `<html><head><meta charset="utf-8"></head><body><p>Hello 世界</p></body></html>`
		htmlBytes := []byte(htmlContent)

		result, err := html.Extract(htmlBytes)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}

		if !strings.Contains(result.Text, "Hello 世界") {
			t.Errorf("Expected 'Hello 世界', got: %s", result.Text)
		}
	})

	t.Run("ASCII HTML", func(t *testing.T) {
		htmlContent := `<html><body><p>Hello World</p></body></html>`
		htmlBytes := []byte(htmlContent)

		result, err := html.Extract(htmlBytes)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}

		expected := "Hello World"
		if !strings.Contains(result.Text, expected) {
			t.Errorf("Expected %q, got: %s", expected, result.Text)
		}
	})
}

// TestExtractToMarkdown tests automatic encoding detection in ExtractToMarkdown
func TestExtractToMarkdown(t *testing.T) {
	t.Run("Windows-1252 to Markdown", func(t *testing.T) {
		testFile := "dev_test/Source Files/a2025q310-qexx311.html"
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Skip("Test file not found")
		}

		htmlBytes, _ := os.ReadFile(testFile)

		markdown, err := html.ExtractToMarkdown(htmlBytes)
		if err != nil {
			t.Fatalf("ExtractToMarkdown failed: %v", err)
		}

		// Verify UTF-8 conversion
		if !strings.Contains(markdown, "registrant's") {
			t.Errorf("Expected UTF-8 apostrophes in markdown")
		}

		// Verify no garbled text
		if strings.Contains(markdown, "\ufffd") {
			t.Error("Found replacement character in markdown")
		}
	})

	t.Run("UTF-8 to Markdown", func(t *testing.T) {
		htmlContent := `<html><body><p>Hello 世界 © 2025</p></body></html>`
		htmlBytes := []byte(htmlContent)

		markdown, err := html.ExtractToMarkdown(htmlBytes)
		if err != nil {
			t.Fatalf("ExtractToMarkdown failed: %v", err)
		}

		if !strings.Contains(markdown, "Hello 世界") {
			t.Errorf("Expected 'Hello 世界', got: %s", markdown)
		}

		if !strings.Contains(markdown, "©") {
			t.Errorf("Expected copyright symbol")
		}
	})
}

// TestExtractText tests automatic encoding detection in ExtractText
func TestExtractText(t *testing.T) {
	t.Run("Windows-1252 text extraction", func(t *testing.T) {
		testFile := "dev_test/Source Files/a2025q310-qexx311.html"
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Skip("Test file not found")
		}

		htmlBytes, _ := os.ReadFile(testFile)

		text, err := html.ExtractText(htmlBytes)
		if err != nil {
			t.Fatalf("ExtractText failed: %v", err)
		}

		// Verify UTF-8
		if !strings.Contains(text, "registrant's") {
			t.Errorf("Expected UTF-8 apostrophes")
		}
	})

	t.Run("UTF-8 text extraction", func(t *testing.T) {
		htmlContent := `<html><body><p>Hello 世界 — Test</p></body></html>`
		htmlBytes := []byte(htmlContent)

		text, err := html.ExtractText(htmlBytes)
		if err != nil {
			t.Fatalf("ExtractText failed: %v", err)
		}

		if !strings.Contains(text, "Hello 世界") {
			t.Errorf("Expected 'Hello 世界', got: %s", text)
		}
	})
}

// TestExtractAllLinksWithEncoding tests automatic encoding detection in ExtractAllLinks
func TestExtractAllLinksWithEncoding(t *testing.T) {
	t.Run("UTF-8 HTML with links", func(t *testing.T) {
		htmlContent := `<html><body>
			<a href="https://example.com">Example</a>
			<a href="/relative">Relative Link</a>
		</body></html>`
		htmlBytes := []byte(htmlContent)

		links, err := html.ExtractAllLinks(htmlBytes)
		if err != nil {
			t.Fatalf("ExtractAllLinks failed: %v", err)
		}

		if len(links) != 2 {
			t.Errorf("Expected 2 links, got %d", len(links))
		}
	})

	t.Run("Windows-1252 HTML with links", func(t *testing.T) {
		// Create Windows-1252 HTML with smart quotes in link text
		htmlBytes := []byte(`<html><head><meta charset="windows-1252"></head>
		<body><a href="/test">Company's Website</a></body></html>`)

		links, err := html.ExtractAllLinks(htmlBytes)
		if err != nil {
			t.Fatalf("ExtractAllLinks failed: %v", err)
		}

		if len(links) != 1 {
			t.Fatalf("Expected 1 link, got %d", len(links))
		}

		// Verify the apostrophe is correctly decoded
		if !strings.Contains(links[0].Title, "Company's") {
			t.Errorf("Expected 'Company's', got: %s", links[0].Title)
		}
	})
}

// TestExtractWithEncoding verifies that Extract handles non-UTF8 encodings correctly
func TestExtractWithEncoding(t *testing.T) {
	testFile := "dev_test/Source Files/a2025q310-qexx311.html"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Test file not found")
	}

	htmlBytes, _ := os.ReadFile(testFile)

	// Extract with automatic encoding detection
	result, err := html.Extract(htmlBytes)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify correct encoding handling
	hasCorrectApostrophes := strings.Contains(result.Text, "registrant's")
	t.Logf("Extract: has correct apostrophes = %v", hasCorrectApostrophes)

	// Extract should handle Windows-1252 encoding correctly
	if !hasCorrectApostrophes {
		t.Error("Extract should correctly handle Windows-1252 encoding")
	}

	// Verify no garbled characters
	if strings.Contains(result.Text, "\ufffd") {
		t.Error("Found replacement character, encoding may not be detected correctly")
	}
}

// TestHTTPScenario simulates real-world HTTP response processing
func TestHTTPScenario(t *testing.T) {
	// Create a test server that serves Windows-1252 HTML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read Windows-1252 test file
		testFile := "dev_test/Source Files/a2025q310-qexx311.html"
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			http.Error(w, "Test file not found", http.StatusNotFound)
			return
		}

		htmlBytes, _ := os.ReadFile(testFile)
		w.Header().Set("Content-Type", "text/html; charset=windows-1252")
		w.Write(htmlBytes)
	}))
	defer server.Close()

	// Simulate HTTP client
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("HTTP GET failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response bytes
	htmlBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// WRONG WAY: Direct string conversion
	_ = string(htmlBytes) // This would cause garbled text

	// CORRECT WAY: Use Extract
	result, err := html.Extract(htmlBytes)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify correct encoding
	if !strings.Contains(result.Text, "registrant's") {
		t.Errorf("Expected correctly decoded UTF-8 text")
	}

	t.Logf("Successfully extracted and decoded Windows-1252 HTML from HTTP response")
}

// TestExtractWithForcedEncoding tests forced encoding override
func TestExtractWithForcedEncoding(t *testing.T) {
	htmlContent := "<html><body><p>Test</p></body></html>"
	htmlBytes := []byte(htmlContent)

	// Test with forced encoding
	config := html.DefaultExtractConfig()
	config.Encoding = "utf-8"

	result, err := html.Extract(htmlBytes, config)
	if err != nil {
		t.Fatalf("Extract with config failed: %v", err)
	}

	if !strings.Contains(result.Text, "Test") {
		t.Errorf("Expected 'Test', got: %s", result.Text)
	}
}
