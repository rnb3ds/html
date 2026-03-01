// Package table provides HTML table extraction and rendering functionality.
// This file contains the Renderer interface for pluggable table output formats.
package table

import (
	"strings"
	"sync"
)

// Renderer defines the interface for table rendering implementations.
// Different implementations can output tables in various formats (Markdown, HTML, etc.)
type Renderer interface {
	// Render renders the table data to the TrackedBuilder.
	Render(tableData [][]CellData, tb *TrackedBuilder, maxCols int, colWidths []string)
	// Format returns the format name (e.g., "markdown", "html").
	Format() string
}

// RendererRegistry manages registered table renderers.
type RendererRegistry struct {
	mu        sync.RWMutex
	renderers map[string]Renderer
}

// globalRegistry is the global renderer registry.
var globalRegistry = &RendererRegistry{
	renderers: make(map[string]Renderer),
}

func init() {
	// Register default renderers
	globalRegistry.register("markdown", &MarkdownRenderer{})
	globalRegistry.register("html", &HTMLRenderer{})
}

// RegisterRenderer registers a renderer for a given format name.
// If a renderer with the same name already exists, it will be replaced.
func RegisterRenderer(format string, renderer Renderer) {
	globalRegistry.register(format, renderer)
}

// GetRenderer returns the renderer for the given format.
// Returns nil if no renderer is registered for the format.
func GetRenderer(format string) Renderer {
	return globalRegistry.get(format)
}

func (r *RendererRegistry) register(format string, renderer Renderer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.renderers[strings.ToLower(format)] = renderer
}

func (r *RendererRegistry) get(format string) Renderer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.renderers[strings.ToLower(format)]
}

// MarkdownRenderer renders tables in Markdown format.
type MarkdownRenderer struct{}

// Format returns "markdown".
func (r *MarkdownRenderer) Format() string {
	return "markdown"
}

// Render renders the table data in Markdown format.
func (r *MarkdownRenderer) Render(tableData [][]CellData, tb *TrackedBuilder, maxCols int, colWidths []string) {
	// Pad rows to have consistent column count
	tableData = padTableColumns(tableData, maxCols)

	// Calculate column properties
	colAligns := calculateColumnAlignments(tableData, maxCols, colWidths)
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

// HTMLRenderer renders tables in HTML format.
type HTMLRenderer struct{}

// Format returns "html".
func (r *HTMLRenderer) Format() string {
	return "html"
}

// Render renders the table data in HTML format.
func (r *HTMLRenderer) Render(tableData [][]CellData, tb *TrackedBuilder, maxCols int, colWidths []string) {
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

// renderMarkdownRow renders a single table row in Markdown format.
// Note: This is defined in render.go
// func renderMarkdownRow(tb *TrackedBuilder, row []CellData, newToOldCol []int,
// 	colAligns []string, colMaxWidths []int, numCols int) { ... }

// renderHTMLCell renders a single table cell in HTML format.
// Note: This is defined in render.go
// func renderHTMLCell(tb *TrackedBuilder, cell CellData) { ... }

// buildCellStyle constructs the style attribute value for a table cell.
// Note: This is defined in render.go
// func buildCellStyle(cell CellData) string { ... }

// intToString converts an integer to string without importing strconv.
// Note: This is defined in render.go
// func intToString(n int) string { ... }
