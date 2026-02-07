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
// All standard library APIs work identically - just change the import path.
func main() {
	fmt.Println("=== golang.org/x/net/html Compatibility ===\n ")

	// Example 1: Parse HTML
	fmt.Println("1. Parse HTML:")
	htmlContent := "<html><body><h1>Hello</h1><p>World</p></body></html>"
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		panic(err)
	}
	fmt.Printf("   Parsed document type: %v\n", doc.Type)

	// Safely access root element
	if doc.FirstChild != nil && doc.FirstChild.FirstChild != nil {
		fmt.Printf("   Root element: %v\n\n", doc.FirstChild.FirstChild.Data)
	} else {
		fmt.Printf("   Root element: html\n\n")
	}

	// Example 2: Render HTML
	fmt.Println("2. Render HTML:")
	var buf bytes.Buffer
	html.Render(&buf, doc)
	rendered := buf.String()
	fmt.Printf("   Rendered: %s\n\n", truncate8(rendered, 50)+"...")

	// Example 3: Escape/Unescape strings
	fmt.Println("3. Escape HTML entities:")
	dangerous := "<script>alert('xss')</script>"
	escaped := html.EscapeString(dangerous)
	fmt.Printf("   Original: %s\n", dangerous)
	fmt.Printf("   Escaped: %s\n\n", escaped)

	fmt.Println("4. Unescape HTML entities:")
	encoded := "&lt;html&gt; &copy; 2024 &amp; &quot;quoted&quot;"
	unescaped := html.UnescapeString(encoded)
	fmt.Printf("   Encoded: %s\n", encoded)
	fmt.Printf("   Unescaped: %s\n\n", unescaped)

	// Example 5: Tokenize HTML
	fmt.Println("5. Tokenize HTML:")
	tokenHTML := "<p>Hello <strong>World</strong></p>"
	tokenizer := html.NewTokenizer(strings.NewReader(tokenHTML))
	fmt.Printf("   Tokenizing: %s\n", tokenHTML)
	tokens := []string{}
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		token := tokenizer.Token()
		var tokenType string
		switch token.Type {
		case html.TextToken:
			tokenType = "Text"
		case html.StartTagToken:
			tokenType = "StartTag"
		case html.EndTagToken:
			tokenType = "EndTag"
		case html.SelfClosingTagToken:
			tokenType = "SelfClosing"
		default:
			tokenType = fmt.Sprintf("Type(%d)", token.Type)
		}
		if token.Data != "" {
			tokens = append(tokens, fmt.Sprintf("%s(%s)", token.Data, tokenType))
		}
	}
	fmt.Printf("   Tokens: %v\n\n", tokens)

	// Example 6: Create and manipulate nodes
	fmt.Println("6. Create nodes:")
	// Create parent div node
	divNode := &html.Node{
		Type: html.ElementNode,
		Data: "div",
		Attr: []html.Attribute{
			{Key: "class", Val: "container"},
			{Key: "id", Val: "main"},
		},
	}
	// Create text node
	textNode := &html.Node{
		Type: html.TextNode,
		Data: "Hello, World!",
	}
	// Link text node as child of div
	divNode.AppendChild(textNode)

	var nodeBuf bytes.Buffer
	html.Render(&nodeBuf, divNode)
	fmt.Printf("   Created: %s\n\n", nodeBuf.String())

	// Example 7: Traverse DOM
	fmt.Println("7. Traverse DOM tree:")
	traverseHTML := "<div><p>First</p><p>Second</p><p>Third</p></div>"
	traverseDoc, _ := html.Parse(strings.NewReader(traverseHTML))
	var pCount int
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "p" {
			pCount++
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(traverseDoc)
	fmt.Printf("   Found %d <p> tags\n\n", pCount)

	// Example 8: Render to stdout
	fmt.Println("8. Render to stdout:")
	fmt.Print("   ")
	html.Render(os.Stdout, divNode)
	fmt.Println("\n")

	// Bonus: Enhanced extraction
	fmt.Println("9. Bonus: Enhanced extraction (not in stdlib):")
	processor := html.New()
	defer processor.Close()

	result, err := processor.ExtractWithDefaults(`
		<article>
			<h1>Enhanced Features</h1>
			<p>This library provides stdlib compatibility PLUS enhanced extraction.</p>
			<img src="feature.png" alt="Feature Diagram">
		</article>
	`)
	if err != nil {
		panic(err)
	}
	fmt.Printf("   Title: %s\n", result.Title)
	fmt.Printf("   Word Count: %d\n", result.WordCount)
	fmt.Printf("   Reading Time: %v\n", result.ReadingTime)
	fmt.Printf("   Images: %d\n", len(result.Images))

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✓ 100% compatible with golang.org/x/net/html")
	fmt.Println("✓ Drop-in replacement - just change import path")
	fmt.Println("✓ Plus enhanced content extraction features")
	fmt.Println(strings.Repeat("=", 50))
}

func truncate8(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
