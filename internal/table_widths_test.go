package internal

import (
	"strings"

	stdxhtml "golang.org/x/net/html"
	"testing"
)

// TestTableColumnWidths tests the table column width collection functionality.
// These functions had 0% coverage and are important for proper table rendering.

func TestCollectColumnWidths(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		html         string
		minLen       int
		description  string
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
			minLen: 3,
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
			minLen: 2,
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
			minLen: 3,
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

			// Extract table data
			tableData, _ := extractTableData(table, "markdown")

			// Verify table data was extracted with expected cell count
			if len(tableData) == 0 {
				t.Error("No table data extracted")
			} else if len(tableData[0]) < tc.minLen {
				t.Errorf("%s: extracted row with %d cells, want at least %d", tc.description, len(tableData[0]), tc.minLen)
			}
		})
	}
}

func TestEnsureColWidthCapacity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		colWidths []string
		index     int
		minLen    int
	}{
		{
			name:      "already has capacity",
			colWidths: []string{"100px", "200px"},
			index:     1,
			minLen:    2,
		},
		{
			name:      "needs to grow",
			colWidths: []string{"100px"},
			index:     2,
			minLen:    3,
		},
		{
			name:      "empty array",
			colWidths: []string{},
			index:     0,
			minLen:    1,
		},
		{
			name:      "large index jump",
			colWidths: []string{},
			index:     10,
			minLen:    11,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ensureColWidthCapacity(tc.colWidths, tc.index)

			if len(result) < tc.minLen {
				t.Errorf("ensureColWidthCapacity() returned array of length %d, want at least %d", len(result), tc.minLen)
			}

			// Verify the array has capacity for the requested index
			if cap(result) <= tc.index {
				t.Errorf("ensureColWidthCapacity() capacity %d is not enough for index %d", cap(result), tc.index)
			}
		})
	}
}

func TestTableStructureRowDetection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		html            string
		expectedRows    int
		structureRows   int
		description     string
	}{
		{
			name: "normal data rows",
			html: `
				<table>
					<tr><td>Data 1</td><td>Data 2</td></tr>
					<tr><td>Data 3</td><td>Data 4</td></tr>
				</table>
			`,
			expectedRows:  2,
			structureRows: 0,
			description:   "All rows are data rows",
		},
		{
			name: "structure row with widths",
			html: `
				<table>
					<tr><td width="100px"></td><td width="200px"></td></tr>
					<tr><td>Data 1</td><td>Data 2</td></tr>
				</table>
			`,
			expectedRows:  1, // Only data row
			structureRows: 1, // Structure row excluded from data
			description:   "Structure row excluded from Markdown output",
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

			tableData, _ := extractTableData(table, "markdown")

			if len(tableData) != tc.expectedRows {
				t.Errorf("%s: extracted %d data rows, want %d", tc.description, len(tableData), tc.expectedRows)
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
		name           string
		html           string
		expectedCells  int
		description    string
	}{
		{
			name: "single colspan",
			html: `
				<table>
					<tr><td colspan="2">Spans 2</td><td>Normal</td></tr>
				</table>
			`,
			expectedCells: 3, // 1 expanded + 1 placeholder + 1 normal
			description:   "Colspan creates placeholder cells",
		},
		{
			name: "multiple colspan",
			html: `
				<table>
					<tr><td colspan="3">Spans 3</td></tr>
				</table>
			`,
			expectedCells: 3, // 1 expanded + 2 placeholders
			description:   "Multiple placeholders created",
		},
		{
			name: "no colspan",
			html: `
				<table>
					<tr><td>A</td><td>B</td><td>C</td></tr>
				</table>
			`,
			expectedCells: 3,
			description:   "All cells are normal",
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

			tableData, _ := extractTableData(table, "markdown")

			if len(tableData) == 0 {
				t.Fatal("No table data extracted")
			}

			row := tableData[0]
			if len(row) != tc.expectedCells {
				t.Errorf("%s: got %d cells, want %d", tc.description, len(row), tc.expectedCells)
			}
		})
	}
}
