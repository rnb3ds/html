package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

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
		<link rel="preload" href="fonts/roboto.woff2" as="font">
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
				
				<video controls>
					<source src="videos/demo.webm" type="video/webm">
					<source src="videos/demo.mp4" type="video/mp4">
					Your browser doesn't support video.
				</video>
				
				<audio controls>
					<source src="audio/podcast.ogg" type="audio/ogg">
					<source src="audio/podcast.mp3" type="audio/mpeg">
				</audio>
				
				<iframe src="https://www.youtube.com/embed/dQw4w9WgXcQ" 
				        title="YouTube Video" width="560" height="315"></iframe>
			</article>
		</main>
		
		<footer>
			<a href="mailto:contact@example.com">Contact</a>
			<a href="https://twitter.com/example" rel="nofollow">Twitter</a>
		</footer>
	</body>
	</html>
	`

	fmt.Println("=== Comprehensive Link Extraction Demo ===")
	fmt.Println()

	// Method 1: Simple convenience function (recommended for basic usage)
	fmt.Println("1. Using convenience function ExtractAllLinks():")
	links, err := html.ExtractAllLinks(htmlContent)
	if err != nil {
		log.Fatal(err)
	}

	printLinksByType(links)

	// Method 2: Using processor with custom configuration
	fmt.Println("\n2. Using processor with custom configuration:")
	processor := html.NewWithDefaults()
	defer processor.Close()

	config := html.LinkExtractionConfig{
		ResolveRelativeURLs:  true,
		BaseURL:              "", // Auto-detect from HTML
		IncludeImages:        true,
		IncludeVideos:        true,
		IncludeAudios:        true,
		IncludeCSS:           true,
		IncludeJS:            true,
		IncludeContentLinks:  true,
		IncludeExternalLinks: true,
		IncludeIcons:         true,
	}

	links2, err := processor.ExtractAllLinks(htmlContent, config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total links extracted: %d\n", len(links2))
	printLinksByType(links2)

	// Method 3: Selective extraction (only CSS and JS)
	fmt.Println("\n3. Selective extraction (CSS and JS only):")
	selectiveConfig := html.LinkExtractionConfig{
		ResolveRelativeURLs:  true,
		IncludeImages:        false,
		IncludeVideos:        false,
		IncludeAudios:        false,
		IncludeCSS:           true, // Only CSS
		IncludeJS:            true, // Only JS
		IncludeContentLinks:  false,
		IncludeExternalLinks: false,
		IncludeIcons:         false,
	}

	selectiveLinks, err := processor.ExtractAllLinks(htmlContent, selectiveConfig)
	if err != nil {
		log.Fatal(err)
	}

	for _, link := range selectiveLinks {
		fmt.Printf("  %s: %s (%s)\n", link.Type, link.URL, link.Title)
	}

	// Method 3: Demonstrate GroupLinksByType convenience function
	fmt.Println("\n3. Using GroupLinksByType convenience function:")
	allLinks, err := html.ExtractAllLinks(htmlContent)
	if err != nil {
		log.Fatal(err)
	}

	// Group links by type using the convenience function
	groupedLinks := html.GroupLinksByType(allLinks)

	fmt.Printf("Total links extracted: %d, grouped into %d types\n", len(allLinks), len(groupedLinks))

	// Access specific types directly
	if cssLinks, exists := groupedLinks["css"]; exists {
		fmt.Printf("CSS files (%d): ", len(cssLinks))
		for i, link := range cssLinks {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(link.Title)
		}
		fmt.Println()
	}

	if jsLinks, exists := groupedLinks["js"]; exists {
		fmt.Printf("JavaScript files (%d): ", len(jsLinks))
		for i, link := range jsLinks {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(link.Title)
		}
		fmt.Println()
	}

	if contentLinks, exists := groupedLinks["link"]; exists {
		fmt.Printf("Content links (%d): ", len(contentLinks))
		for i, link := range contentLinks {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(link.Title)
		}
		fmt.Println()
	}

	// Method 5: Custom base URL resolution
	fmt.Println("\n5. Custom base URL resolution:")
	customConfig := html.LinkExtractionConfig{
		ResolveRelativeURLs:  true,
		BaseURL:              "https://custom-domain.com/section/",
		IncludeImages:        true,
		IncludeContentLinks:  true,
		IncludeCSS:           true,
		IncludeJS:            false,
		IncludeVideos:        false,
		IncludeAudios:        false,
		IncludeExternalLinks: false,
		IncludeIcons:         false,
	}

	// Simple HTML with relative URLs
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

	customLinks, err := processor.ExtractAllLinks(relativeHTML, customConfig)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("  Resolved URLs:")
	for _, link := range customLinks {
		fmt.Printf("    %s: %s\n", link.Type, link.URL)
	}

	// Demonstrate manual base URL specification for CDN scenarios
	demonstrateManualBaseURL()
}

func printLinksByType(links []html.LinkResource) {
	// Use the new GroupLinksByType convenience function
	linksByType := html.GroupLinksByType(links)

	// Print each type
	typeOrder := []string{"css", "js", "icon", "image", "video", "audio", "link"}

	for _, linkType := range typeOrder {
		if links, exists := linksByType[linkType]; exists {
			fmt.Printf("  %s (%d):\n", linkType, len(links))
			for _, link := range links {
				fmt.Printf("    • %s (%s)\n", link.URL, link.Title)
			}
		}
	}
}

// demonstrateManualBaseURL shows how to use manual base URL specification
func demonstrateManualBaseURL() {
	fmt.Println("\n=== Manual Base URL Specification Demo ===")

	// Example with CDN resources that would mislead auto-detection
	cdnHtmlContent := `
	<html>
	<head>
		<!-- CDN resources -->
		<link rel="stylesheet" href="https://cdn.example.com/bootstrap.css">
		<script src="https://cdn.jsdelivr.net/npm/jquery.js"></script>
	</head>
	<body>
		<!-- Relative links that need proper base URL -->
		<a href="/products">Products</a>
		<a href="services.html">Services</a>
		<img src="images/banner.jpg" alt="Banner">
		<video src="media/promo.mp4"></video>
	</body>
	</html>
	`

	fmt.Println("HTML with CDN resources and relative links:")
	fmt.Println("- CDN CSS: https://cdn.example.com/bootstrap.css")
	fmt.Println("- CDN JS: https://cdn.jsdelivr.net/npm/jquery.js")
	fmt.Println("- Relative links: /products, services.html, images/banner.jpg, media/promo.mp4")

	// Method 1: Auto-detection (may be inaccurate due to CDN)
	fmt.Println("\n1. Using auto-detection (may be inaccurate):")
	autoLinks, err := html.ExtractAllLinks(cdnHtmlContent)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Auto-detected links (%d):\n", len(autoLinks))
	for _, link := range autoLinks {
		fmt.Printf("  %s: %s\n", link.Type, link.URL)
	}

	// Method 2: Manual base URL specification (accurate)
	fmt.Println("\n2. Using manual base URL specification (accurate):")
	manualBaseURL := "https://mycompany.com/"
	manualLinks, err := html.ExtractAllLinks(cdnHtmlContent, manualBaseURL)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Manually resolved links (%d):\n", len(manualLinks))
	for _, link := range manualLinks {
		fmt.Printf("  %s: %s\n", link.Type, link.URL)
	}

	// Verify correct resolution
	fmt.Println("\n3. Verification of correct resolution:")
	expectedResolutions := map[string]bool{
		"https://mycompany.com/products":          false,
		"https://mycompany.com/services.html":     false,
		"https://mycompany.com/images/banner.jpg": false,
		"https://mycompany.com/media/promo.mp4":   false,
	}

	for _, link := range manualLinks {
		if _, exists := expectedResolutions[link.URL]; exists {
			expectedResolutions[link.URL] = true
		}
	}

	for url, found := range expectedResolutions {
		status := "✓"
		if !found {
			status = "✗"
		}
		fmt.Printf("  %s %s\n", status, url)
	}
}
