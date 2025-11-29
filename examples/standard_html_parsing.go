package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cybergodev/html"
)

// StandardHTMLParsing demonstrates 100% compatibility with golang.org/x/net/html
// All standard APIs work identically - just change the import path.
func main() {
	fmt.Println("=== Standard HTML Parsing (100% Compatible) ===\n ")

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
	fmt.Print("   ")
	html.Render(os.Stdout, doc)
	fmt.Println("\n")

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

	fmt.Println("\n✓ All standard golang.org/x/net/html APIs work identically!")
	fmt.Println("✓ This library is a 100% compatible drop-in replacement")
}
