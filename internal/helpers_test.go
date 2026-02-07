package internal

import (
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestWalkNodes(t *testing.T) {
	t.Parallel()

	doc, _ := html.Parse(strings.NewReader(`<html><body><p>Test</p></body></html>`))

	count := 0
	WalkNodes(doc, func(n *html.Node) bool {
		count++
		return true
	})

	if count == 0 {
		t.Error("WalkNodes() should visit nodes")
	}
}

func TestWalkNodesEarlyStop(t *testing.T) {
	t.Parallel()

	doc, _ := html.Parse(strings.NewReader(`<html><body><p>Test</p></body></html>`))

	count := 0
	WalkNodes(doc, func(n *html.Node) bool {
		count++
		return count < 2 // Stop after 2 nodes
	})

	if count != 2 {
		t.Errorf("WalkNodes() visited %d nodes, want 2", count)
	}
}

func TestWalkNodesNil(t *testing.T) {
	t.Parallel()

	// Should not panic
	WalkNodes(nil, func(n *html.Node) bool {
		t.Error("Should not visit nodes when root is nil")
		return true
	})
}

func TestFindElementByTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		html    string
		tag     string
		wantNil bool
	}{
		{
			name:    "find title",
			html:    `<html><head><title>Test</title></head></html>`,
			tag:     "title",
			wantNil: false,
		},
		{
			name:    "find body",
			html:    `<html><body><p>Test</p></body></html>`,
			tag:     "body",
			wantNil: false,
		},
		{
			name:    "tag not found",
			html:    `<html><body><p>Test</p></body></html>`,
			tag:     "article",
			wantNil: true,
		},
		{
			name:    "nil document",
			html:    "",
			tag:     "p",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc *html.Node
			if tt.html != "" {
				doc, _ = html.Parse(strings.NewReader(tt.html))
			}

			result := FindElementByTag(doc, tt.tag)

			if tt.wantNil && result != nil {
				t.Errorf("FindElementByTag() = %v, want nil", result)
			}
			if !tt.wantNil && result == nil {
				t.Errorf("FindElementByTag() = nil, want non-nil")
			}
			if result != nil && result.Data != tt.tag {
				t.Errorf("FindElementByTag() found %q, want %q", result.Data, tt.tag)
			}
		})
	}
}

func TestGetTextContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "simple text",
			html: `<p>Hello World</p>`,
			want: "Hello World",
		},
		{
			name: "nested text",
			html: `<div><p>Hello <strong>World</strong></p></div>`,
			want: "Hello World",
		},
		{
			name: "empty",
			html: `<p></p>`,
			want: "",
		},
		{
			name: "whitespace only",
			html: `<p>   </p>`,
			want: "",
		},
		{
			name: "inline elements without space",
			html: `<span>F-<a href="#">2</a></span>`,
			want: "F-2",
		},
		{
			name: "inline elements with space in HTML",
			html: `<span>F- <a href="#">2</a></span>`,
			want: "F- 2",
		},
		{
			name: "nested span without space",
			html: `<div><span>Hello</span><span>World</span></div>`,
			want: "HelloWorld",
		},
		{
			name: "nested span with space in HTML",
			html: `<div><span>Hello</span> <span>World</span></div>`,
			want: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			result := GetTextContent(doc)

			if result != tt.want {
				t.Errorf("GetTextContent() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestGetTextContentNil(t *testing.T) {
	t.Parallel()

	result := GetTextContent(nil)
	if result != "" {
		t.Errorf("GetTextContent(nil) = %q, want empty string", result)
	}
}

func TestGetTextLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want int
	}{
		{
			name: "simple text",
			html: `<p>Hello</p>`,
			want: 5,
		},
		{
			name: "nested text",
			html: `<div><p>Hello</p><p>World</p></div>`,
			want: 10,
		},
		{
			name: "empty",
			html: `<p></p>`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			result := GetTextLength(doc)

			if result != tt.want {
				t.Errorf("GetTextLength() = %d, want %d", result, tt.want)
			}
		})
	}
}

func TestGetTextLengthNil(t *testing.T) {
	t.Parallel()

	result := GetTextLength(nil)
	if result != 0 {
		t.Errorf("GetTextLength(nil) = %d, want 0", result)
	}
}

func TestGetLinkDensity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want float64
	}{
		{
			name: "no links",
			html: `<p>Hello World</p>`,
			want: 0.0,
		},
		{
			name: "all links",
			html: `<p><a href="test.html">Hello World</a></p>`,
			want: 1.0,
		},
		{
			name: "half links",
			html: `<p>Hello <a href="test.html">World</a></p>`,
			want: 0.5,
		},
		{
			name: "empty",
			html: `<p></p>`,
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			result := GetLinkDensity(doc)

			if result < tt.want-0.1 || result > tt.want+0.1 {
				t.Errorf("GetLinkDensity() = %f, want ~%f", result, tt.want)
			}
		})
	}
}

func TestGetLinkDensityNil(t *testing.T) {
	t.Parallel()

	result := GetLinkDensity(nil)
	if result != 0.0 {
		t.Errorf("GetLinkDensity(nil) = %f, want 0.0", result)
	}
}

func TestCleanText(t *testing.T) {
	t.Parallel()

	t.Run("without regex", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  string
		}{
			{
				name:  "HTML entities",
				input: "&lt;html&gt; &amp;",
				want:  "<html> &",
			},
			{
				name:  "empty",
				input: "",
				want:  "",
			},
			{
				name:  "simple text",
				input: "Hello World",
				want:  "Hello World",
			},
			{
				name:  "newlines preserved",
				input: "Line1\nLine2",
				want:  "Line1\nLine2",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := CleanText(tt.input, nil)
				if result != tt.want {
					t.Errorf("CleanText() = %q, want %q", result, tt.want)
				}
			})
		}
	})

	t.Run("with regex", func(t *testing.T) {
		whitespaceRegex := regexp.MustCompile(`\s+`)

		tests := []struct {
			name  string
			input string
			want  string
		}{
			{
				name:  "multiple spaces",
				input: "Hello    World",
				want:  "Hello World",
			},
			{
				name:  "tabs and spaces",
				input: "Hello\t\t\tWorld",
				want:  "Hello World",
			},
			{
				name:  "mixed whitespace",
				input: "Hello  \t  \n  World",
				want:  "Hello\nWorld",
			},
			{
				name:  "leading spaces",
				input: "    Hello",
				want:  "Hello",
			},
			{
				name:  "trailing spaces",
				input: "Hello    ",
				want:  "Hello",
			},
			{
				name:  "multiple newlines collapsed to one blank line",
				input: "Line1\n\n\nLine2",
				want:  "Line1\n\nLine2", // Preserves paragraph spacing with one blank line
			},
			{
				name:  "only whitespace",
				input: "     ",
				want:  "",
			},
			{
				name:  "unicode characters",
				input: "Hello   世界   Test",
				want:  "Hello 世界 Test",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := CleanText(tt.input, whitespaceRegex)
				if result != tt.want {
					t.Errorf("CleanText() = %q, want %q", result, tt.want)
				}
			})
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		whitespaceRegex := regexp.MustCompile(`\s+`)

		t.Run("very long text", func(t *testing.T) {
			longText := strings.Repeat("word ", 10000)
			result := CleanText(longText, whitespaceRegex)
			if len(result) == 0 {
				t.Error("CleanText() should handle long text")
			}
		})

		t.Run("special characters", func(t *testing.T) {
			input := "Test   @#$%   Special"
			result := CleanText(input, whitespaceRegex)
			if !strings.Contains(result, "@#$%") {
				t.Error("CleanText() should preserve special chars")
			}
		})
	})
}

func TestReplaceHTMLEntities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"&nbsp;", " "},
		{"&amp;", "&"},
		{"&lt;", "<"},
		{"&gt;", ">"},
		{"&quot;", "\""},
		{"&apos;", "'"},
		{"&mdash;", "—"},
		{"&ndash;", "–"},
		{"&#8212;", "—"},
		{"&#x2014;", "—"},
		{"&#160;", " "},
		{"&#xa0;", " "},
		{"&hellip;", "…"},
		{"&copy;", "©"},
		{"no entities", "no entities"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ReplaceHTMLEntities(tt.input)
			if result != tt.want {
				t.Errorf("ReplaceHTMLEntities(%q) = %q, want %q", tt.input, result, tt.want)
			}
		})
	}
}

func TestIsExternalURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url  string
		want bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"//example.com", true},
		{"/page.html", false},
		{"page.html", false},
		{"#anchor", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := IsExternalURL(tt.url)
			if result != tt.want {
				t.Errorf("IsExternalURL(%q) = %v, want %v", tt.url, result, tt.want)
			}
		})
	}
}

func TestSelectBestCandidate(t *testing.T) {
	t.Parallel()

	doc, _ := html.Parse(strings.NewReader(`<html><body><div id="a"></div><div id="b"></div></body></html>`))

	var nodeA, nodeB *html.Node
	WalkNodes(doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == "a" {
					nodeA = n
				} else if attr.Key == "id" && attr.Val == "b" {
					nodeB = n
				}
			}
		}
		return true
	})

	tests := []struct {
		name       string
		candidates map[*html.Node]int
		wantNil    bool
	}{
		{
			name:       "empty candidates",
			candidates: map[*html.Node]int{},
			wantNil:    true,
		},
		{
			name: "single candidate",
			candidates: map[*html.Node]int{
				nodeA: 100,
			},
			wantNil: false,
		},
		{
			name: "multiple candidates",
			candidates: map[*html.Node]int{
				nodeA: 100,
				nodeB: 200,
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SelectBestCandidate(tt.candidates)

			if tt.wantNil && result != nil {
				t.Error("SelectBestCandidate() should return nil for empty candidates")
			}
			if !tt.wantNil && result == nil {
				t.Error("SelectBestCandidate() should return non-nil")
			}
		})
	}
}

// Tests for non-breaking space handling in helper functions
// These tests verify that &nbsp;, &#160;, and &#xa0; are properly converted to regular spaces

func TestGetTextLengthWithNbsp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		html     string
		expected int
	}{
		{
			name:     "Regular spaces",
			html:     "<p>Hello World</p>",
			expected: 11, // "Hello World"
		},
		{
			name:     "Named entity &nbsp;",
			html:     "<p>Hello&nbsp;World</p>",
			expected: 11, // "Hello World"
		},
		{
			name:     "Decimal entity &#160;",
			html:     "<p>Hello&#160;World</p>",
			expected: 11, // "Hello World"
		},
		{
			name:     "Hexadecimal entity &#xa0;",
			html:     "<p>Hello&#xa0;World</p>",
			expected: 11, // "Hello World"
		},
		{
			name:     "Multiple nbsp",
			html:     "<p>A&nbsp;&nbsp;&nbsp;B</p>",
			expected: 5, // "A   B" (parser creates single text node)
		},
		{
			name:     "Mixed nbsp and regular spaces",
			html:     "<p>A&nbsp; B &#xa0;C</p>",
			expected: 7, // "A  B  C" (parser creates single text node)
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := html.Parse(strings.NewReader(tc.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			result := GetTextLength(doc)
			if result != tc.expected {
				t.Errorf("GetTextLength() = %d, want %d", result, tc.expected)
			}
		})
	}
}

func TestGetLinkDensityWithNbsp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		html     string
		expected float64
	}{
		{
			name:     "No links",
			html:     "<p>Hello&nbsp;World</p>",
			expected: 0.0,
		},
		{
			name:     "All text in link with nbsp",
			html:     "<p><a href='#'>Hello&nbsp;World</a></p>",
			expected: 1.0, // 100% link density
		},
		{
			name:     "Partial link with nbsp",
			html:     "<p>Hello&nbsp;<a href='#'>World</a></p>",
			expected: 5.0 / 10.0, // "World" is 5 chars, "Hello" is 5 chars (trailing space removed)
		},
		{
			name:     "Multiple spaces in link",
			html:     "<p><a href='#'>A&nbsp;&nbsp;B</a> C</p>",
			expected: 0.8, // Actual value after trim and replace
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := html.Parse(strings.NewReader(tc.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			result := GetLinkDensity(doc)
			if result != tc.expected {
				t.Errorf("GetLinkDensity() = %f, want %f", result, tc.expected)
			}
		})
	}
}

func TestCalculateContentDensityWithNbsp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		html        string
		minExpected float64
		description string
	}{
		{
			name:        "Pure text with nbsp",
			html:        "<p>Hello&nbsp;World</p>",
			minExpected: 0.2, // Reasonable text density
			description: "Should have reasonable text density",
		},
		{
			name:        "Text with tags and nbsp",
			html:        `<div><p>Hello&nbsp;World</p><p>Test&nbsp;Content</p></div>`,
			minExpected: 0.3, // Good text density
			description: "Should have good text density",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := html.Parse(strings.NewReader(tc.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			result := CalculateContentDensity(doc)
			if result < tc.minExpected {
				t.Errorf("%s: CalculateContentDensity() = %f, want >= %f", tc.description, result, tc.minExpected)
			}
		})
	}
}

func BenchmarkGetTextContent(b *testing.B) {
	doc, _ := html.Parse(strings.NewReader(`<html><body><p>Hello World</p><p>More text</p></body></html>`))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTextContent(doc)
	}
}

func BenchmarkCleanText(b *testing.B) {
	text := "Hello    World\n\nWith   multiple   spaces"
	whitespaceRegex := regexp.MustCompile(`\s+`)

	b.Run("without regex", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			CleanText(text, nil)
		}
	})

	b.Run("with regex", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			CleanText(text, whitespaceRegex)
		}
	})
}
