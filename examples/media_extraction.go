//go:build examples

package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// MediaExtraction demonstrates extracting images, videos, audio, and links
// with complete metadata from HTML content.
func main() {
	processor := html.NewWithDefaults()
	defer processor.Close()

	htmlContent := `
		<html>
		<body>
			<article>
				<h1>Multimedia Content Example</h1>
				<p>This article contains various media types.</p>
				
				<img src="https://example.com/hero.jpg" alt="Hero Image" width="1200" height="600">
				<img src="https://example.com/icon.png" alt="" width="32" height="32">
				
				<p>Check out this video:</p>
				<video src="https://example.com/tutorial.mp4" poster="thumbnail.jpg" width="640" height="360">
					<source src="tutorial.webm" type="video/webm">
				</video>
				
				<p>Listen to the podcast:</p>
				<audio src="https://example.com/episode1.mp3" type="audio/mpeg"></audio>
				
				<p>Visit our <a href="https://example.com" title="External Site">website</a> 
				or read the <a href="/docs" title="Documentation">documentation</a>.</p>
				
				<p>Sponsored link: <a href="https://sponsor.com" rel="nofollow">Sponsor</a></p>
			</article>
		</body>
		</html>
	`

	result, err := processor.ExtractWithDefaults(htmlContent)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Media Extraction Example ===\n ")

	// Display images
	fmt.Printf("Images (%d):\n", len(result.Images))
	for i, img := range result.Images {
		fmt.Printf("  [%d] URL: %s\n", i+1, img.URL)
		fmt.Printf("      Alt: %s\n", img.Alt)
		fmt.Printf("      Size: %s x %s\n", img.Width, img.Height)
		fmt.Printf("      Decorative: %v\n", img.IsDecorative)
		fmt.Println()
	}

	// Display videos
	fmt.Printf("Videos (%d):\n", len(result.Videos))
	for i, video := range result.Videos {
		fmt.Printf("  [%d] URL: %s\n", i+1, video.URL)
		fmt.Printf("      Type: %s\n", video.Type)
		fmt.Printf("      Poster: %s\n", video.Poster)
		fmt.Printf("      Size: %s x %s\n", video.Width, video.Height)
		fmt.Println()
	}

	// Display audio
	fmt.Printf("Audio (%d):\n", len(result.Audios))
	for i, audio := range result.Audios {
		fmt.Printf("  [%d] URL: %s\n", i+1, audio.URL)
		fmt.Printf("      Type: %s\n", audio.Type)
		fmt.Println()
	}

	// Display links
	fmt.Printf("Links (%d):\n", len(result.Links))
	for i, link := range result.Links {
		fmt.Printf("  [%d] Text: %s\n", i+1, link.Text)
		fmt.Printf("      URL: %s\n", link.URL)
		fmt.Printf("      External: %v, NoFollow: %v\n", link.IsExternal, link.IsNoFollow)
		fmt.Println()
	}
}
