package internal

import (
	"strings"
	"testing"
)

func TestSanitizeHTML_PreventsJavascriptProtocol(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		shouldFail bool
	}{
		{
			name:       "javascript in href",
			input:      `<a href="javascript:alert('xss')">click</a>`,
			shouldFail: false,
		},
		{
			name:       "javascript with mixed case in href",
			input:      `<a href="JavasCript:alert('xss')">click</a>`,
			shouldFail: false,
		},
		{
			name:       "javascript with spaces in href",
			input:      `<a href=" javascript:alert('xss')">click</a>`,
			shouldFail: false,
		},
		{
			name:       "javascript in src",
			input:      `<img src="javascript:alert('xss')">`,
			shouldFail: false,
		},
		{
			name:       "vbscript in href",
			input:      `<a href="vbscript:msgbox('xss')">click</a>`,
			shouldFail: false,
		},
		{
			name:       "file protocol in href",
			input:      `<a href="file:///etc/passwd">click</a>`,
			shouldFail: false,
		},
		{
			name:       "data URL with javascript",
			input:      `<a href="data:text/html,<script>alert('xss')</script>">click</a>`,
			shouldFail: false,
		},
		{
			name:       "valid http link",
			input:      `<a href="https://example.com">click</a>`,
			shouldFail: true,
		},
		{
			name:       "valid relative link",
			input:      `<a href="/path/to/page">click</a>`,
			shouldFail: true,
		},
		{
			name:       "onclick handler",
			input:      `<a href="https://example.com" onclick="alert('xss')">click</a>`,
			shouldFail: false,
		},
		{
			name:       "onerror handler",
			input:      `<img src="invalid.jpg" onerror="alert('xss')">`,
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := SanitizeHTML(tt.input)
			hasJavascript := strings.Contains(strings.ToLower(output), "javascript:")
			hasVbscript := strings.Contains(strings.ToLower(output), "vbscript:")
			hasFile := strings.Contains(strings.ToLower(output), "file:")
			hasOnclick := strings.Contains(strings.ToLower(output), "onclick")
			hasOnerror := strings.Contains(strings.ToLower(output), "onerror")

			hasAnyDangerous := hasJavascript || hasVbscript || hasFile || hasOnclick || hasOnerror

			if tt.shouldFail {
				if hasAnyDangerous {
					t.Errorf("expected safe output, but found dangerous content in: %s", output)
				}
			} else {
				if hasAnyDangerous {
					t.Errorf("dangerous content should be removed, but found in: %s", output)
				}
			}
		})
	}
}

func TestSanitizeHTML_ReturnsEmptyOnError(t *testing.T) {
	invalidInputs := []string{
		"<<<>>>",
		"<div>",
		"<a href='test'>link</a",
	}

	for _, input := range invalidInputs {
		t.Run(input, func(t *testing.T) {
			output := SanitizeHTML(input)
			if output == input {
				t.Errorf("should not return original content on parse error, got: %s", output)
			}
		})
	}
}

func TestSanitizeHTML_MalformedDataURLs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{
			name:  "valid data URL",
			input: `<img src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==">`,
			valid: true,
		},
		{
			name:  "data URL with control characters",
			input: "<img src=\"data:image/png;base,\x00\x01\x02\">",
			valid: false,
		},
		{
			name:  "data URL with null bytes",
			input: "<img src=\"data:image/png;base,abc\x00def\">",
			valid: false,
		},
		{
			name:  "data URL with invalid base64",
			input: `<img src="data:image/png;base64,!!!invalid!!!">`,
			valid: true,
		},
		{
			name:  "data URL with unsafe script content",
			input: `<img src="data:text/html,<script>alert(1)</script>">`,
			valid: false, // Now rejected due to whitelist
		},
		{
			name:  "data URL with unsafe text content",
			input: `<img src="data:text/plain,hello">`,
			valid: false, // Now rejected due to whitelist
		},
		{
			name:  "data URL with safe font content",
			input: `<img src="data:font/woff2;base64,ABC123">`,
			valid: true, // Font types are allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := SanitizeHTML(tt.input)
			if !tt.valid {
				if strings.Contains(output, "data:") {
					t.Errorf("malformed data URL should be removed, but found in: %s", output)
				}
			}
		})
	}
}

func TestSanitizeHTML_EventHandlers(t *testing.T) {
	eventHandlers := []string{
		"onclick", "onerror", "onload", "onmouseover", "onmouseout",
		"onfocus", "onblur", "onchange", "onsubmit", "onreset", "ondblclick",
	}

	for _, handler := range eventHandlers {
		t.Run(handler, func(t *testing.T) {
			input := `<a href="https://example.com" ` + handler + `="alert('xss')">click</a>`
			output := SanitizeHTML(input)
			if strings.Contains(strings.ToLower(output), strings.ToLower(handler)) {
				t.Errorf("%s handler should be removed, but found in: %s", handler, output)
			}
		})
	}
}

func TestSanitizeHTML_DangerousTags(t *testing.T) {
	dangerousTags := []string{
		"script", "style", "noscript", "iframe", "embed", "object", "form", "input", "button",
	}

	for _, tag := range dangerousTags {
		t.Run(tag, func(t *testing.T) {
			input := `<` + tag + `>content</` + tag + `>`
			output := SanitizeHTML(input)
			if strings.Contains(strings.ToLower(output), "<"+tag) {
				t.Errorf("%s tag should be removed, but found in: %s", tag, output)
			}
		})
	}
}

func TestSanitizeHTML_URIAttributes(t *testing.T) {
	uriAttributes := []string{
		"href", "src", "cite", "action", "data", "formaction", "poster", "background", "longdesc",
	}

	for _, attr := range uriAttributes {
		t.Run(attr, func(t *testing.T) {
			input := `<a ` + attr + `="javascript:alert('xss')">click</a>`
			output := SanitizeHTML(input)
			if strings.Contains(strings.ToLower(output), "javascript:") {
				t.Errorf("javascript: should be removed from %s attribute in: %s", attr, output)
			}
		})
	}
}

func TestIsSafeURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		safe bool
	}{
		{"empty", "", true},
		{"http", "https://example.com", true},
		{"relative", "/path/to/page", true},
		{"javascript", "javascript:alert(1)", false},
		{"javascript with spaces", " javascript:alert(1)", false},
		{"javascript uppercase", "JAVASCRIPT:alert(1)", false},
		{"vbscript", "vbscript:msgbox(1)", false},
		{"file", "file:///etc/passwd", false},
		{"data valid image", "data:image/png;base64,abc123", true},
		{"data valid font", "data:font/woff2;base64,abc", true},
		{"data unsafe image svg", "data:image/svg+xml;base64,abc", false},
		{"data unsafe script", "data:text/html,<script>alert(1)</script>", false},
		{"data unsafe invalid media", "data:invalid/media,abc", false},
		{"data unsafe text", "data:text/plain,abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSafeURI(tt.uri)
			if result != tt.safe {
				t.Errorf("isSafeURI(%q) = %v, want %v", tt.uri, result, tt.safe)
			}
		})
	}
}

func TestIsValidDataURL(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		valid bool
	}{
		{"not data URL", "https://example.com", false},
		{"no comma", "data:text/html", false},
		{"just data:", "data:,", false},
		{"valid image png", "data:image/png;base64,iVBORw0KG", true},
		{"valid image gif", "data:image/gif;base64,R0lGODlh", true},
		{"valid font", "data:font/woff2;base64,ABC123", true},
		{"control characters", "data:text/html,\x00\x01", false},
		{"del character", "data:text/html,\x7f", false},
		{"valid base64", "data:image/png;base64,ABC123", true},
		{"invalid base64 chars with control", "data:image/png;base64,\x01\x02", false},
		{"unsafe text html", "data:text/html,<script>", false},
		{"unsafe text plain", "data:text/plain,abc", false},
		{"too large", "data:image/png;base64," + string(make([]byte, 100001)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidDataURL(tt.url)
			if result != tt.valid {
				t.Errorf("isValidDataURL(%q) = %v, want %v", tt.url, result, tt.valid)
			}
		})
	}
}
