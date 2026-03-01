package table

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

const (
	// initialColWidthsCap is the initial capacity for table column widths.
	initialColWidthsCap = 12
)

// ensureColWidthCapacity ensures the colWidths slice can hold the given index.
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

// Extract extracts HTML table content and converts it to the specified format.
// This is the main entry point for table extraction that orchestrates the multi-step process.
//
// Deprecated: Use Processor.Extract with CellAccessor and NodeWalker interfaces for better testability.
// This function is maintained for backward compatibility.
func Extract(table *html.Node, tb *TrackedBuilder, tableFormat string, getCellAlignFunc func(*html.Node) CellAlignment,
	getColSpanFunc func(*html.Node) int, getRowSpanFunc func(*html.Node) int, getCellWidthFunc func(*html.Node) string,
	getTextContentFunc func(*html.Node) string, walkNodesFunc func(*html.Node, func(*html.Node) bool)) {
	if table == nil {
		return
	}

	// Create adapter using the function-based implementation
	accessor := &FuncCellAccessor{
		GetAlignmentFunc:   getCellAlignFunc,
		GetColSpanFunc:     getColSpanFunc,
		GetRowSpanFunc:     getRowSpanFunc,
		GetWidthFunc:       getCellWidthFunc,
		GetTextContentFunc: getTextContentFunc,
	}
	walker := &FuncNodeWalker{
		WalkFunc: walkNodesFunc,
	}

	processor := NewProcessor(accessor, walker)
	processor.Extract(table, tb, tableFormat)
}

// isStructureRow determines if a row contains only width definitions (no real content).
// Structure rows are used in Markdown tables to specify column widths.
func isStructureRow(cells []CellData) bool {
	hasWidthDefinitions := true
	hasRealContent := false

	for _, cell := range cells {
		if cell.Width == "" {
			hasWidthDefinitions = false
		}
		if cell.Text != " " && cell.Text != "" && cell.Text != "\u00a0" {
			hasRealContent = true
		}
	}

	return hasWidthDefinitions && !hasRealContent
}

// expandColspanCells expands cells with colspan > 1 into multiple placeholder cells.
// This is needed for Markdown format which doesn't support colspan.
func expandColspanCells(rawCells []CellData) []CellData {
	cells := make([]CellData, 0, len(rawCells))

	for _, rawCell := range rawCells {
		// Add the original cell
		cells = append(cells, rawCell)

		// Add placeholder cells for colspan > 1
		originalAlign := rawCell.Align
		for i := 1; i < rawCell.Colspan; i++ {
			cells = append(cells, CellData{
				Text:            " ",
				Align:           originalAlign,
				Colspan:         1,
				Rowspan:         rawCell.Rowspan,
				IsHeader:        rawCell.IsHeader,
				Width:           "",
				IsExpanded:      true,
				OriginalColspan: 1,
			})
		}
	}

	return cells
}

// collectColumnWidths extracts width definitions from a structure row.
func collectColumnWidths(cells []CellData, colWidths []string) []string {
	for i, cell := range cells {
		colWidths = ensureColWidthCapacity(colWidths, i)
		if cell.Width != "" {
			colWidths[i] = cell.Width
		}
	}
	return colWidths
}

// calculateMaxColumns finds the maximum number of columns across all rows.
func calculateMaxColumns(tableData [][]CellData) int {
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
func extractTableAsMarkdown(tableData [][]CellData, tb *TrackedBuilder, maxCols int, structureRowWidths []string) {
	// Pad rows to have consistent column count
	tableData = padTableColumns(tableData, maxCols)

	// Calculate column properties
	colAligns := calculateColumnAlignments(tableData, maxCols, structureRowWidths)
	colMaxWidths := calculateMaxColumnWidths(tableData, maxCols)

	// Filter out columns that are entirely empty expanded cells
	newToOldCol := filterExpandedColumns(tableData, maxCols)
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
func padTableColumns(tableData [][]CellData, maxCols int) [][]CellData {
	for i := range tableData {
		for len(tableData[i]) < maxCols {
			tableData[i] = append(tableData[i], CellData{Text: " ", Align: AlignDefault})
		}
	}
	return tableData
}

// calculateColumnAlignments determines column alignment using majority voting.
// Returns alignment strings in Markdown format (:---, :--:, ---:, etc.)
func calculateColumnAlignments(tableData [][]CellData, maxCols int, structureRowWidths []string) []string {
	colAligns := make([]string, maxCols)
	alignCounts := make([]AlignCount, maxCols)

	// Count alignments from all non-expanded cells
	for _, row := range tableData {
		for i := 0; i < maxCols && i < len(row); i++ {
			if !row[i].IsExpanded && row[i].Text != " " && row[i].Align != AlignDefault {
				switch row[i].Align {
				case AlignLeft:
					alignCounts[i].Left++
				case AlignCenter:
					alignCounts[i].Center++
				case AlignRight:
					alignCounts[i].Right++
				case AlignJustify:
					alignCounts[i].Justify++
				default:
					alignCounts[i].DefaultCount++
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
func determineColumnAlignment(counts AlignCount, firstRow []CellData, colIdx int) string {
	maxCount := 0
	majorityAlign := AlignDefault

	// Find the alignment with the most votes
	if counts.Left > maxCount {
		maxCount = counts.Left
		majorityAlign = AlignLeft
	}
	if counts.Center > maxCount {
		maxCount = counts.Center
		majorityAlign = AlignCenter
	}
	if counts.Right > maxCount {
		maxCount = counts.Right
		majorityAlign = AlignRight
	}
	if counts.Justify > maxCount {
		maxCount = counts.Justify
		majorityAlign = AlignJustify
	}

	// If no clear majority, use first row's alignment
	if maxCount == 0 && len(firstRow) > colIdx {
		majorityAlign = firstRow[colIdx].Align
	}

	// Check for mixed alignment (both left and right present)
	hasMixedAlignment := counts.Left > 0 && counts.Right > 0

	if hasMixedAlignment {
		return "---"
	}

	// Convert to Markdown alignment format
	switch majorityAlign {
	case AlignLeft:
		return ":---"
	case AlignCenter:
		return ":--:"
	case AlignRight:
		return "---:"
	case AlignJustify:
		return "---"
	default:
		return "---"
	}
}

// calculateMaxColumnWidths finds the maximum text width for each column.
func calculateMaxColumnWidths(tableData [][]CellData, maxCols int) []int {
	colMaxWidths := make([]int, maxCols)
	for _, row := range tableData {
		for j := 0; j < maxCols && j < len(row); j++ {
			textLen := len(row[j].Text)
			if textLen > colMaxWidths[j] {
				colMaxWidths[j] = textLen
			}
		}
	}
	return colMaxWidths
}

// filterExpandedColumns identifies columns that should be excluded.
// Returns a list of included column indices (columns with real content).
func filterExpandedColumns(tableData [][]CellData, maxCols int) []int {
	includeCol := make([]bool, maxCols)
	newToOldCol := make([]int, 0, maxCols)

	for j := 0; j < maxCols; j++ {
		// Check if this column has any non-expanded content
		allExpanded := true
		for _, row := range tableData {
			if j < len(row) && (!row[j].IsExpanded || (row[j].Text != " " && row[j].Text != "")) {
				allExpanded = false
				break
			}
		}

		includeCol[j] = !allExpanded
		if !allExpanded {
			newToOldCol = append(newToOldCol, j)
		}
	}

	return newToOldCol
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
func renderMarkdownRow(tb *TrackedBuilder, row []CellData, newToOldCol []int,
	colAligns []string, colMaxWidths []int, numCols int) {

	tb.WriteString("| ")
	for newJ, oldJ := range newToOldCol {
		cellText := " "
		if oldJ < len(row) {
			cellText = row[oldJ].Text
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
func extractTableAsHTML(tableData [][]CellData, tb *TrackedBuilder) {
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
func renderHTMLCell(tb *TrackedBuilder, cell CellData) {
	// Determine tag name
	tag := "td"
	if cell.IsHeader {
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
	if cell.OriginalColspan > 1 && !cell.IsExpanded {
		tb.WriteString(` colspan="`)
		tb.WriteString(strconv.Itoa(cell.OriginalColspan))
		tb.WriteString(`"`)
	}

	// Add rowspan attribute
	if cell.Rowspan > 1 {
		tb.WriteString(` rowspan="`)
		tb.WriteString(strconv.Itoa(cell.Rowspan))
		tb.WriteString(`"`)
	}

	// Write cell content
	tb.WriteString(">")
	tb.WriteString(cell.Text)
	tb.WriteString("</" + tag + ">\n")
}

// buildCellStyle constructs the style attribute value for a table cell.
func buildCellStyle(cell CellData) string {
	if cell.Align == AlignDefault && (cell.Width == "" || cell.IsExpanded) {
		return ""
	}

	var styleParts []string
	switch cell.Align {
	case AlignLeft:
		styleParts = append(styleParts, "text-align:left")
	case AlignCenter:
		styleParts = append(styleParts, "text-align:center")
	case AlignRight:
		styleParts = append(styleParts, "text-align:right")
	case AlignJustify:
		styleParts = append(styleParts, "text-align:justify")
	}

	if cell.Width != "" && !cell.IsExpanded {
		styleParts = append(styleParts, "width:"+cell.Width)
	}

	return strings.Join(styleParts, ";")
}
