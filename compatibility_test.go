package html_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cybergodev/html"
	stdhtml "golang.org/x/net/html"
)

// TestNodeTypeCompatibility verifies all node types match golang.org/x/net/html
func TestNodeTypeCompatibility(t *testing.T) {
	tests := []struct {
		name    string
		ourType html.NodeType
		stdType stdhtml.NodeType
	}{
		{"ErrorNode", html.ErrorNode, stdhtml.ErrorNode},
		{"TextNode", html.TextNode, stdhtml.TextNode},
		{"DocumentNode", html.DocumentNode, stdhtml.DocumentNode},
		{"ElementNode", html.ElementNode, stdhtml.ElementNode},
		{"CommentNode", html.CommentNode, stdhtml.CommentNode},
		{"DoctypeNode", html.DoctypeNode, stdhtml.DoctypeNode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ourType != tt.stdType {
				t.Errorf("%s mismatch: got %v, want %v", tt.name, tt.ourType, tt.stdType)
			}
		})
	}
}

// TestTokenTypeCompatibility verifies all token types match golang.org/x/net/html
func TestTokenTypeCompatibility(t *testing.T) {
	tests := []struct {
		name    string
		ourType html.TokenType
		stdType stdhtml.TokenType
	}{
		{"ErrorToken", html.ErrorToken, stdhtml.ErrorToken},
		{"TextToken", html.TextToken, stdhtml.TextToken},
		{"StartTagToken", html.StartTagToken, stdhtml.StartTagToken},
		{"EndTagToken", html.EndTagToken, stdhtml.EndTagToken},
		{"SelfClosingTagToken", html.SelfClosingTagToken, stdhtml.SelfClosingTagToken},
		{"CommentToken", html.CommentToken, stdhtml.CommentToken},
		{"DoctypeToken", html.DoctypeToken, stdhtml.DoctypeToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ourType != tt.stdType {
				t.Errorf("%s mismatch: got %v, want %v", tt.name, tt.ourType, tt.stdType)
			}
		})
	}
}

// TestParseCompatibility verifies Parse function compatibility
func TestParseCompatibility(t *testing.T) {
	htmlContent := "<html><head><title>Test</title></head><body><p>Content</p></body></html>"

	ourDoc, ourErr := html.Parse(strings.NewReader(htmlContent))
	stdDoc, stdErr := stdhtml.Parse(strings.NewReader(htmlContent))

	if (ourErr == nil) != (stdErr == nil) {
		t.Errorf("Parse error mismatch: our=%v, std=%v", ourErr, stdErr)
	}

	if ourDoc.Type != stdDoc.Type {
		t.Errorf("Parse document type mismatch: got %v, want %v", ourDoc.Type, stdDoc.Type)
	}
}

// TestRenderCompatibility verifies Render function compatibility
func TestRenderCompatibility(t *testing.T) {
	htmlContent := "<html><body><p>Test</p></body></html>"

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var ourBuf, stdBuf bytes.Buffer
	ourErr := html.Render(&ourBuf, doc)
	stdErr := stdhtml.Render(&stdBuf, doc)

	if (ourErr == nil) != (stdErr == nil) {
		t.Errorf("Render error mismatch: our=%v, std=%v", ourErr, stdErr)
	}

	if ourBuf.String() != stdBuf.String() {
		t.Errorf("Render output mismatch:\nour=%q\nstd=%q", ourBuf.String(), stdBuf.String())
	}
}

// TestEscapeStringCompatibility verifies EscapeString function compatibility
func TestEscapeStringCompatibility(t *testing.T) {
	tests := []string{
		"<html>",
		"a&b",
		`"quoted"`,
		"'single'",
		"line1\rline2",
		"<script>alert('xss')</script>",
		"&amp;&lt;&gt;",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			our := html.EscapeString(input)
			std := stdhtml.EscapeString(input)
			if our != std {
				t.Errorf("EscapeString mismatch for %q:\nour=%q\nstd=%q", input, our, std)
			}
		})
	}
}

// TestUnescapeStringCompatibility verifies UnescapeString function compatibility
func TestUnescapeStringCompatibility(t *testing.T) {
	tests := []string{
		"&lt;html&gt;",
		"&amp;",
		"&aacute;",
		"&#225;",
		"&#xE1;",
		"&nbsp;",
		"&copy;",
		"&euro;",
		"&mdash;",
		"&unknown;",
		"&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			our := html.UnescapeString(input)
			std := stdhtml.UnescapeString(input)
			if our != std {
				t.Errorf("UnescapeString mismatch for %q:\nour=%q\nstd=%q", input, our, std)
			}
		})
	}
}

// TestTokenizerCompatibility verifies Tokenizer compatibility
func TestTokenizerCompatibility(t *testing.T) {
	htmlContent := "<p>Test</p><div>Content</div>"

	ourTokenizer := html.NewTokenizer(strings.NewReader(htmlContent))
	stdTokenizer := stdhtml.NewTokenizer(strings.NewReader(htmlContent))

	for {
		ourTT := ourTokenizer.Next()
		stdTT := stdTokenizer.Next()

		if ourTT != stdTT {
			t.Errorf("Tokenizer token type mismatch: got %v, want %v", ourTT, stdTT)
			break
		}

		if ourTT == html.ErrorToken {
			break
		}
	}
}

// TestNodeStructureCompatibility verifies Node structure compatibility
func TestNodeStructureCompatibility(t *testing.T) {
	node := &html.Node{
		Type: html.ElementNode,
		Data: "div",
		Attr: []html.Attribute{
			{Key: "class", Val: "test"},
		},
	}

	if node.Type != html.ElementNode {
		t.Errorf("Node type mismatch")
	}
	if node.Data != "div" {
		t.Errorf("Node data mismatch")
	}
	if len(node.Attr) != 1 || node.Attr[0].Key != "class" {
		t.Errorf("Node attributes mismatch")
	}
}

// TestParseFragmentCompatibility verifies ParseFragment function compatibility
func TestParseFragmentCompatibility(t *testing.T) {
	htmlContent := "<p>Fragment</p><span>Test</span>"
	context := &html.Node{
		Type: html.ElementNode,
		Data: "body",
	}

	ourNodes, ourErr := html.ParseFragment(strings.NewReader(htmlContent), context)
	stdNodes, stdErr := stdhtml.ParseFragment(strings.NewReader(htmlContent), context)

	if (ourErr == nil) != (stdErr == nil) {
		t.Errorf("ParseFragment error mismatch: our=%v, std=%v", ourErr, stdErr)
	}

	if len(ourNodes) != len(stdNodes) {
		t.Errorf("ParseFragment node count mismatch: got %d, want %d", len(ourNodes), len(stdNodes))
	}
}

// TestDropInReplacement verifies the library can be used as a drop-in replacement
func TestDropInReplacement(t *testing.T) {
	// This test demonstrates that code written for golang.org/x/net/html
	// works identically with this library

	htmlContent := `
		<!DOCTYPE html>
		<html>
		<head><title>Test Page</title></head>
		<body>
			<h1>Hello World</h1>
			<p>This is a test.</p>
		</body>
		</html>
	`

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Render HTML
	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Escape/Unescape strings
	escaped := html.EscapeString("<script>alert('test')</script>")
	if !strings.Contains(escaped, "&lt;") {
		t.Errorf("EscapeString failed")
	}

	unescaped := html.UnescapeString("&lt;html&gt;")
	if unescaped != "<html>" {
		t.Errorf("UnescapeString failed: got %q", unescaped)
	}

	// Tokenize HTML
	tokenizer := html.NewTokenizer(strings.NewReader("<p>Test</p>"))
	tokenCount := 0
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		tokenCount++
	}
	if tokenCount == 0 {
		t.Errorf("Tokenizer failed")
	}
}
