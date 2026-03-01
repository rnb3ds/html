//go:build examples

package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/cybergodev/html"
)

// This example demonstrates link extraction, URL resolution, and media extraction.
// Learn how to extract links, images, videos, and audio from HTML content.
func main() {
	fmt.Println("=== Links & Media Extraction ===\n")

	htmlContent := `
		<html>
			<head>
				<base href="https://example.com/blog/">
				<link rel="stylesheet" href="/assets/style.css">
			</head>
			<body>
				<nav>
					<a href="/">Home</a>
					<a href="/about">About</a>
					<a href="https://golang.org">Official Site</a>
				</nav>
				<article>
					<h1>Go Programming</h1>
					<img src="hero.jpg" alt="Hero Image" width="800" height="400">
					<p>Read more at <a href="tutorial.html">Tutorial</a></p>
					<video src="demo.mp4" poster="thumb.jpg" width="640">
						<source src="demo.webm" type="video/webm">
					</video>
					<audio src="podcast.mp3" controls></audio>
				</article>
			</body>
		</html>
	`

	processor, _ := html.New()
	defer processor.Close()

	// ============================================================
	// PART 1: Link extraction with URL resolution
	// ============================================================
	fmt.Println("Part 1: Link Extraction")
	fmt.Println("-----------------------")

	config := html.DefaultLinkExtractionConfig()
	config.ResolveRelativeURLs = true

	links, err := html.ExtractAllLinks([]byte(htmlContent), config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d links:\n", len(links))

	// Categorize links
	for _, link := range links {
		icon := "•"
		switch link.Type {
		case "stylesheet":
			icon = "🎨"
		case "content":
			icon = "📄"
		default:
			if strings.HasPrefix(link.URL, "https://") && !strings.Contains(link.URL, "example.com") {
				icon = "🔗"
			}
		}
		fmt.Printf("  %s %s [%s]\n", icon, link.URL, link.Type)
	}

	// ============================================================
	// PART 2: Filter links by type
	// ============================================================
	fmt.Println("\nPart 2: Filter Links by Type")
	fmt.Println("----------------------------")

	filterConfig := html.DefaultLinkExtractionConfig()
	filterConfig.IncludeCSS = false
	filterConfig.IncludeJS = false

	filteredLinks, _ := html.ExtractAllLinks([]byte(htmlContent), filterConfig)
	fmt.Printf("Content links only (CSS/JS excluded): %d\n", len(filteredLinks))

	// ============================================================
	// PART 3: Image extraction
	// ============================================================
	fmt.Println("\nPart 3: Image Extraction")
	fmt.Println("------------------------")

	result, _ := processor.Extract([]byte(htmlContent), html.DefaultExtractConfig())

	fmt.Printf("Found %d images:\n", len(result.Images))
	for i, img := range result.Images {
		fmt.Printf("  %d. %s\n", i+1, img.URL)
		fmt.Printf("     Alt: %q, Size: %sx%s\n", img.Alt, img.Width, img.Height)
		if img.Alt == "" {
			fmt.Printf("     ⚠ Missing alt text (bad for accessibility)\n")
		}
	}

	// ============================================================
	// PART 4: Video extraction
	// ============================================================
	fmt.Println("\nPart 4: Video Extraction")
	fmt.Println("------------------------")

	fmt.Printf("Found %d videos:\n", len(result.Videos))
	for i, vid := range result.Videos {
		fmt.Printf("  %d. %s\n", i+1, vid.URL)
		fmt.Printf("     Type: %s, Poster: %s\n", vid.Type, vid.Poster)
		fmt.Printf("     Size: %sx%s\n", vid.Width, vid.Height)
	}

	// ============================================================
	// PART 5: Audio extraction
	// ============================================================
	fmt.Println("\nPart 5: Audio Extraction")
	fmt.Println("------------------------")

	fmt.Printf("Found %d audio files:\n", len(result.Audios))
	for i, aud := range result.Audios {
		fmt.Printf("  %d. %s\n", i+1, aud.URL)
		fmt.Printf("     Type: %s\n", aud.Type)
	}

	// ============================================================
	// PART 6: Selective media extraction
	// ============================================================
	fmt.Println("\nPart 6: Selective Media Extraction")
	fmt.Println("----------------------------------")

	// Images only
	imageOnlyConfig := html.ExtractConfig{
		ExtractArticle: true,
		PreserveImages: true,
		PreserveVideos: false,
		PreserveAudios: false,
	}
	imgResult, _ := processor.Extract([]byte(htmlContent), imageOnlyConfig)
	fmt.Printf("Images only: %d images, %d videos, %d audio\n",
		len(imgResult.Images), len(imgResult.Videos), len(imgResult.Audios))

	// Videos and audio only
	mediaOnlyConfig := html.ExtractConfig{
		ExtractArticle: true,
		PreserveImages: false,
		PreserveVideos: true,
		PreserveAudios: true,
	}
	mediaResult, _ := processor.Extract([]byte(htmlContent), mediaOnlyConfig)
	fmt.Printf("Media only: %d images, %d videos, %d audio\n",
		len(mediaResult.Images), len(mediaResult.Videos), len(mediaResult.Audios))

	// ============================================================
	// Summary
	// ============================================================
	fmt.Println("\n=== Quick Reference ===")
	fmt.Println("Link filtering:")
	fmt.Println("  IncludeCSS, IncludeJS, IncludeImages, IncludeVideos, IncludeAudios")
	fmt.Println("  IncludeContentLinks, IncludeExternalLinks, IncludeIcons")
	fmt.Println()
	fmt.Println("URL resolution:")
	fmt.Println("  ResolveRelativeURLs=true, BaseURL=\"https://example.com/\"")
	fmt.Println()
	fmt.Println("Media extraction:")
	fmt.Println("  PreserveImages, PreserveVideos, PreserveAudios")
}
