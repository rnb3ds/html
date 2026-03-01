package internal

import (
	"strings"
	"testing"

	stdxhtml "golang.org/x/net/html"
)

// TestTableColumnWidths tests the table column width collection functionality.
// These tests verify that tables with width definitions are properly extracted.

func TestCollectColumnWidths(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		html        string
		minCells    int
		description string
	}{
		{
			name: "table with width definitions",
			html: `
				<table>
					<tr>
						<td width="100px">Cell 1</td>
						<td width="200px">Cell 2</td>
						<td width="150px">Cell 3</td>
					</tr>
				</table>
			`,
			minCells:    3,
			description: "Should extract row with 3 cells",
		},
		{
			name: "table without width definitions",
			html: `
				<table>
					<tr>
						<td>Cell 1</td>
						<td>Cell 2</td>
					</tr>
				</table>
			`,
			minCells:    2,
			description: "Should extract row with 2 cells",
		},
		{
			name: "mixed widths and percentages",
			html: `
				<table>
					<tr>
						<td width="100px">A</td>
						<td width="50%">B</td>
						<td>C</td>
					</tr>
				</table>
			`,
			minCells:    3,
			description: "Should extract row with 3 cells",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := stdxhtml.Parse(strings.NewReader(tc.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			// Find table
			table := FindElementByTag(doc, "table")
			if table == nil {
				t.Fatal("Table not found")
			}

			// Use the public API to extract table content
			var sb strings.Builder
			ExtractTextWithStructureAndImages(table, &sb, 0, nil, "markdown")

			// Verify content was extracted
			result := sb.String()
			if result == "" {
				t.Error("No table content extracted")
			}

			// Count cells by counting | characters in the output
			cellCount := strings.Count(result, "|") / 2 // Each cell has | before and after
			if cellCount < tc.minCells {
				t.Errorf("%s: extracted %d cells, want at least %d", tc.description, cellCount, tc.minCells)
			}
		})
	}
}

func TestTableStructureRowDetection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		html         string
		expectedRows int
		description  string
	}{
		{
			name: "normal data rows",
			html: `
				<table>
					<tr><td>Data 1</td><td>Data 2</td></tr>
					<tr><td>Data 3</td><td>Data 4</td></tr>
				</table>
			`,
			expectedRows: 2,
			description:  "All rows are data rows",
		},
		{
			name: "structure row with widths",
			html: `
				<table>
					<tr><td width="100px"></td><td width="200px"></td></tr>
					<tr><td>Data 1</td><td>Data 2</td></tr>
				</table>
			`,
			expectedRows: 1, // Only data row (structure row excluded)
			description:  "Structure row excluded from Markdown output",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := stdxhtml.Parse(strings.NewReader(tc.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			table := FindElementByTag(doc, "table")
			if table == nil {
				t.Fatal("Table not found")
			}

			// Use the public API to extract table content
			var sb strings.Builder
			ExtractTextWithStructureAndImages(table, &sb, 0, nil, "markdown")

			// Count rows by counting lines with | characters
			result := sb.String()
			lines := strings.Split(strings.TrimSpace(result), "\n")
			dataRowCount := 0
			for _, line := range lines {
				if strings.Contains(line, "|") && !strings.Contains(line, "| ---") && !strings.Contains(line, "|:--") && !strings.Contains(line, "|---:") {
					dataRowCount++
				}
			}

			if dataRowCount != tc.expectedRows {
				t.Errorf("%s: extracted %d data rows, want %d", tc.description, dataRowCount, tc.expectedRows)
			}
		})
	}
}

func TestTableCellWidthParsing(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		html          string
		expectedWidth string
	}{
		{
			name:          "pixel width",
			html:          `<table><tr><td width="100px">Cell</td></tr></table>`,
			expectedWidth: "100px",
		},
		{
			name:          "percentage width",
			html:          `<table><tr><td width="50%">Cell</td></tr></table>`,
			expectedWidth: "50%",
		},
		{
			name:          "width in style attribute",
			html:          `<table><tr><td style="width: 200px">Cell</td></tr></table>`,
			expectedWidth: "200px",
		},
		{
			name:          "no width",
			html:          `<table><tr><td>Cell</td></tr></table>`,
			expectedWidth: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := stdxhtml.Parse(strings.NewReader(tc.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			// Find the td element
			var td *stdxhtml.Node
			WalkNodes(doc, func(n *stdxhtml.Node) bool {
				if n.Type == stdxhtml.ElementNode && n.Data == "td" {
					td = n
					return false // Stop after finding first TD
				}
				return true
			})

			if td == nil {
				t.Fatal("TD element not found")
			}

			width := getCellWidth(td)
			if width != tc.expectedWidth {
				t.Errorf("getCellWidth() = %q, want %q", width, tc.expectedWidth)
			}
		})
	}
}

func TestTableColspanExpansion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		html            string
		expectedContent string
		description     string
	}{
		{
			name: "single colspan",
			html: `
				<table>
					<tr><td colspan="2">Spans 2</td><td>Normal</td></tr>
				</table>
			`,
			expectedContent: "Spans 2",
			description:     "Colspan content is preserved",
		},
		{
			name: "multiple colspan",
			html: `
				<table>
					<tr><td colspan="3">Spans 3</td></tr>
				</table>
			`,
			expectedContent: "Spans 3",
			description:     "Multiple colspan content is preserved",
		},
		{
			name: "no colspan",
			html: `
				<table>
					<tr><td>A</td><td>B</td><td>C</td></tr>
				</table>
			`,
			expectedContent: "A",
			description:     "All cells are normal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := stdxhtml.Parse(strings.NewReader(tc.html))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			table := FindElementByTag(doc, "table")
			if table == nil {
				t.Fatal("Table not found")
			}

			// Use the public API to extract table content
			var sb strings.Builder
			ExtractTextWithStructureAndImages(table, &sb, 0, nil, "markdown")

			// Verify the expected content is in the output
			result := sb.String()
			if !strings.Contains(result, tc.expectedContent) {
				t.Errorf("%s: expected content %q not found in output %q", tc.description, tc.expectedContent, result)
			}
		})
	}
}
