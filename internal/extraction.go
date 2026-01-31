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
	text           string
	align          cellAlign
	colspan        int
	rowspan        int
	isHeader       bool
	width          string  // Cell width (e.g., "100px", "1.0%", "auto")
	isExpanded     bool    // True if this cell was created from colspan expansion
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

	// containsWord checks if a word/expression exists as a whole word in the text
	// This prevents partial matches like "right" in "upright"
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

	// Get cell alignment from style attribute and align attribute
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

	// Get colspan attribute
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

	// Get rowspan attribute
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

	// Get cell width from style attribute or width attribute
	getCellWidth := func(n *html.Node) string {
		// First check for width attribute
		for _, attr := range n.Attr {
			if strings.ToLower(attr.Key) == "width" {
				widthVal := strings.TrimSpace(attr.Val)
				if widthVal != "" && widthVal != "0" {
					return widthVal
				}
			}
		}
		// Then check for width in style attribute
		for _, attr := range n.Attr {
			if strings.ToLower(attr.Key) == "style" {
				style := attr.Val
				// Extract width from style attribute
				// Common patterns: "width: 100px", "width:1.0%", "width: auto"
				widthLower := strings.ToLower(style)
				if idx := strings.Index(widthLower, "width:"); idx >= 0 {
					// Find the end of the width value (up to ; or end of string)
					start := idx + 6 // len("width:")
					// Skip whitespace
					for start < len(style) && (style[start] == ' ' || style[start] == '\t') {
						start++
					}
					end := start
					// Find the end of the width value
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

	// Extract table structure
	var tableData [][]cellData
	var maxCols int
	// Global column widths array (initialized with 12 common columns)
	colWidths := make([]string, 12)

	// First pass: calculate the actual logical column count
	// by summing colspan values for each row
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
			hasWidthDefinitions := true     // Track if all cells have width definitions
			hasNonEmptyContent := false      // Track if row has actual content
			hasComplexChildElements := false // Track if cells have complex child elements

			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode && (child.Data == "td" || child.Data == "th") {
					cellText := strings.TrimSpace(GetTextContent(child))
					if cellText == "" {
						cellText = " "
					}
					// Check if this cell has width definition
					cellWidth := getCellWidth(child)
					if cellWidth == "" {
						hasWidthDefinitions = false
					}
					// Check if this row contains actual content
					if cellText != " " {
						hasNonEmptyContent = true
					}
					// Check if the cell has complex child elements (not just whitespace)
					if child.FirstChild != nil {
						for grandchild := child.FirstChild; grandchild != nil; grandchild = grandchild.NextSibling {
							if grandchild.Type == html.ElementNode {
								// Check if it's a meaningful element (not just <br>)
								if grandchild.Data != "br" && grandchild.Data != "hr" {
									// Check if the element has content
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
						originalColspan: colspan, // Store original colspan for HTML output
					})
				}
			}

			// Determine if this is a structure row (pure width definition row)
			// A structure row has:
			// 1. All cells with width definitions
			// 2. No non-empty content
			// 3. No complex child elements
			isStructureRow := hasWidthDefinitions && !hasNonEmptyContent && !hasComplexChildElements

			// Process cells based on format
			var cells []cellData
			if tableFormat == "html" {
				// For HTML format: preserve original structure (no expansion)
				// Structure rows keep their colspan for proper width definitions
				cells = rawCells
			} else {
				// For Markdown format: expand colspan cells
				cells = make([]cellData, 0, len(rawCells))
				for _, rawCell := range rawCells {
					cells = append(cells, rawCell)
					// For colspan > 1, add empty cells to expand (Markdown doesn't support colspan)
					// Preserve the original cell's alignment for ALL expanded cells to maintain
					// visual consistency with the original table structure
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

			// If this is a structure row, extract and save width information globally
			if isStructureRow {
				// For HTML format, use rawCells (before expansion)
				// For Markdown format, use cells (after expansion)
				widthSourceCells := rawCells
				for i, cell := range widthSourceCells {
					if i < len(colWidths) && cell.width != "" {
						colWidths[i] = cell.width
					}
				}
			}

			// For HTML format: preserve structure rows as they contain important width information
			// For Markdown format: skip structure rows as they don't contain displayable content
			if tableFormat == "html" {
				// HTML format: include all rows (including structure rows)
				if len(cells) > 0 {
					tableData = append(tableData, cells)
				}
			} else {
				// Markdown format: skip structure rows
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
	// Normalize rows to same length
	for i := range tableData {
		for len(tableData[i]) < maxCols {
			tableData[i] = append(tableData[i], cellData{text: " ", align: 3 /* alignDefault */})
		}
	}

	// Determine column alignments by scanning all rows
	// This handles cases where header row has no alignment but data rows do
	colAligns := make([]string, maxCols)
	colWidths := make([]string, maxCols) // Track column widths
	colAlignsInfo := make([]string, maxCols) // Track alignment info for comments
	hasWidths := false
	hasJustify := false // Track if any column uses justify alignment

	// First, copy structure row widths if available
	if len(structureRowWidths) > 0 {
		for i := 0; i < maxCols && i < len(structureRowWidths); i++ {
			if structureRowWidths[i] != "" {
				colWidths[i] = structureRowWidths[i]
				hasWidths = true
			}
		}
	}

	// Count alignment occurrences for each column
	// alignCounts[colIdx][alignType] = count
	type alignCount struct {
		left, center, right, justify, defaultCount int
	}
	alignCounts := make([]alignCount, maxCols)

	// First pass: collect all widths (from any row) and count alignments
	for _, row := range tableData {
		for i := 0; i < maxCols && i < len(row); i++ {
			// Collect width from any row (not just header)
			if row[i].width != "" && colWidths[i] == "" {
				colWidths[i] = row[i].width
				hasWidths = true
			}
			// Count all non-empty cells for alignment (not just non-default align)
			// Skip expanded cells as they're just artifacts of colspan expansion
			// This ensures empty cells with default alignment don't skew the majority vote
			if !row[i].isExpanded && row[i].text != " " && row[i].align != alignDefault {
				// Only count cells with explicit non-default alignment
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

	// Second pass: determine alignment for each column based on majority vote
	if len(tableData) > 0 {
		for i := 0; i < maxCols; i++ {
			// Find the most common alignment for this column
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

			// If no alignment found in any row (all counts are 0), check first row as fallback
			if maxCount == 0 && len(tableData[0]) > i {
				majorityAlign = tableData[0][i].align
			}

			// Check for mixed alignment columns (columns with both left and right alignments)
			// For such columns, use default alignment to avoid misalignment
			hasMixedAlignment := alignCounts[i].left > 0 && alignCounts[i].right > 0

			if hasMixedAlignment {
				// Use default alignment for mixed columns
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
					// Markdown doesn't support justify natively, use default alignment
					colAligns[i] = "---"
					colAlignsInfo[i] = "justify"
					hasJustify = true
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

	// Add column metadata comment if widths are present or non-default alignments exist
	hasNonDefaultAlign := false
	for i := range colAlignsInfo {
		if colAlignsInfo[i] != "default" && colAlignsInfo[i] != "" {
			hasNonDefaultAlign = true
			break
		}
	}

	if hasWidths || hasJustify || hasNonDefaultAlign {
		tb.WriteString("<!-- Table metadata: ")
		for i := range colWidths {
			hasMeta := false
			if colWidths[i] != "" {
				tb.WriteString("col:")
				tb.WriteString(strconv.Itoa(i+1))
				tb.WriteString("=width:")
				tb.WriteString(colWidths[i])
				hasMeta = true
			}
			// Include alignment info for all columns with non-default alignment
			if colAlignsInfo[i] != "default" && colAlignsInfo[i] != "" {
				if !hasMeta {
					tb.WriteString("col:")
					tb.WriteString(strconv.Itoa(i+1))
					hasMeta = true
				} else {
					tb.WriteString(",")
				}
				tb.WriteString("align:")
				tb.WriteString(colAlignsInfo[i])
			}
			if hasMeta {
				tb.WriteString(" ")
			}
		}
		tb.WriteString("-->\n")
	}

	// Calculate column widths for visual alignment
	// Find the maximum content width for each column
	colMaxWidths := make([]int, maxCols)
	for _, row := range tableData {
		for j := 0; j < maxCols && j < len(row); j++ {
			textLen := len(row[j].text)
			if textLen > colMaxWidths[j] {
				colMaxWidths[j] = textLen
			}
		}
	}

	// Write table rows
	for i, row := range tableData {
		tb.WriteString("| ")
		for j := 0; j < maxCols; j++ {
			var cellText string
			if j < len(row) {
				cellText = row[j].text
			} else {
				cellText = " "
			}

			// Apply visual alignment based on column alignment
			maxWidth := colMaxWidths[j]
			if maxWidth < 3 { // Minimum width for readability
				maxWidth = 3
			}
			textLen := len(cellText)

			// Pad cell content based on alignment
			switch colAligns[j] {
			case ":---": // Left align
				tb.WriteString(cellText)
				tb.WriteString(strings.Repeat(" ", maxWidth-textLen))
			case "---:": // Right align
				tb.WriteString(strings.Repeat(" ", maxWidth-textLen))
				tb.WriteString(cellText)
			case ":--:": // Center align
				leftPad := (maxWidth - textLen) / 2
				rightPad := maxWidth - textLen - leftPad
				tb.WriteString(strings.Repeat(" ", leftPad))
				tb.WriteString(cellText)
				tb.WriteString(strings.Repeat(" ", rightPad))
			default: // No alignment (default)
				tb.WriteString(cellText)
				tb.WriteString(strings.Repeat(" ", maxWidth-textLen))
			}

			if j < maxCols-1 {
				tb.WriteString(" | ")
			}
		}
		tb.WriteString(" |\n")

		// Write separator after header row
		if i == 0 {
			tb.WriteString("| ")
			tb.WriteString(strings.Join(colAligns, " | "))
			tb.WriteString(" |\n")
		}
	}
}

// extractTableAsHTML outputs table in HTML format with alignment, merges, and column widths
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

			// Build style attribute with alignment and width
			var styleParts []string
			// Add alignment
			switch cell.align {
			case 0: // alignLeft
				styleParts = append(styleParts, "text-align:left")
			case 1: // alignCenter
				styleParts = append(styleParts, "text-align:center")
			case 2: // alignRight
				styleParts = append(styleParts, "text-align:right")
			case 3: // alignJustify
				styleParts = append(styleParts, "text-align:justify")
			}
			// Add width if present (only for structure rows)
			// Don't add width to content rows as they inherit column width from structure row
			if cell.width != "" && !cell.isExpanded && cell.text == " " {
				styleParts = append(styleParts, "width:"+cell.width)
			}
			// Write style attribute if we have any style parts
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

			// IMPORTANT: Don't use originalColspan for HTML output
			// Cells have already been expanded for Markdown, so we use colspan=1 for all cells
			// The original colspan structure is preserved by the fact that we have the right number of cells
			// Only use colspan for rows that were NOT expanded (structure rows)
			if cell.originalColspan > 1 && !cell.isExpanded {
				tb.WriteString(` colspan="`)
				tb.WriteString(strconv.Itoa(cell.originalColspan))
				tb.WriteString(`"`)
			}

			// Add rowspan
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
