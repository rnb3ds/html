//go:build examples

package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// MediaAndLinks demonstrates extracting images, videos, audio, and links.
func main() {
	fmt.Println("=== Media and Link Extraction ===\n ")

	htmlContent := `
		<html>
		<head>
			<base href="https://example.com/blog/">
			<link rel="stylesheet" href="style.css" title="Main Style">
			<link rel="icon" href="/favicon.ico">
			<script src="app.js"></script>
		</head>
		<body>
			<article>
				<h1>Multimedia Article</h1>

				<img src="hero.jpg" alt="Hero Image" width="1200" height="600">
				<img src="/icons/thumb.png" alt="Thumbnail">

				<h2>Video Tutorial</h2>
				<video src="tutorial.mp4" poster="video-thumb.jpg" width="640" height="360">
					<source src="tutorial.webm" type="video/webm">
				</video>

				<h2>Audio Podcast</h2>
				<audio src="podcast.mp3" type="audio/mpeg">
					<source src="podcast.ogg" type="audio/ogg">
				</audio>

				<h2>Resources</h2>
				<ul>
					<li><a href="https://golang.org">Go Website</a></li>
					<li><a href="/docs">Documentation</a></li>
					<li><a href="author.html">About Author</a></li>
				</ul>
			</article>
		</body>
		</html>
	`

	processor := html.NewWithDefaults()
	defer processor.Close()

	result, err := processor.ExtractWithDefaults(htmlContent)
	if err != nil {
		log.Fatal(err)
	}

	// Display images
	fmt.Println("1. Images:")
	for i, img := range result.Images {
		fmt.Printf("   [%d] %s\n", i+1, img.URL)
		fmt.Printf("       Alt: %q, Size: %sx%s\n", img.Alt, img.Width, img.Height)
		fmt.Printf("       Decorative: %v\n", img.IsDecorative)
	}

	// Display videos
	fmt.Println("\n2. Videos:")
	for i, vid := range result.Videos {
		fmt.Printf("   [%d] %s\n", i+1, vid.URL)
		fmt.Printf("       Type: %s, Poster: %s\n", vid.Type, vid.Poster)
		fmt.Printf("       Size: %sx%s\n", vid.Width, vid.Height)
	}

	// Display audio
	fmt.Println("\n3. Audio:")
	for i, aud := range result.Audios {
		fmt.Printf("   [%d] %s\n", i+1, aud.URL)
		fmt.Printf("       Type: %s\n", aud.Type)
	}

	// Display links with metadata
	fmt.Println("\n4. Links:")
	for i, link := range result.Links {
		fmt.Printf("   [%d] Text: %q\n", i+1, link.Text)
		fmt.Printf("       URL: %s\n", link.URL)
		fmt.Printf("       External: %v, NoFollow: %v\n", link.IsExternal, link.IsNoFollow)
	}

	// Link extraction (for resources like CSS, JS, etc.)
	fmt.Println("\n5. Resource links:")
	config := html.DefaultLinkExtractionConfig()
	config.ResolveRelativeURLs = true

	links, err := processor.ExtractAllLinks(htmlContent, config)
	if err != nil {
		log.Fatal(err)
	}

	grouped := html.GroupLinksByType(links)
	for _, typ := range []string{"css", "js", "icon", "image", "video", "audio"} {
		if items, ok := grouped[typ]; ok {
			fmt.Printf("   %s (%d): ", typ, len(items))
			for i, item := range items {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(item.URL)
			}
			fmt.Println()
		}
	}

	// Selective extraction
	fmt.Println("\n6. Selective extraction (CSS and JS only):")
	selectiveConfig := html.DefaultLinkExtractionConfig()
	selectiveConfig.IncludeCSS = true
	selectiveConfig.IncludeJS = true
	selectiveConfig.IncludeImages = false
	selectiveConfig.IncludeVideos = false
	selectiveConfig.IncludeAudios = false
	selectiveConfig.IncludeContentLinks = false

	selectiveLinks, _ := processor.ExtractAllLinks(htmlContent, selectiveConfig)
	for _, link := range selectiveLinks {
		fmt.Printf("   - [%s] %s\n", link.Type, link.URL)
	}

}
