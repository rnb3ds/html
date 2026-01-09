//go:build examples

package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/cybergodev/html"
)

// Compatibility demonstrates 100% compatibility with golang.org/x/net/html.
// All standard APIs work identically - just change the import path.
func main() {
	fmt.Println("=== 100% Compatible with golang.org/x/net/html ===\n ")

	// Example 1: Parse HTML documents
	fmt.Println("1. Parse HTML document:")
	htmlContent := "<html><body><h1>Hello</h1><p>World</p></body></html>"
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		panic(err)
	}
	fmt.Printf("   Parsed document type: %v\n", doc.Type)
	fmt.Printf("   First child: %v\n\n", doc.FirstChild.Data)

	// Example 2: Render HTML
	fmt.Println("2. Render HTML:")
	var buf bytes.Buffer
	html.Render(&buf, doc)
	fmt.Printf("   Rendered: %s\n\n", buf.String()[:50]+"...")

	// Example 3: Escape/Unescape HTML entities
	fmt.Println("3. Escape HTML:")
	dangerous := "<script>alert('xss')</script>"
	escaped := html.EscapeString(dangerous)
	fmt.Printf("   Original: %s\n", dangerous)
	fmt.Printf("   Escaped: %s\n\n", escaped)

	fmt.Println("4. Unescape HTML:")
	encoded := "&lt;html&gt; &copy; 2024 &amp; &quot;quotes&quot;"
	unescaped := html.UnescapeString(encoded)
	fmt.Printf("   Encoded: %s\n", encoded)
	fmt.Printf("   Unescaped: %s\n\n", unescaped)

	// Example 5: Tokenize HTML
	fmt.Println("5. Tokenize HTML:")
	tokenHTML := "<p>Hello <strong>World</strong></p>"
	tokenizer := html.NewTokenizer(strings.NewReader(tokenHTML))
	fmt.Printf("   Tokenizing: %s\n", tokenHTML)
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		token := tokenizer.Token()
		fmt.Printf("   Token: %v (type: %v)\n", token.Data, token.Type)
	}
	fmt.Println()

	// Example 6: Node creation and manipulation
	fmt.Println("6. Node creation:")
	node := &html.Node{
		Type: html.ElementNode,
		Data: "div",
		Attr: []html.Attribute{
			{Key: "class", Val: "container"},
			{Key: "id", Val: "main"},
		},
	}
	fmt.Printf("   Created node: <%s class=%q id=%q>\n\n", node.Data, node.Attr[0].Val, node.Attr[1].Val)

	// Example 7: Render node to stdout
	fmt.Println("7. Render to stdout:")
	fmt.Print("   ")
	html.Render(os.Stdout, node)
	fmt.Println("\n")

	// Example 8: Enhanced content extraction (bonus feature)
	fmt.Println("8. Enhanced content extraction (bonus feature):")
	processor := html.NewWithDefaults()
	defer processor.Close()

	articleHTML := `
		<article>
			<h1>Article Title</h1>
			<p>This is the main content of the article with detailed information.</p>
			<img src="image.jpg" alt="Test Image">
		</article>
	`

	result, err := processor.ExtractWithDefaults(articleHTML)
	if err != nil {
		panic(err)
	}

	fmt.Printf("   Title: %s\n", result.Title)
	fmt.Printf("   Text: %s\n", result.Text)
	fmt.Printf("   Word Count: %d\n", result.WordCount)
	fmt.Printf("   Reading Time: %v\n", result.ReadingTime)
	fmt.Printf("   Images: %d\n", len(result.Images))
	fmt.Printf("   Processing Time: %v\n", result.ProcessingTime)

	fmt.Println("\n=== Summary ===")
	fmt.Println("✓ All golang.org/x/net/html APIs available")
	fmt.Println("✓ Drop-in replacement - just change import path")
	fmt.Println("✓ Plus enhanced content extraction features")
	fmt.Println("✓ Zero breaking changes")
}
