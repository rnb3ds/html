package html_test

import (
	"strings"
	"testing"

	"github.com/cybergodev/html"
)

func TestExtractAllLinks_Convenience(t *testing.T) {
	t.Parallel()

	htmlContent := `
	<html>
	<head>
		<link rel="stylesheet" href="styles.css">
		<script src="app.js"></script>
	</head>
	<body>
		<a href="https://example.com">External Link</a>
		<a href="/internal">Internal Link</a>
		<img src="image.jpg" alt="Test Image">
		<video src="video.mp4"></video>
	</body>
	</html>
	`

	links, err := html.ExtractAllLinks(htmlContent)
	if err != nil {
		t.Fatalf("ExtractAllLinks() failed: %v", err)
	}

	if len(links) == 0 {
		t.Fatal("ExtractAllLinks() returned no links")
	}

	// Verify we have different types of links
	types := make(map[string]bool)
	for _, link := range links {
		types[link.Type] = true
	}

	expectedTypes := []string{"css", "js", "link", "image", "video"}
	for _, expectedType := range expectedTypes {
		if !types[expectedType] {
			t.Errorf("Expected link type %q not found", expectedType)
		}
	}
}

func TestExtractAllLinks_WithManualBaseURL_Convenience(t *testing.T) {
	t.Parallel()

	htmlContent := `
	<html>
	<head>
		<!-- CDN resources that would mislead auto-detection -->
		<link rel="stylesheet" href="https://cdn.example.com/styles.css">
		<script src="https://cdn.jsdelivr.net/npm/lib.js"></script>
	</head>
	<body>
		<!-- Relative links that need manual base URL -->
		<a href="/about">About</a>
		<a href="contact.html">Contact</a>
		<img src="images/logo.jpg" alt="Logo">
		<video src="videos/intro.mp4"></video>
	</body>
	</html>
	`

	// Manual base URL specification using config parameter
	config := html.DefaultLinkExtractionConfig()
	config.BaseURL = "https://mysite.com/"
	links, err := html.ExtractAllLinks(htmlContent, config)
	if err != nil {
		t.Fatalf("ExtractAllLinks() with manual base URL failed: %v", err)
	}

	if len(links) == 0 {
		t.Fatal("ExtractAllLinks() with manual base URL returned no links")
	}

	// Verify that relative links are resolved with the specified base URL
	expectedResolutions := map[string]string{
		"https://mysite.com/about":            "link",
		"https://mysite.com/contact.html":     "link",
		"https://mysite.com/images/logo.jpg":  "image",
		"https://mysite.com/videos/intro.mp4": "video",
	}

	foundResolutions := make(map[string]string)
	for _, link := range links {
		foundResolutions[link.URL] = link.Type
	}

	for expectedURL, expectedType := range expectedResolutions {
		if foundType, exists := foundResolutions[expectedURL]; !exists {
			t.Errorf("Expected resolved URL %q not found", expectedURL)
		} else if foundType != expectedType {
			t.Errorf("URL %q has type %q, expected %q", expectedURL, foundType, expectedType)
		}
	}

	// Verify CDN links are preserved as-is
	cdnLinks := []string{
		"https://cdn.example.com/styles.css",
		"https://cdn.jsdelivr.net/npm/lib.js",
	}

	for _, cdnURL := range cdnLinks {
		if _, exists := foundResolutions[cdnURL]; !exists {
			t.Errorf("Expected CDN URL %q to be preserved", cdnURL)
		}
	}
}

func TestProcessor_ExtractAllLinks(t *testing.T) {
	t.Parallel()

	processor := html.NewWithDefaults()
	defer processor.Close()

	t.Run("comprehensive link extraction", func(t *testing.T) {
		htmlContent := `
		<!DOCTYPE html>
		<html>
		<head>
			<base href="https://example.com/">
			<title>Test Page</title>
			<link rel="stylesheet" href="css/main.css" title="Main Styles">
			<link rel="icon" href="/favicon.ico">
			<link rel="canonical" href="https://example.com/page">
			<script src="js/app.js"></script>
			<script src="https://cdn.example.com/lib.js"></script>
		</head>
		<body>
			<nav>
				<a href="/" title="Home">Home</a>
				<a href="/about">About</a>
				<a href="https://external.com" title="External Site">External</a>
			</nav>
			<article>
				<h1>Article Title</h1>
				<img src="images/hero.jpg" alt="Hero Image" title="Main Hero">
				<p>Content with <a href="related.html">related link</a></p>
				<video src="videos/demo.mp4" title="Demo Video"></video>
				<audio src="audio/music.mp3"></audio>
				<iframe src="https://youtube.com/embed/abc123" title="YouTube Video"></iframe>
			</article>
		</body>
		</html>
		`

		config := html.DefaultLinkExtractionConfig()
		links, err := processor.ExtractAllLinks(htmlContent, config)
		if err != nil {
			t.Fatalf("ExtractAllLinks() failed: %v", err)
		}

		if len(links) == 0 {
			t.Fatal("ExtractAllLinks() returned no links")
		}

		// Create maps for easier testing
		linksByType := make(map[string][]html.LinkResource)
		linksByURL := make(map[string]html.LinkResource)

		for _, link := range links {
			linksByType[link.Type] = append(linksByType[link.Type], link)
			linksByURL[link.URL] = link
		}

		// Test CSS links
		if len(linksByType["css"]) == 0 {
			t.Error("Expected CSS links not found")
		} else {
			cssLink := linksByURL["https://example.com/css/main.css"]
			if cssLink.Title != "Main Styles" {
				t.Errorf("CSS link title = %q, want %q", cssLink.Title, "Main Styles")
			}
		}

		// Test JavaScript links
		if len(linksByType["js"]) < 2 {
			t.Error("Expected at least 2 JS links")
		}

		// Test icon links
		if len(linksByType["icon"]) == 0 {
			t.Error("Expected icon links not found")
		}

		// Test content links (now includes all a tags regardless of domain)
		if len(linksByType["link"]) == 0 {
			t.Error("Expected content links not found")
		}

		// Test image links
		if len(linksByType["image"]) == 0 {
			t.Error("Expected image links not found")
		} else {
			imgLink := linksByURL["https://example.com/images/hero.jpg"]
			if imgLink.Title != "Main Hero" {
				t.Errorf("Image link title = %q, want %q", imgLink.Title, "Main Hero")
			}
		}

		// Test video links
		if len(linksByType["video"]) < 2 {
			t.Error("Expected at least 2 video links (direct + embed)")
		}

		// Test audio links
		if len(linksByType["audio"]) == 0 {
			t.Error("Expected audio links not found")
		}

		// Verify URL resolution
		for _, link := range links {
			if !strings.HasPrefix(link.URL, "http") && !strings.HasPrefix(link.URL, "//") {
				t.Errorf("Link URL not resolved: %q", link.URL)
			}
		}
	})

	t.Run("relative URL resolution", func(t *testing.T) {
		htmlContent := `
		<html>
		<head>
			<link rel="stylesheet" href="styles.css">
		</head>
		<body>
			<a href="/page">Root relative</a>
			<a href="page.html">Relative</a>
			<a href="../parent.html">Parent relative</a>
			<img src="image.jpg">
		</body>
		</html>
		`

		config := html.LinkExtractionConfig{
			ResolveRelativeURLs:  true,
			BaseURL:              "https://example.com/section/",
			IncludeImages:        true,
			IncludeCSS:           true,
			IncludeContentLinks:  true,
			IncludeExternalLinks: true,
			IncludeJS:            true,
			IncludeVideos:        true,
			IncludeAudios:        true,
			IncludeIcons:         true,
		}

		links, err := processor.ExtractAllLinks(htmlContent, config)
		if err != nil {
			t.Fatalf("ExtractAllLinks() failed: %v", err)
		}

		expectedURLs := map[string]bool{
			"https://example.com/section/styles.css":     true,
			"https://example.com/page":                   true,
			"https://example.com/section/page.html":      true,
			"https://example.com/section/../parent.html": true,
			"https://example.com/section/image.jpg":      true,
		}

		for _, link := range links {
			if !expectedURLs[link.URL] {
				t.Errorf("Unexpected resolved URL: %q", link.URL)
			}
		}
	})

	t.Run("selective extraction", func(t *testing.T) {
		htmlContent := `
		<html>
		<head>
			<link rel="stylesheet" href="styles.css">
			<script src="app.js"></script>
		</head>
		<body>
			<a href="page.html">Link</a>
			<img src="image.jpg">
			<video src="video.mp4"></video>
		</body>
		</html>
		`

		config := html.LinkExtractionConfig{
			ResolveRelativeURLs:  false,
			IncludeImages:        true,
			IncludeVideos:        false, // Disable videos
			IncludeCSS:           false, // Disable CSS
			IncludeJS:            true,
			IncludeContentLinks:  true,
			IncludeExternalLinks: true,
			IncludeIcons:         true,
		}

		links, err := processor.ExtractAllLinks(htmlContent, config)
		if err != nil {
			t.Fatalf("ExtractAllLinks() failed: %v", err)
		}

		// Check that disabled types are not included
		for _, link := range links {
			if link.Type == "video" {
				t.Error("Video links should be disabled")
			}
			if link.Type == "css" {
				t.Error("CSS links should be disabled")
			}
		}

		// Check that enabled types are included
		hasJS := false
		hasImage := false
		hasLink := false
		for _, link := range links {
			switch link.Type {
			case "js":
				hasJS = true
			case "image":
				hasImage = true
			case "link":
				hasLink = true
			}
		}

		if !hasJS {
			t.Error("JS links should be included")
		}
		if !hasImage {
			t.Error("Image links should be included")
		}
		if !hasLink {
			t.Error("Content links should be included")
		}
	})

	t.Run("deduplication", func(t *testing.T) {
		htmlContent := `
		<html>
		<body>
			<img src="image.jpg">
			<img src="image.jpg" alt="Same image">
			<a href="page.html">Link 1</a>
			<a href="page.html">Link 2</a>
		</body>
		</html>
		`

		config := html.DefaultLinkExtractionConfig()
		links, err := processor.ExtractAllLinks(htmlContent, config)
		if err != nil {
			t.Fatalf("ExtractAllLinks() failed: %v", err)
		}

		// Count occurrences of each URL
		urlCounts := make(map[string]int)
		for _, link := range links {
			urlCounts[link.URL]++
		}

		for url, count := range urlCounts {
			if count > 1 {
				t.Errorf("URL %q appears %d times, should be deduplicated", url, count)
			}
		}
	})

	t.Run("empty input", func(t *testing.T) {
		links, err := processor.ExtractAllLinks("", html.DefaultLinkExtractionConfig())
		if err != nil {
			t.Fatalf("ExtractAllLinks() failed on empty input: %v", err)
		}

		if len(links) != 0 {
			t.Errorf("ExtractAllLinks() returned %d links for empty input, want 0", len(links))
		}
	})

	t.Run("invalid HTML", func(t *testing.T) {
		invalidHTML := `<html><body><img src="test.jpg"<p>Invalid</body>`

		// Should still work with invalid HTML due to Go's lenient parser
		links, err := processor.ExtractAllLinks(invalidHTML, html.DefaultLinkExtractionConfig())
		if err != nil {
			t.Fatalf("ExtractAllLinks() failed on invalid HTML: %v", err)
		}

		// Should still extract the image
		if len(links) == 0 {
			t.Error("ExtractAllLinks() should extract links even from invalid HTML")
		}
	})
}

func TestProcessor_ExtractAllLinks_EdgeCases(t *testing.T) {
	t.Parallel()

	processor := html.NewWithDefaults()
	defer processor.Close()

	t.Run("base URL detection", func(t *testing.T) {
		testCases := []struct {
			name     string
			html     string
			expected string
		}{
			{
				name:     "base tag",
				html:     `<html><head><base href="https://example.com/"></head><body><a href="page.html">Link</a></body></html>`,
				expected: "https://example.com/page.html",
			},
			{
				name:     "canonical meta",
				html:     `<html><head><meta property="og:url" content="https://example.com/page"></head><body><a href="relative.html">Link</a></body></html>`,
				expected: "https://example.com/relative.html",
			},
			{
				name:     "absolute URL extraction",
				html:     `<html><body><img src="https://cdn.example.com/image.jpg"><a href="page.html">Link</a></body></html>`,
				expected: "https://cdn.example.com/page.html",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := html.DefaultLinkExtractionConfig()
				config.IncludeContentLinks = true

				links, err := processor.ExtractAllLinks(tc.html, config)
				if err != nil {
					t.Fatalf("ExtractAllLinks() failed: %v", err)
				}

				found := false
				for _, link := range links {
					if link.Type == "link" && link.URL == tc.expected {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected resolved URL %q not found in links", tc.expected)
				}
			})
		}
	})

	t.Run("complex media elements", func(t *testing.T) {
		htmlContent := `
		<html>
		<body>
			<video>
				<source src="video.webm" type="video/webm">
				<source src="video.mp4" type="video/mp4">
			</video>
			<audio>
				<source src="audio.ogg" type="audio/ogg">
				<source src="audio.mp3" type="audio/mpeg">
			</audio>
		</body>
		</html>
		`

		config := html.DefaultLinkExtractionConfig()
		links, err := processor.ExtractAllLinks(htmlContent, config)
		if err != nil {
			t.Fatalf("ExtractAllLinks() failed: %v", err)
		}

		videoCount := 0
		audioCount := 0
		for _, link := range links {
			switch link.Type {
			case "video":
				videoCount++
			case "audio":
				audioCount++
			}
		}

		if videoCount < 2 {
			t.Errorf("Expected at least 2 video sources, got %d", videoCount)
		}
		if audioCount < 2 {
			t.Errorf("Expected at least 2 audio sources, got %d", audioCount)
		}
	})

	t.Run("preload and prefetch links", func(t *testing.T) {
		htmlContent := `
		<html>
		<head>
			<link rel="preload" href="font.woff2" as="font">
			<link rel="preload" href="critical.css" as="style">
			<link rel="preload" href="hero.jpg" as="image">
			<link rel="prefetch" href="next-page.html">
			<link rel="dns-prefetch" href="//cdn.example.com">
		</head>
		</html>
		`

		config := html.DefaultLinkExtractionConfig()
		links, err := processor.ExtractAllLinks(htmlContent, config)
		if err != nil {
			t.Fatalf("ExtractAllLinks() failed: %v", err)
		}

		typeCount := make(map[string]int)
		for _, link := range links {
			typeCount[link.Type]++
		}

		if typeCount["css"] == 0 {
			t.Error("Expected preloaded CSS not found")
		}
		if typeCount["image"] == 0 {
			t.Error("Expected preloaded image not found")
		}
	})
}

func TestDefaultLinkExtractionConfig(t *testing.T) {
	t.Parallel()

	config := html.DefaultLinkExtractionConfig()

	// Verify all defaults are enabled
	if !config.ResolveRelativeURLs {
		t.Error("ResolveRelativeURLs should be true by default")
	}
	if !config.IncludeImages {
		t.Error("IncludeImages should be true by default")
	}
	if !config.IncludeVideos {
		t.Error("IncludeVideos should be true by default")
	}
	if !config.IncludeAudios {
		t.Error("IncludeAudios should be true by default")
	}
	if !config.IncludeCSS {
		t.Error("IncludeCSS should be true by default")
	}
	if !config.IncludeJS {
		t.Error("IncludeJS should be true by default")
	}
	if !config.IncludeContentLinks {
		t.Error("IncludeContentLinks should be true by default")
	}
	if !config.IncludeExternalLinks {
		t.Error("IncludeExternalLinks should be true by default")
	}
	if !config.IncludeIcons {
		t.Error("IncludeIcons should be true by default")
	}
	if config.BaseURL != "" {
		t.Error("BaseURL should be empty by default")
	}
}

func TestLinkResource(t *testing.T) {
	t.Parallel()

	// Test LinkResource struct
	link := html.LinkResource{
		URL:   "https://example.com/test.css",
		Title: "Test Stylesheet",
		Type:  "css",
	}

	if link.URL != "https://example.com/test.css" {
		t.Errorf("URL = %q, want %q", link.URL, "https://example.com/test.css")
	}
	if link.Title != "Test Stylesheet" {
		t.Errorf("Title = %q, want %q", link.Title, "Test Stylesheet")
	}
	if link.Type != "css" {
		t.Errorf("Type = %q, want %q", link.Type, "css")
	}
}
func TestGroupLinksByType(t *testing.T) {
	t.Parallel()

	t.Run("group mixed links", func(t *testing.T) {
		links := []html.LinkResource{
			{URL: "https://example.com/style.css", Title: "Main CSS", Type: "css"},
			{URL: "https://example.com/app.js", Title: "App Script", Type: "js"},
			{URL: "https://example.com/about", Title: "About Page", Type: "link"},
			{URL: "https://example.com/image.jpg", Title: "Hero Image", Type: "image"},
			{URL: "https://example.com/video.mp4", Title: "Demo Video", Type: "video"},
			{URL: "https://example.com/another.css", Title: "Secondary CSS", Type: "css"},
			{URL: "https://example.com/contact", Title: "Contact Page", Type: "link"},
		}

		grouped := html.GroupLinksByType(links)

		// Verify CSS links
		if len(grouped["css"]) != 2 {
			t.Errorf("Expected 2 CSS links, got %d", len(grouped["css"]))
		}
		if grouped["css"][0].Title != "Main CSS" && grouped["css"][1].Title != "Main CSS" {
			if grouped["css"][0].Title != "Secondary CSS" && grouped["css"][1].Title != "Secondary CSS" {
				t.Error("CSS links not grouped correctly")
			}
		}

		// Verify JS links
		if len(grouped["js"]) != 1 {
			t.Errorf("Expected 1 JS link, got %d", len(grouped["js"]))
		}
		if grouped["js"][0].Title != "App Script" {
			t.Errorf("JS link title = %q, want %q", grouped["js"][0].Title, "App Script")
		}

		// Verify content links
		if len(grouped["link"]) != 2 {
			t.Errorf("Expected 2 content links, got %d", len(grouped["link"]))
		}

		// Verify image links
		if len(grouped["image"]) != 1 {
			t.Errorf("Expected 1 image link, got %d", len(grouped["image"]))
		}

		// Verify video links
		if len(grouped["video"]) != 1 {
			t.Errorf("Expected 1 video link, got %d", len(grouped["video"]))
		}

		// Verify no audio links
		if len(grouped["audio"]) != 0 {
			t.Errorf("Expected 0 audio links, got %d", len(grouped["audio"]))
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		links := []html.LinkResource{}
		grouped := html.GroupLinksByType(links)

		if len(grouped) != 0 {
			t.Errorf("Expected empty map, got %d entries", len(grouped))
		}
	})

	t.Run("nil slice", func(t *testing.T) {
		var links []html.LinkResource
		grouped := html.GroupLinksByType(links)

		if len(grouped) != 0 {
			t.Errorf("Expected empty map, got %d entries", len(grouped))
		}
	})

	t.Run("links with empty type", func(t *testing.T) {
		links := []html.LinkResource{
			{URL: "https://example.com/valid.css", Title: "Valid CSS", Type: "css"},
			{URL: "https://example.com/invalid", Title: "No Type", Type: ""},
			{URL: "https://example.com/page.html", Title: "Page", Type: "link"},
		}

		grouped := html.GroupLinksByType(links)

		// Should only have css and link types
		if len(grouped) != 3 {
			t.Errorf("Expected 3 types, got %d", len(grouped))
		}

		if len(grouped["css"]) != 1 {
			t.Errorf("Expected 1 CSS link, got %d", len(grouped["css"]))
		}

		if len(grouped["link"]) != 1 {
			t.Errorf("Expected 1 content link, got %d", len(grouped["link"]))
		}

		// Empty type should not be included
		if _, exists := grouped[""]; exists {
			t.Error("Empty type should not be included in grouped results")
		}
	})

	t.Run("integration with ExtractAllLinks", func(t *testing.T) {
		htmlContent := `
		<html>
		<head>
			<link rel="stylesheet" href="https://example.com/style.css">
			<script src="https://example.com/app.js"></script>
		</head>
		<body>
			<a href="https://example.com/about">About</a>
			<img src="https://example.com/image.jpg" alt="Image">
		</body>
		</html>
		`

		links, err := html.ExtractAllLinks(htmlContent)
		if err != nil {
			t.Fatalf("ExtractAllLinks failed: %v", err)
		}

		grouped := html.GroupLinksByType(links)

		// Should have at least css, js, link, and image types
		expectedTypes := []string{"css", "js", "link", "image"}
		for _, expectedType := range expectedTypes {
			if len(grouped[expectedType]) == 0 {
				t.Errorf("Expected %s links, but got none", expectedType)
			}
		}
	})
}
