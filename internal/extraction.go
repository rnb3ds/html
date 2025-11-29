package internal

import (
	"fmt"
	"regexp"
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
		content := strings.TrimSpace(n.Data)
		if content != "" {
			if sb.Len() > 0 && !strings.HasSuffix(sb.String(), " ") && !strings.HasSuffix(sb.String(), "\n") {
				sb.WriteByte(' ')
			}
			sb.WriteString(content)
		}
		return
	}

	if n.Type == html.ElementNode {
		if n.Data == "img" && imageCounter != nil {
			*imageCounter++
			if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "\n") {
				sb.WriteByte('\n')
			}
			sb.WriteString(fmt.Sprintf("[IMAGE:%d]\n", *imageCounter))
			return
		}

		if n.Data == "table" {
			extractTable(n, sb)
			return
		}

		isBlockElement := IsBlockElement(n.Data)
		startLen := sb.Len()

		if isBlockElement && startLen > 0 && !strings.HasSuffix(sb.String(), "\n") {
			sb.WriteByte('\n')
			startLen = sb.Len()
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			ExtractTextWithStructureAndImages(c, sb, depth+1, imageCounter)
		}

		hasContent := sb.Len() > startLen

		if isBlockElement && hasContent && !strings.HasSuffix(sb.String(), "\n") {
			sb.WriteByte('\n')
		}

		if !isBlockElement && hasContent && depth > 0 {
			if n.NextSibling != nil && !strings.HasSuffix(sb.String(), " ") && !strings.HasSuffix(sb.String(), "\n") {
				sb.WriteByte(' ')
			}
		}
	} else {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			ExtractTextWithStructureAndImages(c, sb, depth+1, imageCounter)
		}
	}
}

func extractTable(table *html.Node, sb *strings.Builder) {
	if table == nil {
		return
	}

	if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "\n") {
		sb.WriteByte('\n')
	}

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

	for i := range rows {
		for len(rows[i]) < maxCols {
			rows[i] = append(rows[i], " ")
		}
	}

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

func PostProcessText(text string, whitespaceRegex *regexp.Regexp) string {
	return CleanText(text, whitespaceRegex)
}

func CleanContentNode(n *html.Node) *html.Node {
	if n == nil {
		return nil
	}

	var toRemove []*html.Node
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				if ShouldRemoveElement(c) {
					toRemove = append(toRemove, c)
				} else {
					traverse(c)
				}
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
