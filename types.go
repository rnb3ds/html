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
