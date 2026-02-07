//go:build examples

package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/cybergodev/html"
)

// This example demonstrates comprehensive link extraction and URL handling.
// Learn how to extract, categorize, and resolve URLs from web pages.
func main() {
	fmt.Println("=== Link Extraction & URL Resolution ===\n ")

	htmlContent := `
		<html>
			<head>
				<base href="https://example.com/blog/">
				<link rel="stylesheet" href="/assets/style.css">
				<link rel="icon" href="/favicon.ico">
			</head>
			<body>
				<h1>Go Programming Resources</h1>

				<nav>
					<a href="/">Home</a>
					<a href="/about">About</a>
					<a href="https://golang.org">Official Site</a>
					<a href="//cdn.com/lib.js">CDN Library</a>
				</nav>

				<article>
					<h2>Learning Resources</h2>
					<ul>
						<li><a href="https://go.dev/tour/">Interactive Tour</a></li>
						<li><a href="/docs/effective-go.html">Effective Go</a></li>
						<li><a href="basics.html">Go Basics</a></li>
						<li><a href="../advanced/">Advanced Topics</a></li>
					</ul>

					<h3>External Resources</h3>
					<p>Check out these great tutorials:</p>
					<ul>
						<li><a href="https://github.com/golang/go">Go on GitHub</a></li>
						<li><a href="https://pkg.go.dev/">Package Documentation</a></li>
					</ul>

					<h3>Media Files</h3>
					<img src="images/diagram.png" alt="Diagram">
					<a href="downloads/tutorial.pdf">Download Tutorial</a>
					<a href="https://youtube.com/watch?v=example">Video Tutorial</a>
				</article>

				<footer>
					<a href="mailto:contact@example.com">Contact</a>
					<a href="/sitemap.xml">Sitemap</a>
				</footer>
			</body>
		</html>
	`

	processor, _ := html.New()
	defer processor.Close()

	// ============================================================
	// PART 1: Extract all links with automatic URL resolution
	// ============================================================
	fmt.Println("Part 1: Extract All Links (with URL resolution)")
	fmt.Println("----------------------------------------------")

	config := html.DefaultLinkExtractionConfig()
	config.ResolveRelativeURLs = true // Convert relative URLs to absolute

	result, err := processor.Extract([]byte(htmlContent), html.DefaultExtractConfig())
	if err != nil {
		log.Fatal(err)
	}

	links, err := html.ExtractAllLinks([]byte(htmlContent), config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d links:\n\n", len(links))

	// Categorize links by type
	externalLinks := 0
	relativeLinks := 0
	contentLinks := 0
	mediaLinks := 0
	otherLinks := 0

	for _, link := range links {
		switch {
		case link.Type == "content":
			contentLinks++
			fmt.Printf("Content:   %s\n", link.URL)
		case link.Type == "stylesheet":
			otherLinks++
			fmt.Printf("Style:     %s\n", link.URL)
		case link.Type == "icon":
			otherLinks++
			fmt.Printf("Icon:      %s\n", link.URL)
		case link.Type == "script":
			otherLinks++
			fmt.Printf("Script:    %s\n", link.URL)
		case link.Type == "video" || link.Type == "audio":
			mediaLinks++
			fmt.Printf("Media:     %s (%s)\n", link.URL, link.Type)
		default:
			// Check if URL is external by checking for http(s) protocol
			if strings.HasPrefix(link.URL, "http") && !strings.Contains(link.URL, "example.com") {
				externalLinks++
				fmt.Printf("External:  %s\n", link.URL)
			} else {
				relativeLinks++
				fmt.Printf("Internal:  %s\n", link.URL)
			}
		}
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  External links: %d\n", externalLinks)
	fmt.Printf("  Relative links: %d\n", relativeLinks)
	fmt.Printf("  Content links: %d\n", contentLinks)
	fmt.Printf("  Media links: %d\n", mediaLinks)
	fmt.Printf("  Other (CSS/JS): %d\n", otherLinks)

	// ============================================================
	// PART 2: URL resolution details
	// ============================================================
	fmt.Println("\n\nPart 2: URL Resolution Details")
	fmt.Println("---------------------------")

	// Show how relative URLs are resolved
	fmt.Println("Base URL: https://example.com/blog/\n")

	resolutionExamples := []struct {
		original string
		resolved string
		explain  string
	}{
		{
			original: "/about",
			resolved: "https://example.com/about",
			explain:  "Root-relative → protocol + domain + path",
		},
		{
			original: "basics.html",
			resolved: "https://example.com/blog/basics.html",
			explain:  "Relative → base URL + path",
		},
		{
			original: "../advanced/",
			resolved: "https://example.com/advanced/",
			explain:  "Parent directory → go up one level",
		},
		{
			original: "//cdn.com/lib.js",
			resolved: "https://cdn.com/lib.js",
			explain:  "Protocol-relative → use page's protocol",
		},
		{
			original: "https://golang.org",
			resolved: "https://golang.org",
			explain:  "Absolute URL → unchanged",
		},
	}

	for _, ex := range resolutionExamples {
		fmt.Printf("  %s\n", ex.original)
		fmt.Printf("    → %s\n", ex.resolved)
		fmt.Printf("    (%s)\n\n", ex.explain)
	}

	// ============================================================
	// PART 3: Filter links by type
	// ============================================================
	fmt.Println("Part 3: Filter Links by Type")
	fmt.Println("---------------------------")

	// Extract only content links (CSS, JS, etc.)
	config2 := html.DefaultLinkExtractionConfig()
	config2.IncludeCSS = false
	config2.IncludeJS = false
	config2.IncludeIcons = false

	contentOnlyLinks, err := html.ExtractAllLinks([]byte(htmlContent), config2)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Content links only (CSS/JS/icons filtered out):\n")
	for _, link := range contentOnlyLinks {
		fmt.Printf("  • %s\n", link.URL)
	}

	// ============================================================
	// PART 4: Extract images separately
	// ============================================================
	fmt.Println("\n\nPart 4: Image Extraction with Metadata")
	fmt.Println("------------------------------------")

	result, err = processor.Extract([]byte(htmlContent), html.DefaultExtractConfig())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d images:\n\n", len(result.Images))

	for i, img := range result.Images {
		fmt.Printf("Image %d:\n", i+1)
		fmt.Printf("  URL: %s\n", img.URL)
		fmt.Printf("  Alt: %q\n", img.Alt)
		fmt.Printf("  Width: %s, Height: %s\n", img.Width, img.Height)
		fmt.Printf("  Is Decorative: %v\n", img.IsDecorative)
		fmt.Println()
	}

	// ============================================================
	// PART 5: Link extraction best practices
	// ============================================================
	fmt.Println("Part 5: Best Practices for Link Extraction")
	fmt.Println("----------------------------------------")

	fmt.Println("✓ Use ResolveRelativeURLs for complete URLs")
	fmt.Println("  • Makes URLs ready for HTTP requests")
	fmt.Println("  • Easier to store and process later")
	fmt.Println()

	fmt.Println("✓ Filter by type to get what you need")
	fmt.Println("  • Set IncludeImages=false to skip image links")
	fmt.Println("  • Set IncludeCSS=false to ignore stylesheets")
	fmt.Println("  • Set IncludeContentLinks=false for nav/sidebar only")
	fmt.Println()

	fmt.Println("✓ Check URL prefix to identify external links")
	fmt.Println("  • Look for http(s) URLs not containing your domain")
	fmt.Println("  • Useful for distinguishing internal vs external")
	fmt.Println()

	fmt.Println("✓ Use link.Type to understand context")
	fmt.Println("  • 'content' = main content links")
	fmt.Println("  • 'video'/'audio' = media files")
	fmt.Println("  • '' = navigation/utility links")

}
