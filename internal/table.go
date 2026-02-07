package internal

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// Table extraction types and constants

// Cell alignment type for table extraction
type cellAlign int

const (
	alignLeft cellAlign = iota
	alignCenter
	alignRight
	alignJustify
	alignDefault
)

// Cell data with metadata for table extraction
type cellData struct {
	text            string
	align           cellAlign
	colspan         int
	rowspan         int
	isHeader        bool
	width           string // Cell width (e.g., "100px", "1.0%", "auto")
	isExpanded      bool   // True if this cell was created from colspan expansion
	originalColspan int    // Original colspan value before expansion (for HTML output)
}

// containsWord checks if text contains word with proper boundary detection.
// This is used for parsing CSS style attributes to ensure we match complete
// property names (e.g., "text-align:center" not just "align:center").
func containsWord(text, word string) bool {
	return hasWordBoundary(text, word, boundaryCSS)
}

// getCellAlign extracts the alignment from a table cell node.
// It first checks the align attribute, then the style attribute for text-align.
func getCellAlign(n *html.Node) cellAlign {
	// First check align attribute (takes precedence)
	for _, attr := range n.Attr {
		attrKey := strings.ToLower(attr.Key)
		if attrKey == "align" {
			alignVal := strings.ToLower(strings.TrimSpace(attr.Val))
			switch alignVal {
			case "left":
				return alignLeft
			case "center":
				return alignCenter
			case "right":
				return alignRight
			case "justify":
				return alignJustify
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
				return alignJustify
			}
			if containsWord(normalizedStyle, "text-align:right") ||
				containsWord(normalizedStyle, "text-align: right") {
				return alignRight
			}
			if containsWord(normalizedStyle, "text-align:center") ||
				containsWord(normalizedStyle, "text-align: center") {
				return alignCenter
			}
			if containsWord(normalizedStyle, "text-align:left") ||
				containsWord(normalizedStyle, "text-align: left") {
				return alignLeft
			}
		}
	}

	return alignDefault
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
