package internal

import (
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

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

var builderPool = sync.Pool{
	New: func() any {
		sb := &strings.Builder{}
		sb.Grow(1024)
		return sb
	},
}

func getStringBuilder() *strings.Builder {
	return builderPool.Get().(*strings.Builder)
}

func putStringBuilder(sb *strings.Builder) {
	sb.Reset()
	builderPool.Put(sb)
}

type trackedBuilder struct {
	*strings.Builder
	lastChar byte
}

func newTrackedBuilder(sb *strings.Builder) *trackedBuilder {
	return &trackedBuilder{
		Builder:  sb,
		lastChar: 0,
	}
}

func (tb *trackedBuilder) WriteByte(c byte) error {
	tb.lastChar = c
	return tb.Builder.WriteByte(c)
}

func (tb *trackedBuilder) WriteString(s string) (int, error) {
	n, err := tb.Builder.WriteString(s)
	if n > 0 && err == nil {
		tb.lastChar = s[len(s)-1]
	}
	return n, err
}

func ensureNewlineTracked(tb *trackedBuilder) {
	if tb.Builder.Len() > 0 && tb.lastChar != '\n' {
		tb.WriteByte('\n')
	}
}

func ensureSpacingTracked(tb *trackedBuilder, char byte) {
	if tb.Builder.Len() > 0 && tb.lastChar != ' ' && tb.lastChar != '\n' {
		tb.WriteByte(char)
	}
}

func ExtractTextWithStructureAndImages(node *html.Node, sb *strings.Builder, depth int, imageCounter *int, tableFormat string) {
	if node == nil {
		return
	}
	if node.Type == html.ElementNode && IsNonContentElement(node.Data) {
		return
	}

	tb := newTrackedBuilder(sb)
	extractTextWithStructureOptimized(node, tb, depth, imageCounter, tableFormat)
}

func extractTextWithStructureOptimized(node *html.Node, tb *trackedBuilder, depth int, imageCounter *int, tableFormat string) {
	if node == nil {
		return
	}
	if node.Type == html.ElementNode && IsNonContentElement(node.Data) {
		return
	}
	if node.Type == html.TextNode {
		originalData := node.Data
		// Replace internal newlines with spaces to handle multi-line text in HTML
		originalData = strings.ReplaceAll(originalData, "\n", " ")
		originalData = strings.ReplaceAll(originalData, "\r", "")

		if content := strings.TrimSpace(originalData); content != "" {
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
			extractTableTracked(node, tb, tableFormat)
			return
		}
		// Check if this is a paragraph-level block element that needs double newlines
		// Elements like li, br, hr, tr, td, th should not add extra spacing
		isParagraphBlock := isParagraphLevelBlockElement(node.Data)
		isBlockElement := IsBlockElement(node.Data)
		startLen := tb.Len()
		if isBlockElement && startLen > 0 {
			ensureNewlineTracked(tb)
			startLen = tb.Len()
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			extractTextWithStructureOptimized(child, tb, depth+1, imageCounter, tableFormat)
		}
		hasContent := tb.Len() > startLen
		if isBlockElement && hasContent {
			ensureNewlineTracked(tb)
			// Add an extra newline for paragraph-level blocks to create paragraph spacing in Markdown
			if isParagraphBlock && tb.lastChar == '\n' {
				tb.WriteByte('\n')
			}
		}
		if !isBlockElement && hasContent && depth > 0 && node.NextSibling != nil {
			ensureSpacingTracked(tb, ' ')
		}
	} else {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			extractTextWithStructureOptimized(child, tb, depth+1, imageCounter, tableFormat)
		}
	}
}

// isParagraphLevelBlockElement returns true if the element is a block element that should
// be separated by paragraph spacing (double newlines) in the output.
// Elements like li, br, hr, tr, td, th are block elements but don't need paragraph spacing.
func isParagraphLevelBlockElement(tag string) bool {
	switch tag {
	case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6",
		"article", "section", "blockquote", "pre", "ul", "ol", "table":
		return true
	case "li", "br", "hr", "tr", "td", "th":
		return false
	default:
		// For unknown elements, use IsBlockElement as fallback
		return IsBlockElement(tag)
	}
}

func extractTableTracked(table *html.Node, tb *trackedBuilder, tableFormat string) {
	if table == nil {
		return
	}

	ensureNewlineTracked(tb)

	containsWord := func(text, word string) bool {
		idx := strings.Index(text, word)
		if idx == -1 {
			return false
		}
		// Check character before the match
		if idx > 0 {
			before := text[idx-1]
			if before != ';' && before != ':' && before != ' ' && before != '\t' && before != '{' && before != '"' {
				return false
			}
		}
		// Check character after the match
		endIdx := idx + len(word)
		if endIdx < len(text) {
			after := text[endIdx]
			if after != ';' && after != ' ' && after != '\t' && after != '"' && after != '}' && after != ':' {
				return false
			}
		}
		return true
	}

	getCellAlign := func(n *html.Node) cellAlign {
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

	getColSpan := func(n *html.Node) int {
		for _, attr := range n.Attr {
			if strings.ToLower(attr.Key) == "colspan" {
				if val, err := strconv.Atoi(strings.TrimSpace(attr.Val)); err == nil && val > 0 {
					return val
				}
			}
		}
		return 1
	}

	getRowSpan := func(n *html.Node) int {
		for _, attr := range n.Attr {
			if strings.ToLower(attr.Key) == "rowspan" {
				if val, err := strconv.Atoi(strings.TrimSpace(attr.Val)); err == nil && val > 0 {
					return val
				}
			}
		}
		return 1
	}

	getCellWidth := func(n *html.Node) string {
		for _, attr := range n.Attr {
			if strings.ToLower(attr.Key) == "width" {
				widthVal := strings.TrimSpace(attr.Val)
				if widthVal != "" && widthVal != "0" {
					return widthVal
				}
			}
		}
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

	var tableData [][]cellData
	var maxCols int
	colWidths := make([]string, 12)

	WalkNodes(table, func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "tr" {
			logicalCols := 0
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode && (child.Data == "td" || child.Data == "th") {
					colspan := getColSpan(child)
					if colspan < 1 {
						colspan = 1
					}
					logicalCols += colspan
				}
			}
			if logicalCols > maxCols {
				maxCols = logicalCols
			}
		}
		return node.Data != "tr"
	})

	// Second pass: extract cell data
	WalkNodes(table, func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "tr" {
			// First pass: collect raw cell data without expansion
			rawCells := make([]cellData, 0, 4)
			hasWidthDefinitions := true      // Track if all cells have width definitions
			hasNonEmptyContent := false      // Track if row has actual content
			hasComplexChildElements := false // Track if cells have complex child elements

			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode && (child.Data == "td" || child.Data == "th") {
					cellText := strings.TrimSpace(GetTextContent(child))
					if cellText == "" {
						cellText = " "
					}
					cellWidth := getCellWidth(child)
					if cellWidth == "" {
						hasWidthDefinitions = false
					}
					if cellText != " " {
						hasNonEmptyContent = true
					}
					if child.FirstChild != nil {
						for grandchild := child.FirstChild; grandchild != nil; grandchild = grandchild.NextSibling {
							if grandchild.Type == html.ElementNode {
								if grandchild.Data != "br" && grandchild.Data != "hr" {
									elementText := strings.TrimSpace(GetTextContent(grandchild))
									if elementText != "" {
										hasComplexChildElements = true
									}
								}
							}
						}
					}

					colspan := getColSpan(child)
					rowspan := getRowSpan(child)
					if colspan < 1 {
						colspan = 1
					}
					rawCells = append(rawCells, cellData{
						text:            cellText,
						align:           getCellAlign(child),
						colspan:         colspan,
						rowspan:         rowspan,
						isHeader:        child.Data == "th",
						width:           cellWidth,
						originalColspan: colspan,
					})
				}
			}

			isStructureRow := hasWidthDefinitions && !hasNonEmptyContent && !hasComplexChildElements

			var cells []cellData
			if tableFormat == "html" {
				cells = rawCells
			} else {
				cells = make([]cellData, 0, len(rawCells))
				for _, rawCell := range rawCells {
					cells = append(cells, rawCell)
					originalAlign := rawCell.align
					for i := 1; i < rawCell.colspan; i++ {
						cells = append(cells, cellData{
							text:            " ",
							align:           originalAlign,
							colspan:         1,
							rowspan:         rawCell.rowspan,
							isHeader:        rawCell.isHeader,
							width:           "",
							isExpanded:      true, // Mark as expanded cell
							originalColspan: 1,    // Expanded cells have colspan 1
						})
					}
				}
			}

			if isStructureRow {
				widthSourceCells := rawCells
				for i, cell := range widthSourceCells {
					if i < len(colWidths) && cell.width != "" {
						colWidths[i] = cell.width
					}
				}
			}

			if tableFormat == "html" {
				if len(cells) > 0 {
					tableData = append(tableData, cells)
				}
			} else {
				if !isStructureRow && len(cells) > 0 {
					tableData = append(tableData, cells)
				}
			}
			return false
		}
		return node.Data != "tr"
	})

	if len(tableData) == 0 {
		return
	}

	// Output based on format
	switch strings.ToLower(strings.TrimSpace(tableFormat)) {
	case "html":
		extractTableAsHTML(tableData, tb)
	default: // "markdown"
		// Pass column widths extracted from structure rows
		extractTableAsMarkdown(tableData, tb, maxCols, colWidths)
	}

	tb.WriteByte('\n')
}

// extractTableAsMarkdown outputs table in Markdown format with alignment.
// Note: Column widths are included as HTML comments since Markdown doesn't support column widths.
func extractTableAsMarkdown(tableData [][]cellData, tb *trackedBuilder, maxCols int, structureRowWidths []string) {
	for i := range tableData {
		for len(tableData[i]) < maxCols {
			tableData[i] = append(tableData[i], cellData{text: " ", align: alignDefault})
		}
	}

	// Determine column alignments by scanning all rows
	// This handles cases where header row has no alignment but data rows do
	colAligns := make([]string, maxCols)
	colWidths := make([]string, maxCols)
	colAlignsInfo := make([]string, maxCols)

	if len(structureRowWidths) > 0 {
		for i := 0; i < maxCols && i < len(structureRowWidths); i++ {
			if structureRowWidths[i] != "" {
				colWidths[i] = structureRowWidths[i]
			}
		}
	}

	// Count alignment occurrences for each column
	// alignCounts[colIdx][alignType] = count
	type alignCount struct {
		left, center, right, justify, defaultCount int
	}
	alignCounts := make([]alignCount, maxCols)

	for _, row := range tableData {
		for i := 0; i < maxCols && i < len(row); i++ {
			if row[i].width != "" && colWidths[i] == "" {
				colWidths[i] = row[i].width
			}
			if !row[i].isExpanded && row[i].text != " " && row[i].align != alignDefault {
				switch row[i].align {
				case alignLeft:
					alignCounts[i].left++
				case alignCenter:
					alignCounts[i].center++
				case alignRight:
					alignCounts[i].right++
				case alignJustify:
					alignCounts[i].justify++
				default:
					alignCounts[i].defaultCount++
				}
			}
		}
	}

	if len(tableData) > 0 {
		for i := 0; i < maxCols; i++ {
			maxCount := 0
			majorityAlign := alignDefault

			if alignCounts[i].left > maxCount {
				maxCount = alignCounts[i].left
				majorityAlign = alignLeft
			}
			if alignCounts[i].center > maxCount {
				maxCount = alignCounts[i].center
				majorityAlign = alignCenter
			}
			if alignCounts[i].right > maxCount {
				maxCount = alignCounts[i].right
				majorityAlign = alignRight
			}
			if alignCounts[i].justify > maxCount {
				maxCount = alignCounts[i].justify
				majorityAlign = alignJustify
			}

			if maxCount == 0 && len(tableData[0]) > i {
				majorityAlign = tableData[0][i].align
			}

			hasMixedAlignment := alignCounts[i].left > 0 && alignCounts[i].right > 0

			if hasMixedAlignment {
				colAligns[i] = "---"
				colAlignsInfo[i] = "mixed"
			} else {
				switch majorityAlign {
				case alignLeft:
					colAligns[i] = ":---"
					colAlignsInfo[i] = "left"
				case alignCenter:
					colAligns[i] = ":--:"
					colAlignsInfo[i] = "center"
				case alignRight:
					colAligns[i] = "---:"
					colAlignsInfo[i] = "right"
				case alignJustify:
					colAligns[i] = "---"
					colAlignsInfo[i] = "justify"
				default:
					colAligns[i] = "---"
					colAlignsInfo[i] = "default"
				}
			}
		}
	} else {
		for i := range colAligns {
			colAligns[i] = "---"
		}
	}

	colMaxWidths := make([]int, maxCols)
	for _, row := range tableData {
		for j := 0; j < maxCols && j < len(row); j++ {
			textLen := len(row[j].text)
			if textLen > colMaxWidths[j] {
				colMaxWidths[j] = textLen
			}
		}
	}

	for i, row := range tableData {
		tb.WriteString("| ")
		for j := 0; j < maxCols; j++ {
			var cellText string
			if j < len(row) {
				cellText = row[j].text
			} else {
				cellText = " "
			}

			maxWidth := colMaxWidths[j]
			if maxWidth < 3 {
				maxWidth = 3
			}
			textLen := len(cellText)

			switch colAligns[j] {
			case ":---":
				tb.WriteString(cellText)
				tb.WriteString(strings.Repeat(" ", maxWidth-textLen))
			case "---:":
				tb.WriteString(strings.Repeat(" ", maxWidth-textLen))
				tb.WriteString(cellText)
			case ":--:":
				leftPad := (maxWidth - textLen) / 2
				rightPad := maxWidth - textLen - leftPad
				tb.WriteString(strings.Repeat(" ", leftPad))
				tb.WriteString(cellText)
				tb.WriteString(strings.Repeat(" ", rightPad))
			default:
				tb.WriteString(cellText)
				tb.WriteString(strings.Repeat(" ", maxWidth-textLen))
			}

			if j < maxCols-1 {
				tb.WriteString(" | ")
			}
		}
		tb.WriteString(" |\n")

		if i == 0 {
			tb.WriteString("| ")
			tb.WriteString(strings.Join(colAligns, " | "))
			tb.WriteString(" |\n")
		}
	}
}

func extractTableAsHTML(tableData [][]cellData, tb *trackedBuilder) {
	tb.WriteString("<table>\n")

	for i, row := range tableData {
		tb.WriteString("  <tr")
		if i == 0 && len(row) > 0 && row[0].isHeader {
			tb.WriteString(">\n")
		} else {
			tb.WriteString(">\n")
		}

		for _, cell := range row {
			tag := "td"
			if cell.isHeader {
				tag = "th"
			}
			tb.WriteString("    <" + tag)

			var styleParts []string
			switch cell.align {
			case alignLeft:
				styleParts = append(styleParts, "text-align:left")
			case alignCenter:
				styleParts = append(styleParts, "text-align:center")
			case alignRight:
				styleParts = append(styleParts, "text-align:right")
			case alignJustify:
				styleParts = append(styleParts, "text-align:justify")
			}
			if cell.width != "" && !cell.isExpanded {
				styleParts = append(styleParts, "width:"+cell.width)
			}
			if len(styleParts) > 0 {
				tb.WriteString(` style="`)
				for i, part := range styleParts {
					if i > 0 {
						tb.WriteString(";")
					}
					tb.WriteString(part)
				}
				tb.WriteString(`"`)
			}

			if cell.originalColspan > 1 && !cell.isExpanded {
				tb.WriteString(` colspan="`)
				tb.WriteString(strconv.Itoa(cell.originalColspan))
				tb.WriteString(`"`)
			}

			if cell.rowspan > 1 {
				tb.WriteString(` rowspan="`)
				tb.WriteString(strconv.Itoa(cell.rowspan))
				tb.WriteString(`"`)
			}

			tb.WriteString(">")
			tb.WriteString(cell.text)
			tb.WriteString("</" + tag + ">\n")
		}

		tb.WriteString("  </tr>\n")
	}

	tb.WriteString("</table>")
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
