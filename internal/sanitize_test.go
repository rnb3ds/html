package internal

import (
	"strings"
	"testing"
)

func TestSanitizeHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []string
		avoid []string
	}{
		{
			name:  "remove script tags",
			input: `<div>Keep<script>alert('remove')</script>Keep</div>`,
			want:  []string{"Keep", "<div>"},
			avoid: []string{"script", "alert"},
		},
		{
			name:  "remove style tags",
			input: `<div>Keep<style>body{color:red}</style>Keep</div>`,
			want:  []string{"Keep", "<div>"},
			avoid: []string{"style", "color:red"},
		},
		{
			name:  "remove noscript tags",
			input: `<div>Keep<noscript>No JS</noscript>Keep</div>`,
			want:  []string{"Keep", "<div>"},
			avoid: []string{"noscript", "No JS"},
		},
		{
			name:  "remove all three",
			input: `<div>Keep<script>js</script><style>css</style><noscript>nojs</noscript>Keep</div>`,
			want:  []string{"Keep", "<div>"},
			avoid: []string{"script", "style", "noscript", "js", "css", "nojs"},
		},
		{
			name:  "empty string",
			input: "",
			want:  []string{""},
		},
		{
			name:  "no tags to remove",
			input: `<div><p>Normal content</p></div>`,
			want:  []string{"<div>", "<p>", "Normal content"},
		},
		{
			name:  "multiple script tags",
			input: `<div><script>1</script>Keep<script>2</script></div>`,
			want:  []string{"Keep", "<div>"},
			avoid: []string{"script", "1", "2"},
		},
		{
			name:  "nested tags",
			input: `<div><script><span>nested</span></script>Keep</div>`,
			want:  []string{"Keep", "<div>"},
			avoid: []string{"script", "nested"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeHTML(tt.input)

			for _, want := range tt.want {
				if !strings.Contains(result, want) {
					t.Errorf("SanitizeHTML() result should contain %q, got %q", want, result)
				}
			}

			for _, avoid := range tt.avoid {
				if strings.Contains(result, avoid) {
					t.Errorf("SanitizeHTML() result should not contain %q, got %q", avoid, result)
				}
			}
		})
	}
}

func TestRemoveTagContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		tag     string
		want    []string
		avoid   []string
	}{
		{
			name:    "remove script",
			content: `<div>Before<script>remove</script>After</div>`,
			tag:     "script",
			want:    []string{"Before", "After", "<div>"},
			avoid:   []string{"<script>", "remove", "</script>"},
		},
		{
			name:    "remove style",
			content: `<p>Text<style>css</style>More</p>`,
			tag:     "style",
			want:    []string{"Text", "More", "<p>"},
			avoid:   []string{"<style>", "css", "</style>"},
		},
		{
			name:    "tag not present",
			content: `<div>Content</div>`,
			tag:     "script",
			want:    []string{"<div>", "Content", "</div>"},
		},
		{
			name:    "empty content",
			content: "",
			tag:     "script",
			want:    []string{""},
		},
		{
			name:    "empty tag",
			content: `<div>Content</div>`,
			tag:     "",
			want:    []string{"<div>", "Content", "</div>"},
		},
		{
			name:    "multiple tags",
			content: `<div><script>1</script>Middle<script>2</script>End</div>`,
			tag:     "script",
			want:    []string{"Middle", "End", "<div>"},
			avoid:   []string{"<script>", "1", "2", "</script>"},
		},
		{
			name:    "tag with attributes",
			content: `<div><script type="text/javascript">code</script>Text</div>`,
			tag:     "script",
			want:    []string{"Text", "<div>"},
			avoid:   []string{"<script", "code", "</script>"},
		},
		{
			name:    "unclosed tag",
			content: `<div><script>unclosed</div>`,
			tag:     "script",
			want:    []string{"<div>", "unclosed", "</div>"},
		},
		{
			name:    "tag without closing bracket",
			content: `<div><script`,
			tag:     "script",
			want:    []string{"<div>", "<script"},
		},
		{
			name:    "case insensitive",
			content: `<div><SCRIPT>code</SCRIPT>Text</div>`,
			tag:     "script",
			want:    []string{"Text", "<div>"},
			avoid:   []string{"SCRIPT", "code"},
		},
		{
			name:    "mixed case",
			content: `<div><ScRiPt>code</sCrIpT>Text</div>`,
			tag:     "script",
			want:    []string{"Text", "<div>"},
			avoid:   []string{"code"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveTagContent(tt.content, tt.tag)

			for _, want := range tt.want {
				if !strings.Contains(result, want) {
					t.Errorf("RemoveTagContent() result should contain %q, got %q", want, result)
				}
			}

			for _, avoid := range tt.avoid {
				if strings.Contains(result, avoid) {
					t.Errorf("RemoveTagContent() result should not contain %q, got %q", avoid, result)
				}
			}
		})
	}
}

func TestRemoveTagContentEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("nested same tags", func(t *testing.T) {
		content := `<div><script><script>nested</script></script>Text</div>`
		result := RemoveTagContent(content, "script")

		if !strings.Contains(result, "Text") {
			t.Error("Should keep text after nested tags")
		}
	})

	t.Run("tag in attribute", func(t *testing.T) {
		content := `<div data-script="value">Text</div>`
		result := RemoveTagContent(content, "script")

		if !strings.Contains(result, "Text") {
			t.Error("Should not remove tag name in attributes")
		}
	})

	t.Run("very long content", func(t *testing.T) {
		longText := strings.Repeat("word ", 10000)
		content := `<div>` + longText + `<script>remove</script>` + longText + `</div>`
		result := RemoveTagContent(content, "script")

		if strings.Contains(result, "remove") {
			t.Error("Should remove script even in long content")
		}
		if !strings.Contains(result, "word") {
			t.Error("Should keep long text content")
		}
	})
}

func BenchmarkSanitizeHTML(b *testing.B) {
	htmlContent := `<html><body><div>Content<script>alert('test')</script><style>body{}</style><noscript>No JS</noscript>More</div></body></html>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizeHTML(htmlContent)
	}
}

func BenchmarkRemoveTagContent(b *testing.B) {
	content := `<div>Before<script>remove this content</script>After</div>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RemoveTagContent(content, "script")
	}
}

func BenchmarkRemoveTagContentLarge(b *testing.B) {
	longText := strings.Repeat("word ", 1000)
	content := `<div>` + longText + `<script>remove</script>` + longText + `</div>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RemoveTagContent(content, "script")
	}
}

// ============================================================================
// SECURITY TESTS
// ============================================================================

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
		"svg", "math", // SVG and MathML can contain scripts
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

func TestSanitizeHTML_SVGXSSPrevention(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "svg with onload",
			input: `<svg onload="alert('xss')"><circle cx="50" cy="50" r="50"/></svg>`,
		},
		{
			name:  "svg with script",
			input: `<svg><script>alert('xss')</script></svg>`,
		},
		{
			name:  "svg with foreignObject",
			input: `<svg><foreignObject><body onload="alert('xss')"></foreignObject></svg>`,
		},
		{
			name:  "svg with animate",
			input: `<svg><animate onbegin="alert('xss')"/></svg>`,
		},
		{
			name:  "svg inline event",
			input: `<svg><set onbegin="alert('xss')"/></svg>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := SanitizeHTML(tt.input)
			if strings.Contains(strings.ToLower(output), "<svg") {
				t.Errorf("SVG tag should be removed, but found in: %s", output)
			}
			if strings.Contains(strings.ToLower(output), "alert") {
				t.Errorf("Script content should be removed, but found in: %s", output)
			}
		})
	}
}

func TestSanitizeHTML_MathMLXSSPrevention(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "math with annotation-xml",
			input: `<math><annotation-xml encoding="application/xhtml+xml"><script>alert('xss')</script></annotation-xml></math>`,
		},
		{
			name:  "math with href",
			input: `<math href="javascript:alert('xss')"><mtext>click</mtext></math>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := SanitizeHTML(tt.input)
			if strings.Contains(strings.ToLower(output), "<math") {
				t.Errorf("Math tag should be removed, but found in: %s", output)
			}
			if strings.Contains(strings.ToLower(output), "alert") {
				t.Errorf("Script content should be removed, but found in: %s", output)
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
