package internal

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestExtractTextWithStructure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "simple paragraph",
			html: `<p>Hello World</p>`,
			want: "Hello World",
		},
		{
			name: "nested elements",
			html: `<div><p>First</p><p>Second</p></div>`,
			want: "First\nSecond",
		},
		{
			name: "block elements add newlines",
			html: `<div>Text1</div><div>Text2</div>`,
			want: "Text1\nText2",
		},
		{
			name: "inline elements add spaces",
			html: `<p>Hello <strong>World</strong> Test</p>`,
			want: "Hello World Test",
		},
		{
			name: "script tags excluded",
			html: `<div>Visible<script>hidden</script></div>`,
			want: "Visible",
		},
		{
			name: "style tags excluded",
			html: `<div>Visible<style>body{}</style></div>`,
			want: "Visible",
		},
		{
			name: "nav tags excluded",
			html: `<div>Content<nav>Menu</nav></div>`,
			want: "Content",
		},
		{
			name: "empty",
			html: `<div></div>`,
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
			var sb strings.Builder
			ExtractTextWithStructure(doc, &sb, 0)
			result := strings.TrimSpace(sb.String())

			if result != tt.want {
				t.Errorf("ExtractTextWithStructure() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestExtractTextWithStructureAndImages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		html       string
		wantText   string
		wantImages int
	}{
		{
			name:       "single image",
			html:       `<div><img src="test.jpg" alt="Test"></div>`,
			wantText:   "[IMAGE:1]",
			wantImages: 1,
		},
		{
			name:       "multiple images",
			html:       `<div><img src="1.jpg"><img src="2.jpg"></div>`,
			wantText:   "[IMAGE:1]\n[IMAGE:2]",
			wantImages: 2,
		},
		{
			name:       "text with images",
			html:       `<div>Before<img src="test.jpg">After</div>`,
			wantText:   "Before\n[IMAGE:1]\nAfter",
			wantImages: 1,
		},
		{
			name:       "no images",
			html:       `<div>Just text</div>`,
			wantText:   "Just text",
			wantImages: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			var sb strings.Builder
			imageCounter := 0
			ExtractTextWithStructureAndImages(doc, &sb, 0, &imageCounter)
			result := strings.TrimSpace(sb.String())

			if result != tt.wantText {
				t.Errorf("ExtractTextWithStructureAndImages() text = %q, want %q", result, tt.wantText)
			}
			if imageCounter != tt.wantImages {
				t.Errorf("ExtractTextWithStructureAndImages() images = %d, want %d", imageCounter, tt.wantImages)
			}
		})
	}
}

func TestExtractTextWithStructureAndImagesNil(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	ExtractTextWithStructureAndImages(nil, &sb, 0, nil)

	if sb.Len() != 0 {
		t.Error("ExtractTextWithStructureAndImages(nil) should not write anything")
	}
}

func TestExtractTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		html     string
		wantRows int
	}{
		{
			name: "simple table",
			html: `<table>
				<tr><th>Header1</th><th>Header2</th></tr>
				<tr><td>Cell1</td><td>Cell2</td></tr>
			</table>`,
			wantRows: 2,
		},
		{
			name: "table with uneven columns",
			html: `<table>
				<tr><td>A</td><td>B</td><td>C</td></tr>
				<tr><td>D</td><td>E</td></tr>
			</table>`,
			wantRows: 2,
		},
		{
			name:     "empty table",
			html:     `<table></table>`,
			wantRows: 0,
		},
		{
			name: "table with empty cells",
			html: `<table>
				<tr><td>A</td><td></td></tr>
			</table>`,
			wantRows: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			var sb strings.Builder
			ExtractTextWithStructure(doc, &sb, 0)
			result := sb.String()

			if tt.wantRows > 0 {
				// Should have separator line
				if !strings.Contains(result, "| --- |") {
					t.Error("Table should have separator line")
				}
			}

			if tt.wantRows == 0 && strings.TrimSpace(result) != "" {
				t.Error("Empty table should produce no output")
			}
		})
	}
}

func TestExtractTableNil(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	extractTable(nil, &sb)

	if sb.Len() != 0 {
		t.Error("extractTable(nil) should not write anything")
	}
}

func TestPostProcessText(t *testing.T) {
	t.Parallel()

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
		{
			name:  "text with newlines",
			input: "Line1\nLine2",
			check: func(s string) bool { return strings.Contains(s, "Line1") && strings.Contains(s, "Line2") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PostProcessText(tt.input, nil)
			if !tt.check(result) {
				t.Errorf("PostProcessText() = %q, failed check", result)
			}
		})
	}
}

func TestCleanContentNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		html        string
		wantRemoved []string
		wantKept    []string
	}{
		{
			name:        "remove script",
			html:        `<div><p>Keep</p><script>Remove</script></div>`,
			wantRemoved: []string{"script"},
			wantKept:    []string{"Keep"},
		},
		{
			name:        "remove nav",
			html:        `<div><p>Keep</p><nav>Remove</nav></div>`,
			wantRemoved: []string{"nav"},
			wantKept:    []string{"Keep"},
		},
		{
			name:        "remove by class",
			html:        `<div><p>Keep</p><div class="sidebar">Remove</div></div>`,
			wantRemoved: []string{"sidebar"},
			wantKept:    []string{"Keep"},
		},
		{
			name:        "remove hidden",
			html:        `<div><p>Keep</p><div hidden>Remove</div></div>`,
			wantRemoved: []string{"hidden"},
			wantKept:    []string{"Keep"},
		},
		{
			name:     "keep all",
			html:     `<div><p>Keep1</p><p>Keep2</p></div>`,
			wantKept: []string{"Keep1", "Keep2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			cleaned := CleanContentNode(doc)

			text := GetTextContent(cleaned)

			for _, kept := range tt.wantKept {
				if !strings.Contains(text, kept) {
					t.Errorf("CleanContentNode() should keep %q", kept)
				}
			}

			for _, removed := range tt.wantRemoved {
				if strings.Contains(text, removed) {
					t.Errorf("CleanContentNode() should remove %q", removed)
				}
			}
		})
	}
}

func TestCleanContentNodeNil(t *testing.T) {
	t.Parallel()

	result := CleanContentNode(nil)
	if result != nil {
		t.Error("CleanContentNode(nil) should return nil")
	}
}

func BenchmarkExtractTextWithStructure(b *testing.B) {
	htmlContent := `<html><body><article><h1>Title</h1><p>Paragraph 1</p><p>Paragraph 2</p></article></body></html>`
	doc, _ := html.Parse(strings.NewReader(htmlContent))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		ExtractTextWithStructure(doc, &sb, 0)
	}
}

func BenchmarkPostProcessText(b *testing.B) {
	text := "Line1\n\nLine2    with   spaces\n\n\nLine3"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PostProcessText(text, nil)
	}
}
