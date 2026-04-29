package html_test

import (
	"testing"

	"github.com/cybergodev/html"
)

func TestGroupLinksByType(t *testing.T) {
	t.Parallel()

	t.Run("empty slice returns empty map", func(t *testing.T) {
		result := html.GroupLinksByType(nil)
		if len(result) != 0 {
			t.Errorf("expected empty map, got %d entries", len(result))
		}
		result = html.GroupLinksByType([]html.LinkResource{})
		if len(result) != 0 {
			t.Errorf("expected empty map, got %d entries", len(result))
		}
	})

	t.Run("groups links by type", func(t *testing.T) {
		links := []html.LinkResource{
			{URL: "https://a.com", Type: "stylesheet", Title: "CSS"},
			{URL: "https://b.com", Type: "stylesheet", Title: "CSS2"},
			{URL: "https://c.com", Type: "canonical", Title: "Canon"},
			{URL: "https://d.com", Type: "icon", Title: "Favicon"},
		}
		grouped := html.GroupLinksByType(links)
		if len(grouped) != 3 {
			t.Fatalf("expected 3 groups, got %d", len(grouped))
		}
		if len(grouped["stylesheet"]) != 2 {
			t.Errorf("expected 2 stylesheet links, got %d", len(grouped["stylesheet"]))
		}
		if len(grouped["canonical"]) != 1 {
			t.Errorf("expected 1 canonical link, got %d", len(grouped["canonical"]))
		}
		if len(grouped["icon"]) != 1 {
			t.Errorf("expected 1 icon link, got %d", len(grouped["icon"]))
		}
	})

	t.Run("links with empty type grouped as unknown", func(t *testing.T) {
		links := []html.LinkResource{
			{URL: "https://a.com", Type: "", Title: "No type"},
			{URL: "https://b.com", Type: "", Title: "Also no type"},
			{URL: "https://c.com", Type: "stylesheet", Title: "CSS"},
		}
		grouped := html.GroupLinksByType(links)
		if len(grouped) != 2 {
			t.Fatalf("expected 2 groups, got %d", len(grouped))
		}
		if len(grouped["unknown"]) != 2 {
			t.Errorf("expected 2 unknown links, got %d", len(grouped["unknown"]))
		}
	})

	t.Run("groups from extracted links", func(t *testing.T) {
		htmlContent := []byte(`<html><head>
			<link rel="stylesheet" href="https://example.com/style.css">
			<link rel="icon" href="https://example.com/favicon.ico">
			<link rel="canonical" href="https://example.com/page">
		</head><body>
			<a href="https://example.com/link1">Link 1</a>
			<a href="https://example.com/link2">Link 2</a>
		</body></html>`)

		links, err := html.ExtractAllLinks(htmlContent)
		if err != nil {
			t.Fatalf("ExtractAllLinks() failed: %v", err)
		}

		grouped := html.GroupLinksByType(links)

		if len(grouped) == 0 {
			t.Fatal("expected non-empty grouped result")
		}

		for typ, group := range grouped {
			if typ == "" {
				t.Error("group type should never be empty (empty types become 'unknown')")
			}
			for _, link := range group {
				if link.URL == "" {
					t.Error("grouped link should have a URL")
				}
			}
		}
	})
}
