// Package html provides HTML content extraction with automatic encoding detection.
// This file contains type aliases for commonly used types from golang.org/x/net/html.
package html

import (
	htmlstd "html"

	stdxhtml "golang.org/x/net/html"
)

// Type aliases for commonly used types from golang.org/x/net/html.
// These aliases provide convenient access to the underlying HTML types without
// requiring users to import golang.org/x/net/html directly.
type (
	Node        = stdxhtml.Node
	NodeType    = stdxhtml.NodeType
	Token       = stdxhtml.Token
	Attribute   = stdxhtml.Attribute
	Tokenizer   = stdxhtml.Tokenizer
	ParseOption = stdxhtml.ParseOption
)

// Node type constants from golang.org/x/net/html.
const (
	ErrorNode    = stdxhtml.ErrorNode
	TextNode     = stdxhtml.TextNode
	DocumentNode = stdxhtml.DocumentNode
	ElementNode  = stdxhtml.ElementNode
	CommentNode  = stdxhtml.CommentNode
	DoctypeNode  = stdxhtml.DoctypeNode
	RawNode      = stdxhtml.RawNode
)

// Token type constants from golang.org/x/net/html.
const (
	ErrorToken          = stdxhtml.ErrorToken
	TextToken           = stdxhtml.TextToken
	StartTagToken       = stdxhtml.StartTagToken
	EndTagToken         = stdxhtml.EndTagToken
	SelfClosingTagToken = stdxhtml.SelfClosingTagToken
	CommentToken        = stdxhtml.CommentToken
	DoctypeToken        = stdxhtml.DoctypeToken
)

// Function aliases from golang.org/x/net/html and html packages.
var (
	ErrBufferExceeded    = stdxhtml.ErrBufferExceeded
	Parse                = stdxhtml.Parse
	ParseFragment        = stdxhtml.ParseFragment
	Render               = stdxhtml.Render
	EscapeString         = htmlstd.EscapeString
	UnescapeString       = htmlstd.UnescapeString
	NewTokenizer         = stdxhtml.NewTokenizer
	NewTokenizerFragment = stdxhtml.NewTokenizerFragment
)

// ============================================================================
// ContentNode Interface (for Scorer abstraction)
// ============================================================================

// NodeAttr represents a single HTML node attribute.
type NodeAttr struct {
	Key   string
	Value string
}

// ContentNode provides an abstraction over HTML node structure
// for content scoring purposes. This interface hides the internal
// golang.org/x/net/html dependency from public API consumers,
// allowing custom Scorers to be implemented without importing
// the internal HTML parser package.
//
// The interface provides read-only access to node properties
// needed for content quality assessment and filtering.
type ContentNode interface {
	// Type returns the node type as a string.
	// Common values: "element", "text", "comment", "document", "doctype"
	Type() string

	// Data returns the element tag name for element nodes (e.g., "div", "p"),
	// or the text content for text nodes.
	Data() string

	// AttrValue returns the value of the attribute with the given key,
	// or an empty string if the attribute does not exist.
	AttrValue(key string) string

	// Attrs returns all attributes of the node.
	Attrs() []NodeAttr

	// FirstChild returns the first child node, or nil if none.
	FirstChild() ContentNode

	// NextSibling returns the next sibling node, or nil if none.
	NextSibling() ContentNode

	// Parent returns the parent node, or nil if this is the root.
	Parent() ContentNode
}
