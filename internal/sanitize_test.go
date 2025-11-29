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
