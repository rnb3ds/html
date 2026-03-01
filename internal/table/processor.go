// Package table provides HTML table extraction and rendering functionality.
// This file contains the processor interface and implementation for table extraction.
package table

import (
	"strings"

	"golang.org/x/net/html"
)

// CellAccessor provides methods to access cell information from HTML nodes.
// This interface abstracts the cell attribute extraction, allowing for
// different implementations and easier testing.
type CellAccessor interface {
	// GetAlignment returns the text alignment of the cell.
	GetAlignment(node *html.Node) CellAlignment
	// GetColSpan returns the column span of the cell.
	GetColSpan(node *html.Node) int
	// GetRowSpan returns the row span of the cell.
	GetRowSpan(node *html.Node) int
	// GetWidth returns the width specification of the cell.
	GetWidth(node *html.Node) string
	// GetTextContent returns the text content of the node.
	GetTextContent(node *html.Node) string
}

// NodeWalker provides methods for walking the DOM tree.
type NodeWalker interface {
	// Walk traverses the DOM tree starting from node, calling callback for each node.
	// The callback returns false to stop traversal, true to continue.
	Walk(node *html.Node, callback func(*html.Node) bool)
}

// Processor handles table extraction from HTML nodes.
type Processor struct {
	cellAccessor CellAccessor
	nodeWalker   NodeWalker
}

// NewProcessor creates a new table Processor with the given accessor and walker.
func NewProcessor(ca CellAccessor, nw NodeWalker) *Processor {
	return &Processor{
		cellAccessor: ca,
		nodeWalker:   nw,
	}
}

// Extract extracts HTML table content and converts it to the specified format.
// This is the main method for table extraction using the Processor.
func (p *Processor) Extract(table *html.Node, tb *TrackedBuilder, tableFormat string) {
	if table == nil {
		return
	}

	// Ensure blank line before table for proper Markdown parsing
	EnsureNewline(tb)
	if tb.LastChar == '\n' {
		tb.WriteByte('\n')
	}

	// Step 1: Extract all row data from table
	tableData, colWidths := p.extractTableData(table, tableFormat)

	if len(tableData) == 0 {
		return
	}

	// Step 2: Determine maximum columns
	maxCols := calculateMaxColumns(tableData)

	// Step 3: Render in requested format
	switch sanitizeFormat(tableFormat) {
	case "html":
		extractTableAsHTML(tableData, tb)
	default: // "markdown"
		extractTableAsMarkdown(tableData, tb, maxCols, colWidths)
	}

	// Ensure blank line after table for proper Markdown parsing
	tb.WriteByte('\n')
	if tb.LastChar == '\n' {
		tb.WriteByte('\n')
	}
}

// extractTableData walks through table rows and extracts cell data.
func (p *Processor) extractTableData(table *html.Node, tableFormat string) ([][]CellData, []string) {
	var tableData [][]CellData
	colWidths := make([]string, 0, initialColWidthsCap)

	p.nodeWalker.Walk(table, func(node *html.Node) bool {
		if node.Type != html.ElementNode || node.Data != "tr" {
			return true
		}

		// Extract cells from this row
		rawCells := p.extractRowCells(node)
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
func (p *Processor) extractRowCells(rowNode *html.Node) []CellData {
	cells := make([]CellData, 0, 4)

	for child := rowNode.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode || (child.Data != "td" && child.Data != "th") {
			continue
		}

		cellText := sanitizeCellText(p.cellAccessor.GetTextContent(child))

		colspan := p.cellAccessor.GetColSpan(child)
		if colspan < 1 {
			colspan = 1
		}
		rowspan := p.cellAccessor.GetRowSpan(child)

		cells = append(cells, CellData{
			Text:            cellText,
			Align:           p.cellAccessor.GetAlignment(child),
			Colspan:         colspan,
			Rowspan:         rowspan,
			IsHeader:        child.Data == "th",
			Width:           p.cellAccessor.GetWidth(child),
			OriginalColspan: colspan,
		})
	}

	return cells
}

// sanitizeCellText cleans and normalizes cell text content.
func sanitizeCellText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return " "
	}
	return text
}

// sanitizeFormat normalizes the table format string.
func sanitizeFormat(format string) string {
	return strings.ToLower(strings.TrimSpace(format))
}

// FuncCellAccessor is a CellAccessor implementation using function pointers.
// This provides backward compatibility with the callback-based API.
type FuncCellAccessor struct {
	GetAlignmentFunc   func(*html.Node) CellAlignment
	GetColSpanFunc     func(*html.Node) int
	GetRowSpanFunc     func(*html.Node) int
	GetWidthFunc       func(*html.Node) string
	GetTextContentFunc func(*html.Node) string
}

// GetAlignment implements CellAccessor.
func (f *FuncCellAccessor) GetAlignment(node *html.Node) CellAlignment {
	if f.GetAlignmentFunc != nil {
		return f.GetAlignmentFunc(node)
	}
	return AlignDefault
}

// GetColSpan implements CellAccessor.
func (f *FuncCellAccessor) GetColSpan(node *html.Node) int {
	if f.GetColSpanFunc != nil {
		return f.GetColSpanFunc(node)
	}
	return 1
}

// GetRowSpan implements CellAccessor.
func (f *FuncCellAccessor) GetRowSpan(node *html.Node) int {
	if f.GetRowSpanFunc != nil {
		return f.GetRowSpanFunc(node)
	}
	return 1
}

// GetWidth implements CellAccessor.
func (f *FuncCellAccessor) GetWidth(node *html.Node) string {
	if f.GetWidthFunc != nil {
		return f.GetWidthFunc(node)
	}
	return ""
}

// GetTextContent implements CellAccessor.
func (f *FuncCellAccessor) GetTextContent(node *html.Node) string {
	if f.GetTextContentFunc != nil {
		return f.GetTextContentFunc(node)
	}
	return ""
}

// FuncNodeWalker is a NodeWalker implementation using a function pointer.
type FuncNodeWalker struct {
	WalkFunc func(*html.Node, func(*html.Node) bool)
}

// Walk implements NodeWalker.
func (f *FuncNodeWalker) Walk(node *html.Node, callback func(*html.Node) bool) {
	if f.WalkFunc != nil {
		f.WalkFunc(node, callback)
	}
}
