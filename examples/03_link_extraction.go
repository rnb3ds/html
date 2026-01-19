//go:build examples

package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// LinkExtraction demonstrates comprehensive link extraction from HTML,
// including automatic URL resolution, resource type detection, and filtering.
func main() {
	// Example HTML with various types of links
	htmlContent := `
		<!DOCTYPE html>
		<html>
		<head>
			<base href="https://example.com/">
			<title>Link Extraction Demo</title>
			<link rel="stylesheet" href="css/main.css" title="Main Styles">
			<link rel="icon" href="/favicon.ico">
			<script src="js/app.js"></script>
			<script src="https://cdn.jsdelivr.net/npm/library@1.0.0/dist/lib.min.js"></script>
		</head>
		<body>
			<nav>
				<a href="/" title="Homepage">Home</a>
				<a href="/about">About Us</a>
				<a href="https://blog.example.com" title="Our Blog">Blog</a>
			</nav>

			<main>
				<article>
					<h1>Article with Media</h1>
					<img src="images/hero.jpg" alt="Hero Image" title="Main Hero">
					<p>Check out this <a href="related-article.html">related article</a>.</p>

					<video src="videos/demo.mp4" type="video/mp4"></video>
					<audio src="audio/podcast.mp3" type="audio/mpeg"></audio>
				</article>
			</main>

			<footer>
				<a href="mailto:contact@example.com">Contact</a>
				<a href="https://twitter.com/example" rel="nofollow">Twitter</a>
			</footer>
		</body>
		</html>
	`

	fmt.Println("=== Link Extraction Example ===\n ")

	// Example 1: Simple extraction with defaults (recommended)
	fmt.Println("1. Simple extraction with defaults:")
	links, err := html.ExtractAllLinks(htmlContent)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Total links: %d\n", len(links))
	printLinksByType(links)

	// Example 2: Group links by type for easy filtering
	fmt.Println("\n2. Group links by type:")
	groupedLinks := html.GroupLinksByType(links)

	if cssLinks, exists := groupedLinks["css"]; exists {
		fmt.Printf("   CSS files (%d): ", len(cssLinks))
		for i, link := range cssLinks {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(link.Title)
		}
		fmt.Println()
	}

	if jsLinks, exists := groupedLinks["js"]; exists {
		fmt.Printf("   JavaScript files (%d): ", len(jsLinks))
		for i, link := range jsLinks {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(link.Title)
		}
		fmt.Println()
	}

	// Example 3: Selective extraction (only specific resource types)
	fmt.Println("\n3. Selective extraction (CSS and JS only):")
	processor := html.NewWithDefaults()
	defer processor.Close()

	selectiveConfig := html.LinkExtractionConfig{
		ResolveRelativeURLs:  true,
		IncludeImages:        false,
		IncludeVideos:        false,
		IncludeAudios:        false,
		IncludeCSS:           true,
		IncludeJS:            true,
		IncludeContentLinks:  false,
		IncludeExternalLinks: false,
		IncludeIcons:         false,
	}

	selectiveLinks, err := processor.ExtractAllLinks(htmlContent, selectiveConfig)
	if err != nil {
		log.Fatal(err)
	}

	for _, link := range selectiveLinks {
		fmt.Printf("   %s: %s\n", link.Type, link.URL)
	}

	// Example 4: Custom base URL resolution
	fmt.Println("\n4. Custom base URL resolution:")
	relativeHTML := `
		<html>
		<head>
			<link rel="stylesheet" href="styles.css">
		</head>
		<body>
			<a href="page.html">Relative Link</a>
			<a href="/root-relative.html">Root Relative</a>
			<img src="image.jpg" alt="Image">
		</body>
		</html>
	`

	customConfig := html.LinkExtractionConfig{
		ResolveRelativeURLs: true,
		BaseURL:             "https://custom-domain.com/section/",
		IncludeImages:       true,
		IncludeContentLinks: true,
		IncludeCSS:          true,
	}

	customLinks, err := processor.ExtractAllLinks(relativeHTML, customConfig)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   Resolved URLs:")
	for _, link := range customLinks {
		fmt.Printf("   %s: %s\n", link.Type, link.URL)
	}

	// Example 5: Manual base URL for CDN scenarios
	fmt.Println("\n5. Manual base URL (CDN scenario):")
	cdnHTML := `
		<html>
		<head>
			<link rel="stylesheet" href="https://cdn.example.com/bootstrap.css">
			<script src="https://cdn.jsdelivr.net/npm/jquery.js"></script>
		</head>
		<body>
			<a href="/products">Products</a>
			<img src="images/banner.jpg" alt="Banner">
		</body>
		</html>
	`

	// Without manual base URL (maybe inaccurate)
	autoLinks, _ := html.ExtractAllLinks(cdnHTML)
	fmt.Printf("   Auto-detected: %d links\n", len(autoLinks))

	// With manual base URL (accurate)
	manualConfig := html.LinkExtractionConfig{
		ResolveRelativeURLs: true,
		BaseURL:             "https://mycompany.com/",
		IncludeImages:       true,
		IncludeContentLinks: true,
		IncludeCSS:          true,
		IncludeJS:           true,
	}
	manualLinks, _ := html.ExtractAllLinks(cdnHTML, manualConfig)
	fmt.Printf("   Manual base URL: %d links\n", len(manualLinks))
	for _, link := range manualLinks {
		if link.Type == "link" || link.Type == "image" {
			fmt.Printf("   %s: %s\n", link.Type, link.URL)
		}
	}

	fmt.Println("\n✓ Link extraction complete!")
}

// printLinksByType displays links grouped by resource type
func printLinksByType(links []html.LinkResource) {
	linksByType := html.GroupLinksByType(links)
	typeOrder := []string{"css", "js", "icon", "image", "video", "audio", "link"}

	for _, linkType := range typeOrder {
		if typeLinks, exists := linksByType[linkType]; exists {
			fmt.Printf("   %s (%d):\n", linkType, len(typeLinks))
			for _, link := range typeLinks {
				fmt.Printf("     • %s\n", link.URL)
			}
		}
	}
}
