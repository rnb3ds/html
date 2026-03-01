package internal

import (
	"strconv"
	"strings"

	"github.com/cybergodev/html/internal/table"
	"golang.org/x/net/html"
)

// htmlCellAccessor implements table.CellAccessor using the existing helper functions.
type htmlCellAccessor struct{}

// GetAlignment implements table.CellAccessor.
func (a *htmlCellAccessor) GetAlignment(node *html.Node) table.CellAlignment {
	return getCellAlign(node)
}

// GetColSpan implements table.CellAccessor.
func (a *htmlCellAccessor) GetColSpan(node *html.Node) int {
	return getColSpan(node)
}

// GetRowSpan implements table.CellAccessor.
func (a *htmlCellAccessor) GetRowSpan(node *html.Node) int {
	return getRowSpan(node)
}

// GetWidth implements table.CellAccessor.
func (a *htmlCellAccessor) GetWidth(node *html.Node) string {
	return getCellWidth(node)
}

// GetTextContent implements table.CellAccessor.
func (a *htmlCellAccessor) GetTextContent(node *html.Node) string {
	return GetTextContent(node)
}

// htmlNodeWalker implements table.NodeWalker using the existing WalkNodes function.
type htmlNodeWalker struct{}

// Walk implements table.NodeWalker.
func (w *htmlNodeWalker) Walk(node *html.Node, callback func(*html.Node) bool) {
	WalkNodes(node, callback)
}

// defaultTableAccessor is the default accessor instance for table extraction.
var defaultTableAccessor = &htmlCellAccessor{}

// defaultTableWalker is the default walker instance for table extraction.
var defaultTableWalker = &htmlNodeWalker{}

// TableProcessor returns the table processor with default accessor and walker.
func TableProcessor() *table.Processor {
	return table.NewProcessor(defaultTableAccessor, defaultTableWalker)
}

// Type aliases for table package types (for internal use)
type cellAlign = table.CellAlignment

// Constants for cell alignment (for internal use)
const (
	alignLeft    = table.AlignLeft
	alignCenter  = table.AlignCenter
	alignRight   = table.AlignRight
	alignJustify = table.AlignJustify
	alignDefault = table.AlignDefault
)

// containsWord checks if text contains word with proper boundary detection.
// This is used for parsing CSS style attributes to ensure we match complete
// property names (e.g., "text-align:center" not just "align:center").
func containsWord(text, word string) bool {
	return hasWordBoundary(text, word, boundaryCSS)
}

// getCellAlign extracts the alignment from a table cell node.
// It first checks the align attribute, then the style attribute for text-align.
func getCellAlign(n *html.Node) table.CellAlignment {
	// First check align attribute (takes precedence)
	for _, attr := range n.Attr {
		attrKey := strings.ToLower(attr.Key)
		if attrKey == "align" {
			alignVal := strings.ToLower(strings.TrimSpace(attr.Val))
			switch alignVal {
			case "left":
				return table.AlignLeft
			case "center":
				return table.AlignCenter
			case "right":
				return table.AlignRight
			case "justify":
				return table.AlignJustify
			}
		}
	}

	// Then check style attribute for text-align
	for _, attr := range n.Attr {
		if strings.ToLower(attr.Key) == "style" {
			style := strings.ToLower(attr.Val)
			// Normalize spaces: remove extra spaces around colons
			normalizedStyle := strings.ReplaceAll(style, " :", ":")
			normalizedStyle = strings.ReplaceAll(normalizedStyle, ": ", ":")
			// Check for text-align patterns with better boundary detection
			// Order matters: check longer/more specific patterns first
			if containsWord(normalizedStyle, "text-align:justify") ||
				containsWord(normalizedStyle, "text-align: justify") {
				return table.AlignJustify
			}
			if containsWord(normalizedStyle, "text-align:right") ||
				containsWord(normalizedStyle, "text-align: right") {
				return table.AlignRight
			}
			if containsWord(normalizedStyle, "text-align:center") ||
				containsWord(normalizedStyle, "text-align: center") {
				return table.AlignCenter
			}
			if containsWord(normalizedStyle, "text-align:left") ||
				containsWord(normalizedStyle, "text-align: left") {
				return table.AlignLeft
			}
		}
	}

	return table.AlignDefault
}

// getColSpan extracts the colspan attribute value from a table cell.
// Returns 1 if no colspan attribute is present or if the value is invalid.
func getColSpan(n *html.Node) int {
	for _, attr := range n.Attr {
		if strings.ToLower(attr.Key) == "colspan" {
			if val, err := strconv.Atoi(strings.TrimSpace(attr.Val)); err == nil && val > 0 {
				return val
			}
		}
	}
	return 1
}

// getRowSpan extracts the rowspan attribute value from a table cell.
// Returns 1 if no rowspan attribute is present or if the value is invalid.
func getRowSpan(n *html.Node) int {
	for _, attr := range n.Attr {
		if strings.ToLower(attr.Key) == "rowspan" {
			if val, err := strconv.Atoi(strings.TrimSpace(attr.Val)); err == nil && val > 0 {
				return val
			}
		}
	}
	return 1
}

// getCellWidth extracts the width from a table cell node.
// It checks both the width attribute and the style attribute.
func getCellWidth(n *html.Node) string {
	// First check width attribute
	for _, attr := range n.Attr {
		if strings.ToLower(attr.Key) == "width" {
			widthVal := strings.TrimSpace(attr.Val)
			if widthVal != "" && widthVal != "0" {
				return widthVal
			}
		}
	}
	// Then check style attribute
	for _, attr := range n.Attr {
		if strings.ToLower(attr.Key) == "style" {
			style := attr.Val
			widthLower := strings.ToLower(style)
			if idx := strings.Index(widthLower, "width:"); idx >= 0 {
				start := idx + 6
				for start < len(style) && (style[start] == ' ' || style[start] == '\t') {
					start++
				}
				end := start
				for end < len(style) {
					c := style[end]
					if c == ';' || c == '"' || c == '\'' || c == '}' {
						break
					}
					end++
				}
				widthVal := strings.TrimSpace(style[start:end])
				if widthVal != "" && widthVal != "0" && widthVal != "0px" && widthVal != "0%" {
					return widthVal
				}
			}
		}
	}
	return ""
}
