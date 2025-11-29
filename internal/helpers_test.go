package internal

import (
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
		{"&mdash;", "-"},
		{"&ndash;", "-"},
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

func BenchmarkGetTextContent(b *testing.B) {
	doc, _ := html.Parse(strings.NewReader(`<html><body><p>Hello World</p><p>More text</p></body></html>`))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTextContent(doc)
	}
}

func BenchmarkCleanText(b *testing.B) {
	text := "Hello    World\n\nWith   multiple   spaces"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CleanText(text, nil)
	}
}
