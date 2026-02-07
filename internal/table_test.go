package internal

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestContainsWord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		text  string
		word  string
		want  bool
	}{
		{
			name: "word at start",
			text: "text-align:left",
			word: "text-align",
			want: true,
		},
		{
			name: "word in middle",
			text: "display:block;text-align:center",
			word: "text-align",
			want: true,
		},
		{
			name: "word at end",
			text: "color:red;text-align:left",
			word: "left",
			want: true,
		},
		{
			name: "partial match should fail",
			text: "textalign:left",
			word: "text-align",
			want: false,
		},
		{
			name: "word without boundary",
			text: "mytext-align:left",
			word: "text",
			want: false,
		},
		{
			name: "hyphen is NOT a word boundary for align",
			text: "text-align:left",
			word: "align",
			want: false, // Hyphen before align means text-align is a single word
		},
		{
			name: "with space boundary",
			text: "display block text",
			word: "text",
			want: true,
		},
		{
			name: "with semicolon boundary",
			text: "display:block;text",
			word: "text",
			want: true,
		},
		{
			name: "with quote boundary",
			text: `"text-align:left"`,
			word: "left",
			want: true,
		},
		{
			name: "empty text",
			text: "",
			word: "test",
			want: false,
		},
		{
			name: "empty word",
			text: "test",
			word: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsWord(tt.text, tt.word)
			if result != tt.want {
				t.Errorf("containsWord(%q, %q) = %v, want %v", tt.text, tt.word, result, tt.want)
			}
		})
	}
}

func TestGetCellAlign(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		html     string
		wantAlign cellAlign
	}{
		{
			name:     "align attribute left",
			html:     `<table><tr><td align="left">Text</td></tr></table>`,
			wantAlign: alignLeft,
		},
		{
			name:     "align attribute center",
			html:     `<table><tr><td align="center">Text</td></tr></table>`,
			wantAlign: alignCenter,
		},
		{
			name:     "align attribute right",
			html:     `<table><tr><td align="right">Text</td></tr></table>`,
			wantAlign: alignRight,
		},
		{
			name:     "align attribute justify",
			html:     `<table><tr><td align="justify">Text</td></tr></table>`,
			wantAlign: alignJustify,
		},
		{
			name:     "style attribute text-align",
			html:     `<table><tr><td style="text-align:center">Text</td></tr></table>`,
			wantAlign: alignCenter,
		},
		{
			name:     "style with colon space",
			html:     `<table><tr><td style="text-align: center">Text</td></tr></table>`,
			wantAlign: alignCenter,
		},
		{
			name:     "align takes precedence over style",
			html:     `<table><tr><td align="left" style="text-align:center">Text</td></tr></table>`,
			wantAlign: alignLeft,
		},
		{
			name:     "no alignment specified",
			html:     `<table><tr><td>Text</td></tr></table>`,
			wantAlign: alignDefault,
		},
		{
			name:     "uppercase style",
			html:     `<table><tr><td style="TEXT-ALIGN:CENTER">Text</td></tr></table>`,
			wantAlign: alignCenter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			var td *html.Node
			WalkNodes(doc, func(n *html.Node) bool {
				if n.Type == html.ElementNode && n.Data == "td" {
					td = n
					return false
				}
				return true
			})

			if td == nil {
				t.Fatal("Could not find td element")
			}

			result := getCellAlign(td)
			if result != tt.wantAlign {
				t.Errorf("getCellAlign() = %v, want %v", result, tt.wantAlign)
			}
		})
	}
}

func TestGetColSpan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		html     string
		wantSpan int
	}{
		{
			name:     "default colspan",
			html:     `<table><tr><td>Text</td></tr></table>`,
			wantSpan: 1,
		},
		{
			name:     "colspan 2",
			html:     `<table><tr><td colspan="2">Text</td></tr></table>`,
			wantSpan: 2,
		},
		{
			name:     "colspan 5",
			html:     `<table><tr><td colspan="5">Text</td></tr></table>`,
			wantSpan: 5,
		},
		{
			name:     "invalid colspan defaults to 1",
			html:     `<table><tr><td colspan="0">Text</td></tr></table>`,
			wantSpan: 1,
		},
		{
			name:     "negative colspan defaults to 1",
			html:     `<table><tr><td colspan="-1">Text</td></tr></table>`,
			wantSpan: 1,
		},
		{
			name:     "non-numeric colspan defaults to 1",
			html:     `<table><tr><td colspan="abc">Text</td></tr></table>`,
			wantSpan: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			var td *html.Node
			WalkNodes(doc, func(n *html.Node) bool {
				if n.Type == html.ElementNode && n.Data == "td" {
					td = n
					return false
				}
				return true
			})

			if td == nil {
				t.Fatal("Could not find td element")
			}

			result := getColSpan(td)
			if result != tt.wantSpan {
				t.Errorf("getColSpan() = %d, want %d", result, tt.wantSpan)
			}
		})
	}
}

func TestGetRowSpan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		html     string
		wantSpan int
	}{
		{
			name:     "default rowspan",
			html:     `<table><tr><td>Text</td></tr></table>`,
			wantSpan: 1,
		},
		{
			name:     "rowspan 2",
			html:     `<table><tr><td rowspan="2">Text</td></tr></table>`,
			wantSpan: 2,
		},
		{
			name:     "rowspan 3",
			html:     `<table><tr><td rowspan="3">Text</td></tr></table>`,
			wantSpan: 3,
		},
		{
			name:     "invalid rowspan defaults to 1",
			html:     `<table><tr><td rowspan="0">Text</td></tr></table>`,
			wantSpan: 1,
		},
		{
			name:     "non-numeric rowspan defaults to 1",
			html:     `<table><tr><td rowspan="abc">Text</td></tr></table>`,
			wantSpan: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			var td *html.Node
			WalkNodes(doc, func(n *html.Node) bool {
				if n.Type == html.ElementNode && n.Data == "td" {
					td = n
					return false
				}
				return true
			})

			if td == nil {
				t.Fatal("Could not find td element")
			}

			result := getRowSpan(td)
			if result != tt.wantSpan {
				t.Errorf("getRowSpan() = %d, want %d", result, tt.wantSpan)
			}
		})
	}
}

func TestGetCellWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		html    string
		want    string
		wantNil bool
	}{
		{
			name:    "width attribute pixels",
			html:    `<table><tr><td width="100">Text</td></tr></table>`,
			want:    "100",
			wantNil: false,
		},
		{
			name:    "width attribute percent",
			html:    `<table><tr><td width="50%">Text</td></tr></table>`,
			want:    "50%",
			wantNil: false,
		},
		{
			name:    "style width pixels",
			html:    `<table><tr><td style="width:200px">Text</td></tr></table>`,
			want:    "200px",
			wantNil: false,
		},
		{
			name:    "style width percent",
			html:    `<table><tr><td style="width:25%">Text</td></tr></table>`,
			want:    "25%",
			wantNil: false,
		},
		{
			name:    "width attribute zero returns empty",
			html:    `<table><tr><td width="0">Text</td></tr></table>`,
			want:    "",
			wantNil: true,
		},
		{
			name:    "no width specified",
			html:    `<table><tr><td>Text</td></tr></table>`,
			want:    "",
			wantNil: true,
		},
		{
			name:    "width attribute takes precedence",
			html:    `<table><tr><td width="100" style="width:200px">Text</td></tr></table>`,
			want:    "100",
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			var td *html.Node
			WalkNodes(doc, func(n *html.Node) bool {
				if n.Type == html.ElementNode && n.Data == "td" {
					td = n
					return false
				}
				return true
			})

			if td == nil {
				t.Fatal("Could not find td element")
			}

			result := getCellWidth(td)
			if result != tt.want {
				t.Errorf("getCellWidth() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestCellAlignValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		align cellAlign
		name  string
	}{
		{alignLeft, "left"},
		{alignCenter, "center"},
		{alignRight, "right"},
		{alignJustify, "justify"},
		{alignDefault, "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the constants are different
			aligns := []cellAlign{alignLeft, alignCenter, alignRight, alignJustify, alignDefault}
			unique := make(map[cellAlign]bool)
			for _, a := range aligns {
				unique[a] = true
			}
			if len(unique) != len(aligns) {
				t.Error("cellAlign constants should be unique")
			}
		})
	}
}
