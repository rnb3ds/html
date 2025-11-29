package internal

import (
	"regexp"
	"strings"
	"testing"
)

func TestCleanTextWithRegex(t *testing.T) {
	t.Parallel()

	whitespaceRegex := regexp.MustCompile(`\s+`)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "multiple spaces with regex",
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
			want:  "Hello\nWorld", // Newlines are preserved
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
			name:  "newlines preserved",
			input: "Line1\nLine2",
			want:  "Line1\nLine2",
		},
		{
			name:  "multiple newlines",
			input: "Line1\n\n\nLine2",
			want:  "Line1\nLine2", // Multiple newlines collapsed
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
}

func TestCleanTextEdgeCases(t *testing.T) {
	t.Parallel()

	whitespaceRegex := regexp.MustCompile(`\s+`)

	t.Run("very long text", func(t *testing.T) {
		longText := strings.Repeat("word ", 10000)
		result := CleanText(longText, whitespaceRegex)
		if len(result) == 0 {
			t.Error("CleanText() should handle long text")
		}
	})

	t.Run("only whitespace", func(t *testing.T) {
		result := CleanText("     ", whitespaceRegex)
		if result != "" {
			t.Errorf("CleanText() = %q, want empty", result)
		}
	})

	t.Run("unicode characters", func(t *testing.T) {
		input := "Hello   世界   Test"
		result := CleanText(input, whitespaceRegex)
		if !strings.Contains(result, "世界") {
			t.Error("CleanText() should preserve unicode")
		}
	})

	t.Run("special characters", func(t *testing.T) {
		input := "Test   @#$%   Special"
		result := CleanText(input, whitespaceRegex)
		if !strings.Contains(result, "@#$%") {
			t.Error("CleanText() should preserve special chars")
		}
	})
}

func TestPostProcessTextWithRegex(t *testing.T) {
	t.Parallel()

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
			name:  "empty lines removed",
			input: "Line1\n\n\nLine2",
			check: func(s string) bool {
				lines := strings.Split(s, "\n")
				for _, line := range lines {
					if strings.TrimSpace(line) == "" {
						return false
					}
				}
				return true
			},
		},
		{
			name:  "leading trailing per line",
			input: "  Line1  \n  Line2  ",
			check: func(s string) bool {
				return strings.Contains(s, "Line1") && strings.Contains(s, "Line2")
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
			result := PostProcessText(tt.input, whitespaceRegex)
			if !tt.check(result) {
				t.Errorf("PostProcessText() = %q, failed check", result)
			}
		})
	}
}

func TestPostProcessTextLongContent(t *testing.T) {
	t.Parallel()

	whitespaceRegex := regexp.MustCompile(`\s+`)

	t.Run("very long text with many lines", func(t *testing.T) {
		var sb strings.Builder
		for i := 0; i < 1000; i++ {
			sb.WriteString("Line ")
			sb.WriteString(strings.Repeat(" ", 10))
			sb.WriteString("Content\n")
		}

		result := PostProcessText(sb.String(), whitespaceRegex)
		if len(result) == 0 {
			t.Error("PostProcessText() should handle long content")
		}

		lines := strings.Split(result, "\n")
		if len(lines) == 0 {
			t.Error("PostProcessText() should preserve line structure")
		}
	})

	t.Run("text with unicode and newlines", func(t *testing.T) {
		input := "中文   测试\n日本語   テスト\n한국어   테스트"
		result := PostProcessText(input, whitespaceRegex)

		if !strings.Contains(result, "中文") || !strings.Contains(result, "日本語") || !strings.Contains(result, "한국어") {
			t.Error("PostProcessText() should preserve unicode characters")
		}
	})
}

func BenchmarkCleanTextWithRegex(b *testing.B) {
	whitespaceRegex := regexp.MustCompile(`\s+`)
	text := "Hello    World\n\nWith   multiple   spaces"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CleanText(text, whitespaceRegex)
	}
}

func BenchmarkPostProcessTextWithRegex(b *testing.B) {
	whitespaceRegex := regexp.MustCompile(`\s+`)
	text := strings.Repeat("Line   with   spaces\n", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PostProcessText(text, whitespaceRegex)
	}
}
