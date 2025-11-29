package html_test

import (
	"strings"
	"testing"

	"github.com/cybergodev/html"
)

func TestInlineImageFormatting(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	t.Run("markdown format with multiple images", func(t *testing.T) {
		htmlContent := `<div><p>Text1</p><img src="img1.jpg" alt="Alt1"><p>Text2</p><img src="img2.jpg" alt="Alt2"><p>Text3</p></div>`
		config := html.DefaultExtractConfig()
		config.InlineImageFormat = "markdown"

		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "![Alt1](img1.jpg)") {
			t.Errorf("Text should contain markdown image 1, got %q", result.Text)
		}
		if !strings.Contains(result.Text, "![Alt2](img2.jpg)") {
			t.Errorf("Text should contain markdown image 2, got %q", result.Text)
		}
	})

	t.Run("html format with dimensions", func(t *testing.T) {
		htmlContent := `<div><img src="test.jpg" alt="Test" width="100" height="200"></div>`
		config := html.DefaultExtractConfig()
		config.InlineImageFormat = "html"

		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, `<img src="test.jpg"`) {
			t.Errorf("Text should contain HTML img tag, got %q", result.Text)
		}
		if !strings.Contains(result.Text, `alt="Test"`) {
			t.Errorf("Text should contain alt attribute, got %q", result.Text)
		}
	})

	t.Run("placeholder format", func(t *testing.T) {
		htmlContent := `<div><p>Before</p><img src="test.jpg" alt="Test"><p>After</p></div>`
		config := html.DefaultExtractConfig()
		config.InlineImageFormat = "placeholder"

		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "[IMAGE:1]") {
			t.Errorf("Text should contain placeholder, got %q", result.Text)
		}
	})

	t.Run("none format no placeholders", func(t *testing.T) {
		htmlContent := `<div><p>Before</p><img src="test.jpg" alt="Test"><p>After</p></div>`
		config := html.DefaultExtractConfig()
		config.InlineImageFormat = "none"

		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if strings.Contains(result.Text, "[IMAGE:") {
			t.Errorf("Text should not contain placeholders, got %q", result.Text)
		}
		if strings.Contains(result.Text, "![") {
			t.Errorf("Text should not contain markdown, got %q", result.Text)
		}
	})

	t.Run("empty format defaults to none", func(t *testing.T) {
		htmlContent := `<div><img src="test.jpg" alt="Test"></div>`
		config := html.DefaultExtractConfig()
		config.InlineImageFormat = ""

		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if strings.Contains(result.Text, "[IMAGE:") {
			t.Errorf("Empty format should default to none, got %q", result.Text)
		}
	})

	t.Run("markdown with empty alt", func(t *testing.T) {
		htmlContent := `<div><img src="test.jpg"></div>`
		config := html.DefaultExtractConfig()
		config.InlineImageFormat = "markdown"

		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "![Image 1](test.jpg)") {
			t.Errorf("Empty alt should use default, got %q", result.Text)
		}
	})

	t.Run("no images with inline format", func(t *testing.T) {
		htmlContent := `<div><p>Just text</p></div>`
		config := html.DefaultExtractConfig()
		config.InlineImageFormat = "markdown"

		result, err := p.Extract(htmlContent, config)
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if strings.Contains(result.Text, "![") || strings.Contains(result.Text, "[IMAGE:") {
			t.Errorf("No images should produce no placeholders, got %q", result.Text)
		}
	})
}

func TestExtractWithComplexFormatting(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	t.Run("table extraction", func(t *testing.T) {
		htmlContent := `<table>
			<tr><th>Header1</th><th>Header2</th></tr>
			<tr><td>Cell1</td><td>Cell2</td></tr>
			<tr><td>Cell3</td><td>Cell4</td></tr>
		</table>`

		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "Header1") || !strings.Contains(result.Text, "Header2") {
			t.Errorf("Table headers should be extracted, got %q", result.Text)
		}
		if !strings.Contains(result.Text, "Cell1") || !strings.Contains(result.Text, "Cell4") {
			t.Errorf("Table cells should be extracted, got %q", result.Text)
		}
		if !strings.Contains(result.Text, "|") {
			t.Errorf("Table should use markdown format, got %q", result.Text)
		}
	})

	t.Run("pre tag preserves formatting", func(t *testing.T) {
		htmlContent := `<pre>Line1
Line2
Line3</pre>`

		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "Line1") || !strings.Contains(result.Text, "Line3") {
			t.Errorf("Pre tag content should be preserved, got %q", result.Text)
		}
	})

	t.Run("code block", func(t *testing.T) {
		htmlContent := `<code>function test() { return true; }</code>`

		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "function test") {
			t.Errorf("Code block should be extracted, got %q", result.Text)
		}
	})

	t.Run("blockquote", func(t *testing.T) {
		htmlContent := `<blockquote>This is a quote</blockquote>`

		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "This is a quote") {
			t.Errorf("Blockquote should be extracted, got %q", result.Text)
		}
	})

	t.Run("lists", func(t *testing.T) {
		htmlContent := `<ul><li>Item 1</li><li>Item 2</li><li>Item 3</li></ul>`

		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "Item 1") || !strings.Contains(result.Text, "Item 3") {
			t.Errorf("List items should be extracted, got %q", result.Text)
		}
	})
}

func TestExtractWithSpecialCharacters(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	tests := []struct {
		name        string
		html        string
		wantContain string
	}{
		{
			name:        "HTML entities",
			html:        `<p>&lt;html&gt; &amp; &quot;test&quot;</p>`,
			wantContain: "<html>",
		},
		{
			name:        "unicode characters",
			html:        `<p>Hello 世界 🌍</p>`,
			wantContain: "世界",
		},
		{
			name:        "special symbols",
			html:        `<p>Price: $100 & €50</p>`,
			wantContain: "$100",
		},
		{
			name:        "nbsp entity",
			html:        `<p>Word1&nbsp;Word2</p>`,
			wantContain: "Word1",
		},
		{
			name:        "mdash entity",
			html:        `<p>Text&mdash;more text</p>`,
			wantContain: "Text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Extract(tt.html, html.DefaultExtractConfig())
			if err != nil {
				t.Fatalf("Extract() failed: %v", err)
			}

			if !strings.Contains(result.Text, tt.wantContain) {
				t.Errorf("Text should contain %q, got %q", tt.wantContain, result.Text)
			}
		})
	}
}

func TestExtractWithNestedStructures(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	t.Run("deeply nested divs", func(t *testing.T) {
		htmlContent := `<div><div><div><div><p>Deep content</p></div></div></div></div>`

		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "Deep content") {
			t.Errorf("Deeply nested content should be extracted, got %q", result.Text)
		}
	})

	t.Run("nested lists", func(t *testing.T) {
		htmlContent := `<ul><li>Item 1<ul><li>Nested 1</li><li>Nested 2</li></ul></li><li>Item 2</li></ul>`

		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "Item 1") || !strings.Contains(result.Text, "Nested 1") {
			t.Errorf("Nested list items should be extracted, got %q", result.Text)
		}
	})

	t.Run("nested tables", func(t *testing.T) {
		htmlContent := `<table><tr><td>Outer<table><tr><td>Inner</td></tr></table></td></tr></table>`

		result, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		if !strings.Contains(result.Text, "Outer") || !strings.Contains(result.Text, "Inner") {
			t.Errorf("Nested table content should be extracted, got %q", result.Text)
		}
	})
}
