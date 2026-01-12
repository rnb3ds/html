package internal

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

func ExtractTextWithStructureAndImages(node *html.Node, sb *strings.Builder, depth int, imageCounter *int) {
	if node == nil {
		return
	}
	if node.Type == html.ElementNode && IsNonContentElement(node.Data) {
		return
	}
	if node.Type == html.TextNode {
		if content := strings.TrimSpace(node.Data); content != "" {
			ensureSpacing(sb, ' ')
			sb.WriteString(content)
		}
		return
	}
	if node.Type == html.ElementNode {
		if node.Data == "img" && imageCounter != nil {
			*imageCounter++
			ensureNewline(sb)
			fmt.Fprintf(sb, "[IMAGE:%d]\n", *imageCounter)
			return
		}
		if node.Data == "table" {
			extractTable(node, sb)
			return
		}
		isBlockElement := IsBlockElement(node.Data)
		startLen := sb.Len()
		if isBlockElement && startLen > 0 {
			ensureNewline(sb)
			startLen = sb.Len()
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			ExtractTextWithStructureAndImages(child, sb, depth+1, imageCounter)
		}
		hasContent := sb.Len() > startLen
		if isBlockElement && hasContent {
			ensureNewline(sb)
		}
		if !isBlockElement && hasContent && depth > 0 && node.NextSibling != nil {
			ensureSpacing(sb, ' ')
		}
	} else {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			ExtractTextWithStructureAndImages(child, sb, depth+1, imageCounter)
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
