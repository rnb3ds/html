package internal

import (
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

// builderPool reuses string builders to reduce allocations.
var builderPool = sync.Pool{
	New: func() any {
		sb := &strings.Builder{}
		sb.Grow(1024) // Pre-allocate reasonable size
		return sb
	},
}

// getStringBuilder acquires a string builder from the pool.
func getStringBuilder() *strings.Builder {
	return builderPool.Get().(*strings.Builder)
}

// putStringBuilder returns a string builder to the pool after resetting it.
func putStringBuilder(sb *strings.Builder) {
	sb.Reset()
	builderPool.Put(sb)
}

// trackedBuilder wraps strings.Builder to track last character without allocation.
type trackedBuilder struct {
	*strings.Builder
	lastChar byte
	lastLen  int
}

// newTrackedBuilder creates a new tracked builder.
func newTrackedBuilder(sb *strings.Builder) *trackedBuilder {
	return &trackedBuilder{
		Builder:  sb,
		lastChar: 0,
		lastLen:  0,
	}
}

// WriteByte implements io.ByteWriter with character tracking.
func (tb *trackedBuilder) WriteByte(c byte) error {
	tb.lastChar = c
	tb.lastLen = tb.Builder.Len()
	return tb.Builder.WriteByte(c)
}

// WriteString writes string and tracks last character.
func (tb *trackedBuilder) WriteString(s string) (int, error) {
	n, err := tb.Builder.WriteString(s)
	if n > 0 && err == nil {
		tb.lastChar = s[len(s)-1]
		tb.lastLen = tb.Builder.Len()
	}
	return n, err
}

// ensureNewlineTracked adds newline if last character is not newline (using tracked builder).
func ensureNewlineTracked(tb *trackedBuilder) {
	if tb.lastLen > 0 && tb.lastChar != '\n' {
		tb.WriteByte('\n')
	}
}

// ensureSpacingTracked adds spacing character if last character is not space or newline (using tracked builder).
func ensureSpacingTracked(tb *trackedBuilder, char byte) {
	if tb.lastLen > 0 && tb.lastChar != ' ' && tb.lastChar != '\n' {
		tb.WriteByte(char)
	}
}

func ExtractTextWithStructureAndImages(node *html.Node, sb *strings.Builder, depth int, imageCounter *int) {
	if node == nil {
		return
	}
	if node.Type == html.ElementNode && IsNonContentElement(node.Data) {
		return
	}

	// Wrap with tracked builder for better performance
	tb := newTrackedBuilder(sb)
	extractTextWithStructureOptimized(node, tb, depth, imageCounter)
}

func extractTextWithStructureOptimized(node *html.Node, tb *trackedBuilder, depth int, imageCounter *int) {
	if node == nil {
		return
	}
	if node.Type == html.ElementNode && IsNonContentElement(node.Data) {
		return
	}
	if node.Type == html.TextNode {
		if content := strings.TrimSpace(node.Data); content != "" {
			ensureSpacingTracked(tb, ' ')
			tb.WriteString(content)
		}
		return
	}
	if node.Type == html.ElementNode {
		if node.Data == "img" && imageCounter != nil {
			*imageCounter++
			ensureNewlineTracked(tb)
			tb.WriteString("[IMAGE:")
			tb.WriteString(strconv.Itoa(*imageCounter))
			tb.WriteString("]\n")
			return
		}
		if node.Data == "table" {
			extractTableOptimized(node, tb)
			return
		}
		isBlockElement := IsBlockElement(node.Data)
		startLen := tb.Len()
		if isBlockElement && startLen > 0 {
			ensureNewlineTracked(tb)
			startLen = tb.Len()
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			extractTextWithStructureOptimized(child, tb, depth+1, imageCounter)
		}
		hasContent := tb.Len() > startLen
		if isBlockElement && hasContent {
			ensureNewlineTracked(tb)
		}
		if !isBlockElement && hasContent && depth > 0 && node.NextSibling != nil {
			ensureSpacingTracked(tb, ' ')
		}
	} else {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			extractTextWithStructureOptimized(child, tb, depth+1, imageCounter)
		}
	}
}

func extractTableOptimized(table *html.Node, tb *trackedBuilder) {
	ensureNewlineTracked(tb)

	rows := make([][]string, 0, 8)
	var maxCols int
	WalkNodes(table, func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "tr" {
			cells := make([]string, 0, 4)
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode && (child.Data == "td" || child.Data == "th") {
					cellText := strings.TrimSpace(GetTextContent(child))
					if cellText == "" {
						cellText = " "
					}
					cells = append(cells, cellText)
				}
			}
			if len(cells) > 0 {
				rows = append(rows, cells)
				if len(cells) > maxCols {
					maxCols = len(cells)
				}
			}
			return false
		}
		return node.Data != "tr"
	})
	if len(rows) == 0 {
		return
	}

	// Pad rows to same length
	for i := range rows {
		for len(rows[i]) < maxCols {
			rows[i] = append(rows[i], " ")
		}
	}

	// Write table with markdown format
	for i, row := range rows {
		tb.WriteString("| ")
		tb.WriteString(strings.Join(row, " | "))
		tb.WriteString(" |\n")
		if i == 0 {
			tb.WriteByte('|')
			for j := 0; j < maxCols; j++ {
				tb.WriteString(" --- |")
			}
			tb.WriteByte('\n')
		}
	}
	tb.WriteByte('\n')
}

// Backward compatibility: keep old function signature but use optimized implementation
func ensureNewline(sb *strings.Builder) {
	if length := sb.Len(); length > 0 {
		// Fallback: for simple cases we accept the allocation cost
		// This maintains backward compatibility while most code uses tracked builder
		s := sb.String()
		if s[length-1] != '\n' {
			sb.WriteByte('\n')
		}
	}
}

func ensureSpacing(sb *strings.Builder, char byte) {
	if length := sb.Len(); length > 0 {
		s := sb.String()
		lastChar := s[length-1]
		if lastChar != ' ' && lastChar != '\n' {
			sb.WriteByte(char)
		}
	}
}

func extractTable(table *html.Node, sb *strings.Builder) {
	tb := newTrackedBuilder(sb)
	extractTableOptimized(table, tb)
}

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
