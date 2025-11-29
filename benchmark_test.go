package html_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cybergodev/html"
)

func BenchmarkExtract(b *testing.B) {
	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `
		<html>
		<head><title>Benchmark Test</title></head>
		<body>
			<article>
				<h1>Article Title</h1>
				<p>This is a paragraph with some content.</p>
				<p>Another paragraph with more text.</p>
				<img src="image.jpg" alt="Test Image">
				<a href="link.html">Test Link</a>
			</article>
		</body>
		</html>
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			b.Fatalf("Extract() failed: %v", err)
		}
	}
}

func BenchmarkExtractWithCache(b *testing.B) {
	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><p>Cached content</p></body></html>`

	// Prime the cache
	p.Extract(htmlContent, html.DefaultExtractConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			b.Fatalf("Extract() failed: %v", err)
		}
	}
}

func BenchmarkExtractLargeDocument(b *testing.B) {
	p := html.NewWithDefaults()
	defer p.Close()

	// Create a large HTML document
	var sb strings.Builder
	sb.WriteString("<html><body><article>")
	for i := 0; i < 100; i++ {
		sb.WriteString(fmt.Sprintf("<h2>Section %d</h2>", i))
		for j := 0; j < 10; j++ {
			sb.WriteString(fmt.Sprintf("<p>Paragraph %d in section %d with some content.</p>", j, i))
		}
	}
	sb.WriteString("</article></body></html>")
	htmlContent := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			b.Fatalf("Extract() failed: %v", err)
		}
	}
}

func BenchmarkExtractWithImages(b *testing.B) {
	p := html.NewWithDefaults()
	defer p.Close()

	var sb strings.Builder
	sb.WriteString("<html><body>")
	for i := 0; i < 50; i++ {
		sb.WriteString(fmt.Sprintf(`<img src="image%d.jpg" alt="Image %d">`, i, i))
	}
	sb.WriteString("</body></html>")
	htmlContent := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			b.Fatalf("Extract() failed: %v", err)
		}
	}
}

func BenchmarkExtractWithLinks(b *testing.B) {
	p := html.NewWithDefaults()
	defer p.Close()

	var sb strings.Builder
	sb.WriteString("<html><body>")
	for i := 0; i < 50; i++ {
		sb.WriteString(fmt.Sprintf(`<a href="link%d.html">Link %d</a>`, i, i))
	}
	sb.WriteString("</body></html>")
	htmlContent := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			b.Fatalf("Extract() failed: %v", err)
		}
	}
}

func BenchmarkExtractBatch(b *testing.B) {
	p := html.NewWithDefaults()
	defer p.Close()

	htmlContents := make([]string, 10)
	for i := range htmlContents {
		htmlContents[i] = fmt.Sprintf(`<html><body><p>Content %d</p></body></html>`, i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.ExtractBatch(htmlContents, html.DefaultExtractConfig())
		if err != nil {
			b.Fatalf("ExtractBatch() failed: %v", err)
		}
	}
}

func BenchmarkNew(b *testing.B) {
	config := html.DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p, err := html.New(config)
		if err != nil {
			b.Fatalf("New() failed: %v", err)
		}
		p.Close()
	}
}

func BenchmarkParse(b *testing.B) {
	htmlContent := `<html><head><title>Test</title></head><body><p>Content</p></body></html>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := html.Parse(strings.NewReader(htmlContent))
		if err != nil {
			b.Fatalf("Parse() failed: %v", err)
		}
	}
}

func BenchmarkEscapeString(b *testing.B) {
	input := `<script>alert("xss")</script>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = html.EscapeString(input)
	}
}

func BenchmarkUnescapeString(b *testing.B) {
	input := `&lt;html&gt;&amp;&aacute;&#225;&#xE1;`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = html.UnescapeString(input)
	}
}

func BenchmarkConcurrentExtract(b *testing.B) {
	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `<html><body><p>Concurrent test</p></body></html>`

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
			if err != nil {
				b.Fatalf("Extract() failed: %v", err)
			}
		}
	})
}

func BenchmarkArticleExtraction(b *testing.B) {
	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `
		<html>
		<body>
			<nav>Navigation</nav>
			<article>
				<h1>Main Article</h1>
				<p>This is the main content.</p>
				<p>More content here.</p>
				<p>Even more content.</p>
			</article>
			<aside>Sidebar</aside>
			<footer>Footer</footer>
		</body>
		</html>
	`

	config := html.DefaultExtractConfig()
	config.ExtractArticle = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Extract(htmlContent, config)
		if err != nil {
			b.Fatalf("Extract() failed: %v", err)
		}
	}
}

func BenchmarkInlineImageFormatting(b *testing.B) {
	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `
		<html>
		<body>
			<p>Before</p>
			<img src="test1.jpg" alt="Test 1">
			<p>Middle</p>
			<img src="test2.jpg" alt="Test 2">
			<p>After</p>
		</body>
		</html>
	`

	tests := []struct {
		name   string
		format string
	}{
		{"None", "none"},
		{"Placeholder", "placeholder"},
		{"Markdown", "markdown"},
		{"HTML", "html"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			config := html.DefaultExtractConfig()
			config.InlineImageFormat = tt.format

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := p.Extract(htmlContent, config)
				if err != nil {
					b.Fatalf("Extract() failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkMediaExtraction(b *testing.B) {
	p := html.NewWithDefaults()
	defer p.Close()

	htmlContent := `
		<html>
		<body>
			<img src="image.jpg" alt="Image">
			<video src="video.mp4"></video>
			<audio src="audio.mp3"></audio>
			<a href="link.html">Link</a>
		</body>
		</html>
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Extract(htmlContent, html.DefaultExtractConfig())
		if err != nil {
			b.Fatalf("Extract() failed: %v", err)
		}
	}
}
