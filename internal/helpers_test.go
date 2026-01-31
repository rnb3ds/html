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
		{"&#160;", "\u00a0"},
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

func TestPostProcessText(t *testing.T) {
	t.Parallel()

	t.Run("basic functionality", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			check func(string) bool
		}{
			{
				name:  "empty string",
				input: "",
				check: func(s string) bool { return s == "" },
			},
			{
				name:  "whitespace only",
				input: "   \n   \n   ",
				check: func(s string) bool { return s == "" },
			},
			{
				name:  "simple text",
				input: "Hello World",
				check: func(s string) bool { return strings.Contains(s, "Hello") && strings.Contains(s, "World") },
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := CleanText(tt.input, nil)
				if !tt.check(result) {
					t.Errorf("CleanText() = %q, failed check", result)
				}
			})
		}
	})

	t.Run("with regex", func(t *testing.T) {
		whitespaceRegex := regexp.MustCompile(`\s+`)

		tests := []struct {
			name  string
			input string
			check func(string) bool
		}{
			{
				name:  "multiple spaces per line",
				input: "Hello    World\nFoo    Bar",
				check: func(s string) bool {
					return strings.Contains(s, "Hello World") && strings.Contains(s, "Foo Bar")
				},
			},
			{
				name:  "excessive empty lines collapsed to single blank line",
				input: "Line1\n\n\nLine2",
				check: func(s string) bool {
					// Should preserve ONE blank line (two newlines total) for paragraph spacing
					lines := strings.Split(s, "\n")
					emptyCount := 0
					for _, line := range lines {
						if strings.TrimSpace(line) == "" {
							emptyCount++
						}
					}
					// Should have exactly 2 lines ("Line1" and "Line2") with ONE empty line between them
					return len(lines) == 3 && emptyCount == 1
				},
			},
			{
				name:  "tabs converted to spaces",
				input: "Hello\t\tWorld",
				check: func(s string) bool {
					return strings.Contains(s, "Hello") && strings.Contains(s, "World") && !strings.Contains(s, "\t")
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := CleanText(tt.input, whitespaceRegex)
				if !tt.check(result) {
					t.Errorf("CleanText() = %q, failed check", result)
				}
			})
		}
	})

	t.Run("long content", func(t *testing.T) {
		whitespaceRegex := regexp.MustCompile(`\s+`)

		t.Run("many lines", func(t *testing.T) {
			var sb strings.Builder
			for i := 0; i < 1000; i++ {
				sb.WriteString("Line ")
				sb.WriteString(strings.Repeat(" ", 10))
				sb.WriteString("Content\n")
			}

			result := CleanText(sb.String(), whitespaceRegex)
			if len(result) == 0 {
				t.Error("CleanText() should handle long content")
			}
		})

		t.Run("unicode and newlines", func(t *testing.T) {
			input := "中文   测试\n日本語   テスト\n한국어   테스트"
			result := CleanText(input, whitespaceRegex)

			if !strings.Contains(result, "中文") || !strings.Contains(result, "日本語") || !strings.Contains(result, "한국어") {
				t.Error("CleanText() should preserve unicode characters")
			}
		})
	})
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

func BenchmarkPostProcessText(b *testing.B) {
	text := strings.Repeat("Line   with   spaces\n", 100)
	whitespaceRegex := regexp.MustCompile(`\s+`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CleanText(text, whitespaceRegex)
	}
}
