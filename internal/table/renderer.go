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

func (r *RendererRegistry) register(format string, renderer Renderer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.renderers[strings.ToLower(format)] = renderer
}

// MarkdownRenderer renders tables in Markdown format.
type MarkdownRenderer struct{}

// Format returns "markdown".
func (r *MarkdownRenderer) Format() string {
	return "markdown"
}

// Render renders the table data in Markdown format.
// Delegates to extractTableAsMarkdown to avoid code duplication.
func (r *MarkdownRenderer) Render(tableData [][]CellData, tb *TrackedBuilder, maxCols int, colWidths []string) {
	extractTableAsMarkdown(tableData, tb, maxCols, colWidths)
}

// HTMLRenderer renders tables in HTML format.
type HTMLRenderer struct{}

// Format returns "html".
func (r *HTMLRenderer) Format() string {
	return "html"
}

// Render renders the table data in HTML format.
// Delegates to extractTableAsHTML to avoid code duplication.
func (r *HTMLRenderer) Render(tableData [][]CellData, tb *TrackedBuilder, maxCols int, colWidths []string) {
	extractTableAsHTML(tableData, tb)
}

// Note: The following functions are defined in render.go:
//   - renderMarkdownRow: renders a single table row in Markdown format
//   - renderHTMLCell: renders a single table cell in HTML format
//   - buildCellStyle: constructs the style attribute value for a table cell
