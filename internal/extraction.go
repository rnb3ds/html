// Package internal provides implementation details for the cybergodev/html library.
// It contains content extraction, table processing, and text manipulation functionality
// that is not part of the public API.
package internal

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

const (
	// Initial capacity for table column widths slice
	initialColWidthsCap = 12
)

type trackedBuilder struct {
	*strings.Builder
	lastChar byte
}

// alignCount tracks the number of cells with each alignment type for a column.
type alignCount struct {
	left, center, right, justify, defaultCount int
}

func newTrackedBuilder(sb *strings.Builder) *trackedBuilder {
	return &trackedBuilder{
		Builder:  sb,
		lastChar: 0,
	}
}

// ensureColWidthCapacity ensures the colWidths slice can hold the given index
func ensureColWidthCapacity(colWidths []string, index int) []string {
	if index < len(colWidths) {
		return colWidths
	}
	// Grow the slice to accommodate the index
	newCap := index + 1
	if cap(colWidths) >= newCap {
		// Extend to available capacity
		return colWidths[:newCap]
	}
	// Allocate new slice with larger capacity
	newSlice := make([]string, newCap, newCap+initialColWidthsCap)
	copy(newSlice, colWidths)
	return newSlice
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
		textData := node.Data
		// Replace non-breaking spaces (U+00A0) with regular spaces
		// The HTML parser converts &nbsp;, &#160;, &#xa0; to U+00A0 automatically
		textData = normalizeNonBreakingSpaces(textData)
		textData = ReplaceHTMLEntities(textData)
		// Replace internal newlines with spaces for multi-line text in HTML
		textData = strings.ReplaceAll(textData, "\n", " ")
		textData = strings.ReplaceAll(textData, "\r", "")

		if content := strings.TrimSpace(textData); content != "" {
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

// extractTableTracked extracts HTML table content and converts it to the specified format.
// This is the main entry point for table extraction that orchestrates the multi-step process.
func extractTableTracked(table *html.Node, tb *trackedBuilder, tableFormat string) {
	if table == nil {
		return
	}

	// Ensure blank line before table for proper Markdown parsing
	ensureNewlineTracked(tb)
	if tb.lastChar == '\n' {
		tb.WriteByte('\n')
	}

	// Step 1: Extract all row data from table
	tableData, colWidths := extractTableData(table, tableFormat)

	if len(tableData) == 0 {
		return
	}

	// Step 2: Determine maximum columns
	maxCols := calculateMaxColumns(tableData)

	// Step 3: Render in requested format
	switch strings.ToLower(strings.TrimSpace(tableFormat)) {
	case "html":
		extractTableAsHTML(tableData, tb)
	default: // "markdown"
		extractTableAsMarkdown(tableData, tb, maxCols, colWidths)
	}

	// Ensure blank line after table for proper Markdown parsing
	tb.WriteByte('\n')
	if tb.lastChar == '\n' {
		tb.WriteByte('\n')
	}
}

// extractTableData walks through table rows and extracts cell data.
// Returns table rows with cell metadata and column widths from structure rows.
func extractTableData(table *html.Node, tableFormat string) ([][]cellData, []string) {
	var tableData [][]cellData
	colWidths := make([]string, 0, initialColWidthsCap)

	WalkNodes(table, func(node *html.Node) bool {
		if node.Type != html.ElementNode || node.Data != "tr" {
			return true
		}

		// Extract cells from this row
		rawCells := extractRowCells(node)
		if len(rawCells) == 0 {
			return false
		}

		// Determine if this is a structure row (width definitions only, no real content)
		isStructureRow := isStructureRow(rawCells)

		// Expand cells with colspan for Markdown format
		cells := rawCells
		if tableFormat != "html" {
			cells = expandColspanCells(rawCells)
		}

		// Collect column widths from structure rows
		if isStructureRow {
			colWidths = collectColumnWidths(rawCells, colWidths)
		}

		// Add row to table data (skip structure rows for Markdown)
		if tableFormat == "html" {
			tableData = append(tableData, cells)
		} else if !isStructureRow {
			tableData = append(tableData, cells)
		}

		return false
	})

	return tableData, colWidths
}

// extractRowCells extracts all cell data from a single table row (tr element).
func extractRowCells(rowNode *html.Node) []cellData {
	cells := make([]cellData, 0, 4)

	for child := rowNode.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode || (child.Data != "td" && child.Data != "th") {
			continue
		}

		cellText := strings.TrimSpace(GetTextContent(child))
		if cellText == "" {
			cellText = " "
		}

		colspan := getColSpan(child)
		if colspan < 1 {
			colspan = 1
		}
		rowspan := getRowSpan(child)

		cells = append(cells, cellData{
			text:            cellText,
			align:           getCellAlign(child),
			colspan:         colspan,
			rowspan:         rowspan,
			isHeader:        child.Data == "th",
			width:           getCellWidth(child),
			originalColspan: colspan,
		})
	}

	return cells
}

// isStructureRow determines if a row contains only width definitions (no real content).
// Structure rows are used in Markdown tables to specify column widths.
func isStructureRow(cells []cellData) bool {
	hasWidthDefinitions := true
	hasComplexChildElements := false
	hasRealContent := false

	for _, cell := range cells {
		if cell.width == "" {
			hasWidthDefinitions = false
		}
		if cell.text != " " && cell.text != "" && cell.text != "\u00a0" {
			hasRealContent = true
		}
	}

	return hasWidthDefinitions && !hasRealContent && !hasComplexChildElements
}

// expandColspanCells expands cells with colspan > 1 into multiple placeholder cells.
// This is needed for Markdown format which doesn't support colspan.
func expandColspanCells(rawCells []cellData) []cellData {
	cells := make([]cellData, 0, len(rawCells))

	for _, rawCell := range rawCells {
		// Add the original cell
		cells = append(cells, rawCell)

		// Add placeholder cells for colspan > 1
		originalAlign := rawCell.align
		for i := 1; i < rawCell.colspan; i++ {
			cells = append(cells, cellData{
				text:            " ",
				align:           originalAlign,
				colspan:         1,
				rowspan:         rawCell.rowspan,
				isHeader:        rawCell.isHeader,
				width:           "",
				isExpanded:      true,
				originalColspan: 1,
			})
		}
	}

	return cells
}

// collectColumnWidths extracts width definitions from a structure row.
func collectColumnWidths(cells []cellData, colWidths []string) []string {
	for i, cell := range cells {
		colWidths = ensureColWidthCapacity(colWidths, i)
		if cell.width != "" {
			colWidths[i] = cell.width
		}
	}
	return colWidths
}

// calculateMaxColumns finds the maximum number of columns across all rows.
func calculateMaxColumns(tableData [][]cellData) int {
	maxCols := 0
	for _, row := range tableData {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}
	return maxCols
}

// extractTableAsMarkdown outputs table in Markdown format with alignment.
// Note: Column widths are included as HTML comments since Markdown doesn't support column widths.
func extractTableAsMarkdown(tableData [][]cellData, tb *trackedBuilder, maxCols int, structureRowWidths []string) {
	// Pad rows to have consistent column count
	tableData = padTableColumns(tableData, maxCols)

	// Calculate column properties
	colAligns := calculateColumnAlignments(tableData, maxCols, structureRowWidths)
	colMaxWidths := calculateMaxColumnWidths(tableData, maxCols)

	// Filter out columns that are entirely empty expanded cells
	_, newToOldCol := filterExpandedColumns(tableData, maxCols)
	numIncludedCols := len(newToOldCol)

	// Build arrays for included columns only
	includedColAligns := filterArray(colAligns, newToOldCol)
	includedColMaxWidths := filterIntArray(colMaxWidths, newToOldCol)

	// Ensure minimum width for alignment markers
	for i := range includedColMaxWidths {
		if includedColMaxWidths[i] < 3 {
			includedColMaxWidths[i] = 3
		}
	}

	// Render table rows with alignment separator after the first row
	if len(tableData) > 0 {
		// Render first row (header)
		renderMarkdownRow(tb, tableData[0], newToOldCol, includedColAligns, includedColMaxWidths, numIncludedCols)

		// Add alignment separator after header row (required by Markdown)
		tb.WriteString("| ")
		tb.WriteString(strings.Join(includedColAligns, " | "))
		tb.WriteString(" |\n")

		// Render remaining rows
		for i := 1; i < len(tableData); i++ {
			renderMarkdownRow(tb, tableData[i], newToOldCol, includedColAligns, includedColMaxWidths, numIncludedCols)
		}
	}
}

// padTableColumns ensures all rows have the same number of columns.
func padTableColumns(tableData [][]cellData, maxCols int) [][]cellData {
	for i := range tableData {
		for len(tableData[i]) < maxCols {
			tableData[i] = append(tableData[i], cellData{text: " ", align: alignDefault})
		}
	}
	return tableData
}

// calculateColumnAlignments determines column alignment using majority voting.
// Returns alignment strings in Markdown format (:---, :--:, ---:, etc.)
func calculateColumnAlignments(tableData [][]cellData, maxCols int, structureRowWidths []string) []string {
	colAligns := make([]string, maxCols)
	alignCounts := make([]alignCount, maxCols)

	// Count alignments from all non-expanded cells
	for _, row := range tableData {
		for i := 0; i < maxCols && i < len(row); i++ {
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

	// Determine majority alignment for each column
	if len(tableData) > 0 {
		for i := 0; i < maxCols; i++ {
			colAligns[i] = determineColumnAlignment(alignCounts[i], tableData[0], i)
		}
	} else {
		for i := range colAligns {
			colAligns[i] = "---"
		}
	}

	return colAligns
}

// determineColumnAlignment picks the majority alignment for a single column.
func determineColumnAlignment(counts alignCount, firstRow []cellData, colIdx int) string {
	maxCount := 0
	majorityAlign := alignDefault

	// Find the alignment with the most votes
	if counts.left > maxCount {
		maxCount = counts.left
		majorityAlign = alignLeft
	}
	if counts.center > maxCount {
		maxCount = counts.center
		majorityAlign = alignCenter
	}
	if counts.right > maxCount {
		maxCount = counts.right
		majorityAlign = alignRight
	}
	if counts.justify > maxCount {
		maxCount = counts.justify
		majorityAlign = alignJustify
	}

	// If no clear majority, use first row's alignment
	if maxCount == 0 && len(firstRow) > colIdx {
		majorityAlign = firstRow[colIdx].align
	}

	// Check for mixed alignment (both left and right present)
	hasMixedAlignment := counts.left > 0 && counts.right > 0

	if hasMixedAlignment {
		return "---"
	}

	// Convert to Markdown alignment format
	switch majorityAlign {
	case alignLeft:
		return ":---"
	case alignCenter:
		return ":--:"
	case alignRight:
		return "---:"
	case alignJustify:
		return "---"
	default:
		return "---"
	}
}

// calculateMaxColumnWidths finds the maximum text width for each column.
func calculateMaxColumnWidths(tableData [][]cellData, maxCols int) []int {
	colMaxWidths := make([]int, maxCols)
	for _, row := range tableData {
		for j := 0; j < maxCols && j < len(row); j++ {
			textLen := len(row[j].text)
			if textLen > colMaxWidths[j] {
				colMaxWidths[j] = textLen
			}
		}
	}
	return colMaxWidths
}

// filterExpandedColumns identifies columns that should be excluded.
// Returns a list of included column indices (columns with real content).
func filterExpandedColumns(tableData [][]cellData, maxCols int) ([]int, []int) {
	includeCol := make([]bool, maxCols)
	newToOldCol := make([]int, 0, maxCols)

	for j := 0; j < maxCols; j++ {
		// Check if this column has any non-expanded content
		allExpanded := true
		for _, row := range tableData {
			if j < len(row) && (!row[j].isExpanded || (row[j].text != " " && row[j].text != "")) {
				allExpanded = false
				break
			}
		}

		includeCol[j] = !allExpanded
		if !allExpanded {
			newToOldCol = append(newToOldCol, j)
		}
	}

	// Build mapping from new to old indices
	oldToNewCol := make([]int, maxCols)
	numIncluded := 0
	for j, included := range includeCol {
		if included {
			oldToNewCol[j] = numIncluded
			numIncluded++
		}
	}

	return oldToNewCol, newToOldCol
}

// filterArray filters a string array to include only specified indices.
func filterArray(arr []string, indices []int) []string {
	result := make([]string, len(indices))
	for i, idx := range indices {
		if idx < len(arr) {
			result[i] = arr[idx]
		}
	}
	return result
}

// filterIntArray filters an int array to include only specified indices.
func filterIntArray(arr []int, indices []int) []int {
	result := make([]int, len(indices))
	for i, idx := range indices {
		if idx < len(arr) {
			result[i] = arr[idx]
		}
	}
	return result
}

// renderMarkdownRow renders a single table row in Markdown format.
func renderMarkdownRow(tb *trackedBuilder, row []cellData, newToOldCol []int,
	colAligns []string, colMaxWidths []int, numCols int) {

	tb.WriteString("| ")
	for newJ, oldJ := range newToOldCol {
		cellText := " "
		if oldJ < len(row) {
			cellText = row[oldJ].text
		}

		maxWidth := colMaxWidths[newJ]
		textLen := len(cellText)

		// Apply alignment-based padding
		switch colAligns[newJ] {
		case ":---": // left
			tb.WriteString(cellText)
			tb.WriteString(strings.Repeat(" ", maxWidth-textLen))
		case "---:": // right
			tb.WriteString(strings.Repeat(" ", maxWidth-textLen))
			tb.WriteString(cellText)
		case ":--:": // center
			leftPad := (maxWidth - textLen) / 2
			rightPad := maxWidth - textLen - leftPad
			tb.WriteString(strings.Repeat(" ", leftPad))
			tb.WriteString(cellText)
			tb.WriteString(strings.Repeat(" ", rightPad))
		default: // left (default)
			tb.WriteString(cellText)
			tb.WriteString(strings.Repeat(" ", maxWidth-textLen))
		}

		if newJ < numCols-1 {
			tb.WriteString(" | ")
		}
	}
	tb.WriteString(" |\n")
}

// extractTableAsHTML outputs table in HTML format with proper attributes.
func extractTableAsHTML(tableData [][]cellData, tb *trackedBuilder) {
	tb.WriteString("<table>\n")

	for _, row := range tableData {
		tb.WriteString("  <tr>\n")
		for _, cell := range row {
			renderHTMLCell(tb, cell)
		}
		tb.WriteString("  </tr>\n")
	}

	tb.WriteString("</table>")
}

// renderHTMLCell renders a single table cell in HTML format.
func renderHTMLCell(tb *trackedBuilder, cell cellData) {
	// Determine tag name
	tag := "td"
	if cell.isHeader {
		tag = "th"
	}
	tb.WriteString("    <" + tag)

	// Add style attribute
	style := buildCellStyle(cell)
	if style != "" {
		tb.WriteString(` style="`)
		tb.WriteString(style)
		tb.WriteString(`"`)
	}

	// Add colspan attribute
	if cell.originalColspan > 1 && !cell.isExpanded {
		tb.WriteString(` colspan="`)
		tb.WriteString(strconv.Itoa(cell.originalColspan))
		tb.WriteString(`"`)
	}

	// Add rowspan attribute
	if cell.rowspan > 1 {
		tb.WriteString(` rowspan="`)
		tb.WriteString(strconv.Itoa(cell.rowspan))
		tb.WriteString(`"`)
	}

	// Write cell content
	tb.WriteString(">")
	tb.WriteString(cell.text)
	tb.WriteString("</" + tag + ">\n")
}

// buildCellStyle constructs the style attribute value for a table cell.
func buildCellStyle(cell cellData) string {
	if cell.align == alignDefault && (cell.width == "" || cell.isExpanded) {
		return ""
	}

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

	return strings.Join(styleParts, ";")
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
