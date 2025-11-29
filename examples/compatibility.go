package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/cybergodev/html"
)

func main() {
	fmt.Println("=== 100% Compatible with golang.org/x/net/html ===")
	fmt.Println()

	// Example 1: Standard HTML Parsing (Drop-in Replacement)
	fmt.Println("1. Standard HTML Parsing:")
	doc, err := html.Parse(strings.NewReader("<html><body><h1>Hello</h1></body></html>"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("   Parsed document type: %v\n", doc.Type)

	// Example 2: Rendering HTML
	fmt.Println("\n2. Rendering HTML:")
	var buf bytes.Buffer
	html.Render(&buf, doc)
	fmt.Printf("   Rendered: %s\n", buf.String()[:50]+"...")

	// Example 3: Escaping/Unescaping
	fmt.Println("\n3. Escaping and Unescaping:")
	escaped := html.EscapeString("<script>alert('xss')</script>")
	fmt.Printf("   Escaped: %s\n", escaped)
	unescaped := html.UnescapeString("&lt;html&gt; &copy; 2024")
	fmt.Printf("   Unescaped: %s\n", unescaped)

	// Example 4: Tokenizer
	fmt.Println("\n4. HTML Tokenizer:")
	tokenizer := html.NewTokenizer(strings.NewReader("<p>Test</p>"))
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		fmt.Printf("   Token: %v\n", tt)
	}

	// Example 5: Node Creation
	fmt.Println("\n5. Node Creation:")
	node := &html.Node{
		Type: html.ElementNode,
		Data: "div",
		Attr: []html.Attribute{
			{Key: "class", Val: "container"},
			{Key: "id", Val: "main"},
		},
	}
	fmt.Printf("   Created node: <%s class=%q id=%q>\n", node.Data, node.Attr[0].Val, node.Attr[1].Val)

	// Example 6: Enhanced Content Extraction (Library-Specific)
	fmt.Println("\n6. Enhanced Content Extraction (Bonus Feature):")
	processor := html.NewWithDefaults()
	defer processor.Close()

	htmlContent := `
		<article>
			<h1>Article Title</h1>
			<p>This is the main content of the article.</p>
			<img src="image.jpg" alt="Test Image">
		</article>
	`

	result, err := processor.ExtractWithDefaults(htmlContent)
	if err != nil {
		panic(err)
	}

	fmt.Printf("   Title: %s\n", result.Title)
	fmt.Printf("   Text: %s\n", result.Text)
	fmt.Printf("   Word Count: %d\n", result.WordCount)
	fmt.Printf("   Images: %d\n", len(result.Images))
	fmt.Printf("   Processing Time: %v\n", result.ProcessingTime)

	fmt.Println("\n=== Summary ===")
	fmt.Println("✓ All golang.org/x/net/html APIs available")
	fmt.Println("✓ Drop-in replacement - just change import path")
	fmt.Println("✓ Plus enhanced content extraction features")
}
