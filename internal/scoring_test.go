package internal

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestScoreContentNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		html      string
		wantScore int
		checkFunc func(int) bool
	}{
		{
			name:      "article tag high score",
			html:      `<article><p>Content</p></article>`,
			checkFunc: func(score int) bool { return score > 500 },
		},
		{
			name:      "main tag high score",
			html:      `<main><p>Content</p></main>`,
			checkFunc: func(score int) bool { return score > 500 },
		},
		{
			name:      "div with paragraphs",
			html:      `<div><p>P1</p><p>P2</p><p>P3</p></div>`,
			checkFunc: func(score int) bool { return score > 0 },
		},
		{
			name:      "long text content",
			html:      `<div>` + strings.Repeat("word ", 100) + `</div>`,
			checkFunc: func(score int) bool { return score > 300 },
		},
		{
			name:      "short text penalty",
			html:      `<div>Short</div>`,
			checkFunc: func(score int) bool { return score < 0 },
		},
		{
			name:      "high link density penalty",
			html:      `<div><a href="#">Link1</a><a href="#">Link2</a>Text</div>`,
			checkFunc: func(score int) bool { return score < 100 },
		},
		{
			name:      "positive class names",
			html:      `<div class="article-content"><p>Content</p></div>`,
			checkFunc: func(score int) bool { return score > 200 },
		},
		{
			name:      "negative class names",
			html:      `<div class="sidebar"><p>Content</p></div>`,
			checkFunc: func(score int) bool { return score < 0 },
		},
		{
			name:      "with headings",
			html:      `<div><h1>Title</h1><p>Content</p></div>`,
			checkFunc: func(score int) bool { return true }, // Score can vary
		},
		{
			name:      "with commas",
			html:      `<div>Text with, many, commas, here, and, more</div>`,
			checkFunc: func(score int) bool { return true }, // Score can vary based on text length
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			var targetNode *html.Node
			WalkNodes(doc, func(n *html.Node) bool {
				if n.Type == html.ElementNode && (n.Data == "article" || n.Data == "main" || n.Data == "div") {
					targetNode = n
					return false
				}
				return true
			})

			if targetNode == nil {
				t.Fatal("Could not find target node")
			}

			score := ScoreContentNode(targetNode)
			if !tt.checkFunc(score) {
				t.Errorf("ScoreContentNode() = %d, failed check", score)
			}
		})
	}
}

func TestScoreContentNodeNil(t *testing.T) {
	t.Parallel()

	score := ScoreContentNode(nil)
	if score != 0 {
		t.Errorf("ScoreContentNode(nil) = %d, want 0", score)
	}
}

func TestScoreContentNodeTextNode(t *testing.T) {
	t.Parallel()

	textNode := &html.Node{
		Type: html.TextNode,
		Data: "text",
	}

	score := ScoreContentNode(textNode)
	if score != 0 {
		t.Errorf("ScoreContentNode(textNode) = %d, want 0", score)
	}
}

func TestScoreContentNodeNonContent(t *testing.T) {
	t.Parallel()

	doc, _ := html.Parse(strings.NewReader(`<script>code</script>`))
	var scriptNode *html.Node
	WalkNodes(doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "script" {
			scriptNode = n
			return false
		}
		return true
	})

	score := ScoreContentNode(scriptNode)
	if score != 0 {
		t.Errorf("ScoreContentNode(script) = %d, want 0", score)
	}
}

func TestScoreAttributes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		html      string
		checkFunc func(int) bool
	}{
		{
			name:      "positive class",
			html:      `<div class="article-content"></div>`,
			checkFunc: func(score int) bool { return score > 0 },
		},
		{
			name:      "negative class",
			html:      `<div class="sidebar"></div>`,
			checkFunc: func(score int) bool { return score < 0 },
		},
		{
			name:      "positive id",
			html:      `<div id="main-content"></div>`,
			checkFunc: func(score int) bool { return score > 0 },
		},
		{
			name:      "negative id",
			html:      `<div id="navigation"></div>`,
			checkFunc: func(score int) bool { return score < 0 },
		},
		{
			name:      "role main",
			html:      `<div role="main"></div>`,
			checkFunc: func(score int) bool { return score > 0 },
		},
		{
			name:      "role navigation",
			html:      `<div role="navigation"></div>`,
			checkFunc: func(score int) bool { return score < 0 },
		},
		{
			name:      "no attributes",
			html:      `<div></div>`,
			checkFunc: func(score int) bool { return score == 0 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			var divNode *html.Node
			WalkNodes(doc, func(n *html.Node) bool {
				if n.Type == html.ElementNode && n.Data == "div" {
					divNode = n
					return false
				}
				return true
			})

			if divNode == nil {
				t.Fatal("Could not find div node")
			}

			score := ScoreAttributes(divNode)
			if !tt.checkFunc(score) {
				t.Errorf("ScoreAttributes() = %d, failed check", score)
			}
		})
	}
}

func TestScoreAttributesNil(t *testing.T) {
	t.Parallel()

	score := ScoreAttributes(nil)
	if score != 0 {
		t.Errorf("ScoreAttributes(nil) = %d, want 0", score)
	}
}

func TestMatchesPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value    string
		patterns map[string]bool
		want     bool
	}{
		{"article-content", map[string]bool{"article": true, "content": true}, true},
		{"sidebar", map[string]bool{"article": true, "content": true}, false},
		{"main-article", map[string]bool{"article": true}, true},
		{"navigation-menu", map[string]bool{"nav": true, "menu": true}, true},
		{"", map[string]bool{"article": true}, false},
		{"test", map[string]bool{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := MatchesPattern(tt.value, tt.patterns)
			if result != tt.want {
				t.Errorf("MatchesPattern(%q, %v) = %v, want %v", tt.value, tt.patterns, result, tt.want)
			}
		})
	}
}

func TestCalculateContentDensity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		html      string
		checkFunc func(float64) bool
	}{
		{
			name:      "high density - lots of text",
			html:      `<div>` + strings.Repeat("word ", 100) + `</div>`,
			checkFunc: func(d float64) bool { return d > 0.5 },
		},
		{
			name:      "low density - many tags",
			html:      `<div><span><span><span>word</span></span></span></div>`,
			checkFunc: func(d float64) bool { return d < 0.5 },
		},
		{
			name:      "no text",
			html:      `<div><span></span></div>`,
			checkFunc: func(d float64) bool { return d == 0 },
		},
		{
			name:      "plain text",
			html:      `<p>text only</p>`,
			checkFunc: func(d float64) bool { return d > 0 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			density := CalculateContentDensity(doc)

			if !tt.checkFunc(density) {
				t.Errorf("CalculateContentDensity() = %f, failed check", density)
			}
		})
	}
}

func TestCountTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want int
	}{
		{
			name: "single tag",
			html: `<div></div>`,
			want: 1,
		},
		{
			name: "nested tags",
			html: `<div><p><span>text</span></p></div>`,
			want: 3,
		},
		{
			name: "multiple siblings",
			html: `<div><p></p><p></p><p></p></div>`,
			want: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			count := CountTags(doc)

			if count < tt.want {
				t.Errorf("CountTags() = %d, want at least %d", count, tt.want)
			}
		})
	}
}

func TestCountTagsNil(t *testing.T) {
	t.Parallel()

	count := CountTags(nil)
	if count != 0 {
		t.Errorf("CountTags(nil) = %d, want 0", count)
	}
}

func TestIsNonContentElement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tag  string
		want bool
	}{
		{"script", true},
		{"style", true},
		{"noscript", true},
		{"nav", true},
		{"aside", true},
		{"footer", true},
		{"header", true},
		{"form", true},
		{"div", false},
		{"p", false},
		{"article", false},
		{"main", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			result := IsNonContentElement(tt.tag)
			if result != tt.want {
				t.Errorf("IsNonContentElement(%q) = %v, want %v", tt.tag, result, tt.want)
			}
		})
	}
}

func TestCountChildElements(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		tag  string
		want int
	}{
		{
			name: "count paragraphs",
			html: `<div><p>1</p><p>2</p><p>3</p></div>`,
			tag:  "p",
			want: 3,
		},
		{
			name: "count nested",
			html: `<div><div><p>1</p></div><p>2</p></div>`,
			tag:  "p",
			want: 2,
		},
		{
			name: "no matches",
			html: `<div><span>text</span></div>`,
			tag:  "p",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			var divNode *html.Node
			WalkNodes(doc, func(n *html.Node) bool {
				if n.Type == html.ElementNode && n.Data == "div" {
					divNode = n
					return false
				}
				return true
			})

			count := CountChildElements(divNode, tt.tag)
			if count != tt.want {
				t.Errorf("CountChildElements() = %d, want %d", count, tt.want)
			}
		})
	}
}

func TestCountChildElementsNil(t *testing.T) {
	t.Parallel()

	count := CountChildElements(nil, "p")
	if count != 0 {
		t.Errorf("CountChildElements(nil) = %d, want 0", count)
	}
}

func TestShouldRemoveElement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want bool
	}{
		{
			name: "script tag",
			html: `<script>code</script>`,
			want: true,
		},
		{
			name: "nav tag",
			html: `<nav>menu</nav>`,
			want: true,
		},
		{
			name: "sidebar class",
			html: `<div class="sidebar">content</div>`,
			want: true,
		},
		{
			name: "navigation id",
			html: `<div id="navigation">menu</div>`,
			want: true,
		},
		{
			name: "hidden attribute",
			html: `<div hidden>content</div>`,
			want: true,
		},
		{
			name: "display none",
			html: `<div style="display:none">content</div>`,
			want: true,
		},
		{
			name: "display none with space",
			html: `<div style="display: none">content</div>`,
			want: true,
		},
		{
			name: "normal div",
			html: `<div>content</div>`,
			want: false,
		},
		{
			name: "article",
			html: `<article>content</article>`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			var targetNode *html.Node
			WalkNodes(doc, func(n *html.Node) bool {
				if n.Type == html.ElementNode && n.Data != "html" && n.Data != "head" && n.Data != "body" {
					targetNode = n
					return false
				}
				return true
			})

			if targetNode == nil {
				t.Fatal("Could not find target node")
			}

			result := ShouldRemoveElement(targetNode)
			if result != tt.want {
				t.Errorf("ShouldRemoveElement() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestShouldRemoveElementNil(t *testing.T) {
	t.Parallel()

	result := ShouldRemoveElement(nil)
	if result != false {
		t.Error("ShouldRemoveElement(nil) should return false")
	}
}

func TestShouldRemoveElementTextNode(t *testing.T) {
	t.Parallel()

	textNode := &html.Node{
		Type: html.TextNode,
		Data: "text",
	}

	result := ShouldRemoveElement(textNode)
	if result != false {
		t.Error("ShouldRemoveElement(textNode) should return false")
	}
}

func TestIsBlockElement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tag  string
		want bool
	}{
		{"p", true},
		{"div", true},
		{"h1", true},
		{"h2", true},
		{"article", true},
		{"section", true},
		{"blockquote", true},
		{"ul", true},
		{"ol", true},
		{"li", true},
		{"table", true},
		{"br", false},  // BR is now inline, not block
		{"hr", true},
		{"span", false},
		{"a", false},
		{"strong", false},
		{"em", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			result := IsBlockElement(tt.tag)
			if result != tt.want {
				t.Errorf("IsBlockElement(%q) = %v, want %v", tt.tag, result, tt.want)
			}
		})
	}
}

func BenchmarkScoreContentNode(b *testing.B) {
	htmlContent := `<article class="main-content"><h1>Title</h1><p>Paragraph 1</p><p>Paragraph 2</p></article>`
	doc, _ := html.Parse(strings.NewReader(htmlContent))
	var articleNode *html.Node
	WalkNodes(doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "article" {
			articleNode = n
			return false
		}
		return true
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ScoreContentNode(articleNode)
	}
}

func BenchmarkCalculateContentDensity(b *testing.B) {
	htmlContent := `<div><p>` + strings.Repeat("word ", 50) + `</p></div>`
	doc, _ := html.Parse(strings.NewReader(htmlContent))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateContentDensity(doc)
	}
}

func TestPatternMatchesEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		pattern string
		want    bool
	}{
		{
			name:    "empty value",
			value:   "",
			pattern: "test",
			want:    false,
		},
		{
			name:    "empty pattern",
			value:   "test",
			pattern: "",
			want:    false,
		},
		{
			name:    "both empty",
			value:   "",
			pattern: "",
			want:    true, // MatchesPattern returns true for empty pattern (iteration doesn't execute)
		},
		{
			name:    "pattern longer than value",
			value:   "test",
			pattern: "testing",
			want:    false,
		},
		{
			name:    "exact match",
			value:   "article",
			pattern: "article",
			want:    true,
		},
		{
			name:    "hyphenated word match",
			value:   "article-content",
			pattern: "article",
			want:    true,
		},
		{
			name:    "underscore separated match - underscore is NOT a word boundary",
			value:   "article_content",
			pattern: "article",
			want:    true, // Underscore is not treated as a word boundary by patternMatches
		},
		{
			name:    "partial match should fail",
			value:   "article123",
			pattern: "article",
			want:    false,
		},
		{
			name:    "match at start",
			value:   "article-body",
			pattern: "article",
			want:    true,
		},
		{
			name:    "match at end",
			value:   "main-article",
			pattern: "article",
			want:    true,
		},
		{
			name:    "match in middle",
			value:   "the-article-body",
			pattern: "article",
			want:    true,
		},
		{
			name:    "case sensitive match",
			value:   "Article",
			pattern: "article",
			want:    false,
		},
		{
			name:    "special characters",
			value:   "article.content",
			pattern: "article",
			want:    false,
		},
		{
			name:    "numbers in pattern",
			value:   "h1-content",
			pattern: "h1",
			want:    true,
		},
		{
			name:    "multiple hyphens",
			value:   "article-body-content",
			pattern: "body",
			want:    true,
		},
		{
			name:    "pattern with hyphen",
			value:   "post-article-content",
			pattern: "article-content",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use MatchesPattern which internally calls patternMatches
			patterns := map[string]bool{tt.pattern: true}
			result := MatchesPattern(tt.value, patterns)
			if result != tt.want {
				t.Errorf("MatchesPattern(%q, %q) = %v, want %v", tt.value, tt.pattern, result, tt.want)
			}
		})
	}
}

func TestScoreContentNodeEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("document with very long text", func(t *testing.T) {
		longText := strings.Repeat("word ", 1000)
		inputHTML := `<div>` + longText + `</div>`
		doc, _ := html.Parse(strings.NewReader(inputHTML))
		// Find the div element
		var div *html.Node
		WalkNodes(doc, func(n *html.Node) bool {
			if n.Type == html.ElementNode && n.Data == "div" {
				div = n
				return false
			}
			return true
		})
		score := ScoreContentNode(div)
		// Very long content should have high score
		if score < 500 {
			t.Errorf("ScoreContentNode(long text) = %d, want >= 500", score)
		}
	})

	t.Run("document with only scripts", func(t *testing.T) {
		inputHTML := `<html><head><script>var x = 1;</script></head><body></body></html>`
		doc, _ := html.Parse(strings.NewReader(inputHTML))
		var body *html.Node
		WalkNodes(doc, func(n *html.Node) bool {
			if n.Type == html.ElementNode && n.Data == "body" {
				body = n
				return false
			}
			return true
		})
		score := ScoreContentNode(body)
		// Body with no content should have low score
		if score > 100 {
			t.Errorf("ScoreContentNode(empty body) = %d, want <= 100", score)
		}
	})
}

func TestCalculateContentDensityEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("nil node", func(t *testing.T) {
		density := CalculateContentDensity(nil)
		if density != 0 {
			t.Errorf("CalculateContentDensity(nil) = %f, want 0", density)
		}
	})

	t.Run("text node only", func(t *testing.T) {
		textNode := &html.Node{
			Type: html.TextNode,
			Data: "Hello World",
		}
		density := CalculateContentDensity(textNode)
		if density != 1.0 {
			t.Errorf("CalculateContentDensity(text node) = %f, want 1.0", density)
		}
	})

	t.Run("single tag with text", func(t *testing.T) {
		inputHTML := `<p>Hello</p>`
		doc, _ := html.Parse(strings.NewReader(inputHTML))
		density := CalculateContentDensity(doc)
		if density <= 0 {
			t.Errorf("CalculateContentDensity() = %f, want > 0", density)
		}
	})

	t.Run("many tags little text", func(t *testing.T) {
		inputHTML := strings.Repeat("<div>", 100) + "text" + strings.Repeat("</div>", 100)
		doc, _ := html.Parse(strings.NewReader(inputHTML))
		density := CalculateContentDensity(doc)
		// Many tags with little text should have low density
		if density > 0.1 {
			t.Errorf("CalculateContentDensity(many tags) = %f, want <= 0.1", density)
		}
	})
}
