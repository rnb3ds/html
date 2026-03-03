// Package internal provides implementation details for the cybergodev/html library.
// It contains content extraction, table processing, and text manipulation functionality
// that is not part of the public API.
package internal

import (
	"strconv"
	"strings"

	"github.com/cybergodev/html/internal/table"
	"golang.org/x/net/html"
)

// ExtractTextWithStructureAndImages extracts text content from an HTML node tree
// while preserving document structure (headings, paragraphs, lists, tables).
func ExtractTextWithStructureAndImages(node *html.Node, sb *strings.Builder, imageCounter *int, linkCounter *int, tableFormat string) {
	if node == nil {
		return
	}
	if node.Type == html.ElementNode && IsNonContentElement(node.Data) {
		return
	}

	tb := table.NewTrackedBuilder(sb)
	extractTextWithStructure(node, tb, imageCounter, linkCounter, tableFormat, nil, 0)
}

func extractTextWithStructure(node *html.Node, tb *table.TrackedBuilder, imageCounter *int, linkCounter *int, tableFormat string, parentBlock *html.Node, depth int) {
	if node == nil {
		return
	}
	if node.Type == html.ElementNode && IsNonContentElement(node.Data) {
		return
	}
	if node.Type == html.TextNode {
		textData := node.Data
		// Replace non-breaking spaces (U+00A0) with regular spaces
		// The HTML parser converts &nbsp;, &#160;, &#xa0; to U+00A0 automatically
		textData = normalizeNonBreakingSpaces(textData)
		textData = ReplaceHTMLEntities(textData)
		// Replace internal newlines with spaces for multi-line text in HTML
		// Combined into a single pass for better performance
		textData = normalizeLineBreaks(textData)

		// Check if we're inside an inline/namespace element
		isInsideInline := false
		if parentBlock != nil && parentBlock.Type == html.ElementNode {
			isInsideInline = IsInlineElement(parentBlock.Data) || IsNamespaceTag(parentBlock.Data)
		}

		if isInsideInline {
			// Inside inline elements, handle trailing space based on next sibling
			hasTrailingSpace := strings.HasSuffix(textData, " ") || strings.HasSuffix(textData, "\t")
			content := strings.TrimSpace(textData)
			if content != "" {
				tb.WriteString(content)
				// Preserve trailing space UNLESS next sibling is a namespace tag
				// Namespace tags (ix:*, xbrl:*, etc.) should be concatenated without spaces
				if hasTrailingSpace {
					shouldPreserveSpace := true
					if node.NextSibling != nil && node.NextSibling.Type == html.ElementNode {
						// Check if next sibling is a namespace tag
						nextTag := node.NextSibling.Data
						if IsNamespaceTag(nextTag) || IsKnownInlineNamespacePrefix(GetNamespacePrefix(nextTag)) {
							shouldPreserveSpace = false
						}
					}
					if shouldPreserveSpace {
						tb.WriteByte(' ')
					}
				}
			}
		} else {
			// For regular text nodes, check for trailing space and preserve it
			hasTrailingSpace := strings.HasSuffix(textData, " ") || strings.HasSuffix(textData, "\t")
			content := strings.TrimSpace(textData)
			if content != "" {
				table.EnsureSpacing(tb, ' ')
				tb.WriteString(content)
				// Preserve trailing space from original HTML
				if hasTrailingSpace {
					tb.WriteByte(' ')
				}
			}
		}
		return
	}
	if node.Type == html.ElementNode {
		if node.Data == "img" && imageCounter != nil {
			*imageCounter++
			table.EnsureNewline(tb)
			tb.WriteString("[IMAGE:")
			tb.WriteString(strconv.Itoa(*imageCounter))
			tb.WriteString("]\n")
			return
		}
		if node.Data == "a" && linkCounter != nil {
			*linkCounter++
			tb.WriteString("[LINK:")
			tb.WriteString(strconv.Itoa(*linkCounter))
			tb.WriteString("]")
			// Continue processing children for link text
		}
		if node.Data == "br" {
			// BR creates a single line break, not paragraph spacing
			// Only add newline if we have content and don't already have one
			if tb.Builder.Len() > 0 && tb.LastChar != '\n' {
				tb.WriteByte('\n')
			}
			return
		}
		if node.Data == "table" {
			// Use the table processor for table extraction
			TableProcessor().Extract(node, tb, tableFormat)
			return
		}
		// Check if this is a paragraph-level block element that needs double newlines
		// Elements like li, br, hr, tr, td, th should not add extra spacing
		isParagraphBlock := IsParagraphLevelBlockElement(node.Data)

		// Structure-aware: for unknown tags, dynamically determine if they should be treated as block elements
		isBlockElement := IsBlockElement(node.Data)
		if !isBlockElement && !isParagraphBlock {
			isBlockElement = ShouldTreatAsBlockElement(node)
			// If dynamically determined to be a block, also treat as paragraph block
			if isBlockElement {
				isParagraphBlock = true
			}
		}

		startLen := tb.Builder.Len()
		if isBlockElement && startLen > 0 {
			table.EnsureNewline(tb)
			// Add Markdown list prefix based on padding-left level
			paddingLeft := extractPaddingLeft(node)
			if paddingLeft > 0 {
				listPrefix := getListPrefix(paddingLeft)
				if listPrefix != "" {
					tb.WriteString(listPrefix)
				}
			}
			startLen = tb.Builder.Len()
		} else if isBlockElement && startLen == 0 {
			// First element - add list prefix if it has padding-left
			paddingLeft := extractPaddingLeft(node)
			if paddingLeft > 0 {
				listPrefix := getListPrefix(paddingLeft)
				if listPrefix != "" {
					tb.WriteString(listPrefix)
				}
				startLen = tb.Builder.Len()
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			extractTextWithStructure(child, tb, imageCounter, linkCounter, tableFormat, node, depth+1)
		}
		// Add closing link tag after processing children
		if node.Data == "a" && linkCounter != nil {
			tb.WriteString("[/LINK]")
		}
		hasContent := tb.Builder.Len() > startLen
		if isBlockElement && hasContent {
			table.EnsureNewline(tb)
			// Add an extra newline for paragraph-level blocks to create paragraph spacing in Markdown
			if isParagraphBlock && tb.LastChar == '\n' {
				tb.WriteByte('\n')
			}
		}
		// Add spacing for non-root inline elements (depth > 0)
		// This ensures proper spacing between inline elements at the same level
		if !isBlockElement && hasContent && node.NextSibling != nil && depth > 0 {
			table.EnsureSpacing(tb, ' ')
		}
	} else {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			extractTextWithStructure(child, tb, imageCounter, linkCounter, tableFormat, parentBlock, depth+1)
		}
	}
}

// CleanContentNode removes non-content elements from the node tree.
func CleanContentNode(node *html.Node) *html.Node {
	if node == nil {
		return nil
	}
	toRemove := make([]*html.Node, 0, 8)
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == html.ElementNode && ShouldRemoveElement(child) {
				toRemove = append(toRemove, child)
			} else {
				traverse(child)
			}
		}
	}
	traverse(node)
	for _, n := range toRemove {
		if n.Parent != nil {
			n.Parent.RemoveChild(n)
		}
	}
	return node
}
