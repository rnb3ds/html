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
	if node == nil {
		return table.AlignDefault
	}
	return getCellAlign(node)
}

// GetColSpan implements table.CellAccessor.
func (a *htmlCellAccessor) GetColSpan(node *html.Node) int {
	if node == nil {
		return 1
	}
	return getColSpan(node)
}

// GetRowSpan implements table.CellAccessor.
func (a *htmlCellAccessor) GetRowSpan(node *html.Node) int {
	if node == nil {
		return 1
	}
	return getRowSpan(node)
}

// GetWidth implements table.CellAccessor.
func (a *htmlCellAccessor) GetWidth(node *html.Node) string {
	if node == nil {
		return ""
	}
	return getCellWidth(node)
}

// GetTextContent implements table.CellAccessor.
func (a *htmlCellAccessor) GetTextContent(node *html.Node) string {
	if node == nil {
		return ""
	}
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

// containsWord checks if text contains word with proper boundary detection.
// This is used for parsing CSS style attributes to ensure we match complete
// property names (e.g., "text-align:center" not just "align:center").
func containsWord(text, word string) bool {
	return hasWordBoundary(text, word, boundaryCSS)
}

// getCellAlign extracts the alignment from a table cell node.
// It first checks the align attribute, then the style attribute for text-align.
// Optimized to use a single loop through attributes.
func getCellAlign(n *html.Node) table.CellAlignment {
	if n == nil {
		return table.AlignDefault
	}

	var styleAttr string

	// Single pass through attributes - collect style and check align
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
		} else if attrKey == "style" {
			styleAttr = attr.Val
		}
	}

	// Check style attribute for text-align (only if found)
	if styleAttr != "" {
		style := strings.ToLower(styleAttr)
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

	return table.AlignDefault
}

// getColSpan extracts the colspan attribute value from a table cell.
// Returns 1 if no colspan attribute is present or if the value is invalid.
func getColSpan(n *html.Node) int {
	if n == nil {
		return 1
	}
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
	if n == nil {
		return 1
	}
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
	if n == nil {
		return ""
	}
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
