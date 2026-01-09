package internal

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

func ExtractTextWithStructure(n *html.Node, sb *strings.Builder, depth int) {
	ExtractTextWithStructureAndImages(n, sb, depth, nil)
}

func ExtractTextWithStructureAndImages(n *html.Node, sb *strings.Builder, depth int, imageCounter *int) {
	if n == nil {
		return
	}
	if n.Type == html.ElementNode && IsNonContentElement(n.Data) {
		return
	}
	if n.Type == html.TextNode {
		if content := strings.TrimSpace(n.Data); content != "" {
			ensureSpacing(sb, ' ')
			sb.WriteString(content)
		}
		return
	}
	if n.Type == html.ElementNode {
		if n.Data == "img" && imageCounter != nil {
			*imageCounter++
			ensureNewline(sb)
			fmt.Fprintf(sb, "[IMAGE:%d]\n", *imageCounter)
			return
		}
		if n.Data == "table" {
			extractTable(n, sb)
			return
		}
		isBlockElement := IsBlockElement(n.Data)
		startLen := sb.Len()
		if isBlockElement && startLen > 0 {
			ensureNewline(sb)
			startLen = sb.Len()
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			ExtractTextWithStructureAndImages(c, sb, depth+1, imageCounter)
		}
		hasContent := sb.Len() > startLen
		if isBlockElement && hasContent {
			ensureNewline(sb)
		}
		if !isBlockElement && hasContent && depth > 0 && n.NextSibling != nil {
			ensureSpacing(sb, ' ')
		}
	} else {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			ExtractTextWithStructureAndImages(c, sb, depth+1, imageCounter)
		}
	}
}

// ensureNewline adds newline if last character is not newline.
func ensureNewline(sb *strings.Builder) {
	if length := sb.Len(); length > 0 {
		s := sb.String()
		if s[length-1] != '\n' {
			sb.WriteByte('\n')
		}
	}
}

// ensureSpacing adds spacing character if last character is not space or newline.
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
	ensureNewline(sb)

	rows := make([][]string, 0, 8)
	var maxCols int
	WalkNodes(table, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "tr" {
			cells := make([]string, 0, 4)
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
					cellText := strings.TrimSpace(GetTextContent(c))
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
		return n.Data != "tr"
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
		sb.WriteString("| ")
		sb.WriteString(strings.Join(row, " | "))
		sb.WriteString(" |\n")
		if i == 0 {
			sb.WriteByte('|')
			for j := 0; j < maxCols; j++ {
				sb.WriteString(" --- |")
			}
			sb.WriteByte('\n')
		}
	}
	sb.WriteByte('\n')
}

func CleanContentNode(n *html.Node) *html.Node {
	if n == nil {
		return nil
	}
	toRemove := make([]*html.Node, 0, 8)
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && ShouldRemoveElement(c) {
				toRemove = append(toRemove, c)
			} else {
				traverse(c)
			}
		}
	}
	traverse(n)
	for _, node := range toRemove {
		if node.Parent != nil {
			node.Parent.RemoveChild(node)
		}
	}
	return n
}
