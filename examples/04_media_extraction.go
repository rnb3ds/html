//go:build examples

package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// This example demonstrates extraction of media files: images, videos, and audio.
// Learn how to get media URLs, metadata, and technical details.
func main() {
	fmt.Println("=== Media Extraction: Images, Videos, Audio ===\n ")

	htmlContent := `
		<html>
			<head>
				<base href="https://example.com/article/">
			</head>
			<body>
				<article>
					<h1>Multimedia Tutorial</h1>

					<h2>Featured Image</h2>
					<img src="hero-banner.jpg"
					     alt="Article hero banner"
					     width="1920"
					     height="600">

					<h2>Inline Diagram</h2>
					<img src="../images/architecture.png"
					     alt="System Architecture Diagram"
					     width="800"
					     height="600">

					<h2>Video Tutorial</h2>
					<video src="tutorial.mp4"
					       poster="video-thumbnail.jpg"
					       width="1280"
					       height="720"
					       controls>
						<source src="tutorial.webm" type="video/webm">
						<source src="tutorial-480p.mp4" type="video/mp4">
					</video>

					<h2>Audio Podcast</h2>
					<audio src="podcast.mp3" controls>
						<source src="podcast.ogg" type="audio/ogg">
					</audio>

					<h2>Image Gallery</h2>
					<p>
						<img src="thumb1.jpg" alt="Screenshot 1">
						<img src="thumb2.jpg" alt="Screenshot 2">
						<img src="thumb3.jpg" alt="Screenshot 3">
					</p>

					<h2>Embedded Content</h2>
					<iframe src="https://www.youtube.com/embed/dQw4w9WgXcQ"
					        width="560"
					        height="315">
					</iframe>
				</article>
			</body>
		</html>
	`

	processor, _ := html.New()
	defer processor.Close()

	result, err := processor.Extract([]byte(htmlContent), html.DefaultExtractConfig())
	if err != nil {
		log.Fatal(err)
	}

	// ============================================================
	// PART 1: Image extraction
	// ============================================================
	fmt.Println("Part 1: Images")
	fmt.Println("-----------")

	fmt.Printf("Total images found: %d\n\n", len(result.Images))

	for i, img := range result.Images {
		fmt.Printf("Image #%d:\n", i+1)
		fmt.Printf("  URL:      %s\n", img.URL)
		fmt.Printf("  Alt Text: %q\n", img.Alt)
		fmt.Printf("  Size:     %s × %s pixels\n", img.Width, img.Height)
		fmt.Printf("  Decorative: %v\n", img.IsDecorative)

		// Practical tip
		if img.IsDecorative {
			fmt.Printf("  → Tip: Marked as decorative (good for accessibility)\n")
		}
		if img.Alt == "" {
			fmt.Printf("  → Warning: Missing alt text (bad for SEO/accessibility)\n")
		}
		fmt.Println()
	}

	// ============================================================
	// PART 2: Video extraction
	// ============================================================
	fmt.Println("Part 2: Videos")
	fmt.Println("-------------")

	fmt.Printf("Total videos found: %d\n\n", len(result.Videos))

	for i, vid := range result.Videos {
		fmt.Printf("Video #%d:\n", i+1)
		fmt.Printf("  URL:       %s\n", vid.URL)
		fmt.Printf("  Type:      %s\n", vid.Type)
		fmt.Printf("  Poster:    %s\n", vid.Poster)
		fmt.Printf("  Size:      %s × %s pixels\n", vid.Width, vid.Height)

		// Practical tips
		if vid.Poster != "" {
			fmt.Printf("  → Has thumbnail image\n")
		}
		if vid.Type == "video/webm" {
			fmt.Printf("  → WebM format (good for modern browsers)\n")
		}
		fmt.Println()
	}

	// ============================================================
	// PART 3: Audio extraction
	// ============================================================
	fmt.Println("Part 3: Audio")
	fmt.Println("-----------")

	fmt.Printf("Total audio files found: %d\n\n", len(result.Audios))

	for i, aud := range result.Audios {
		fmt.Printf("Audio #%d:\n", i+1)
		fmt.Printf("  URL:  %s\n", aud.URL)
		fmt.Printf("  Type: %s\n", aud.Type)

		// Type hints
		switch aud.Type {
		case "audio/mpeg":
			fmt.Printf("  → MP3 format (universal compatibility)\n")
		case "audio/ogg":
			fmt.Printf("  → OGG format (open source)\n")
		default:
			fmt.Printf("  → Custom audio format\n")
		}
		fmt.Println()
	}

	// ============================================================
	// PART 4: Extract only specific media types
	// ============================================================
	fmt.Println("Part 4: Selective Media Extraction")
	fmt.Println("----------------------------------")

	extractOptions := []struct {
		name        string
		config      html.ExtractConfig
		description string
	}{
		{
			name: "Images only",
			config: html.ExtractConfig{
				ExtractArticle: true,
				PreserveImages: true,
				PreserveVideos: false,
				PreserveAudios: false,
			},
			description: "Extract only images, skip videos/audio",
		},
		{
			name: "Videos and audio only",
			config: html.ExtractConfig{
				ExtractArticle: true,
				PreserveImages: false,
				PreserveVideos: true,
				PreserveAudios: true,
			},
			description: "Extract media files, skip images",
		},
		{
			name: "Everything (default)",
			config: html.ExtractConfig{
				ExtractArticle: true,
				PreserveImages: true,
				PreserveVideos: true,
				PreserveAudios: true,
			},
			description: "Extract all media types",
		},
	}

	for _, opt := range extractOptions {
		result, err := processor.Extract([]byte(htmlContent), opt.config)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("\n%s:\n", opt.name)
		fmt.Printf("  %s\n", opt.description)
		fmt.Printf("  Images: %d, Videos: %d, Audio: %d\n",
			len(result.Images), len(result.Videos), len(result.Audios))
	}

	// ============================================================
	// PART 5: Best practices for media extraction
	// ============================================================
	fmt.Println("\n\nPart 5: Media Extraction Best Practices")
	fmt.Println("------------------------------------")

	fmt.Println("✓ Always check alt text for accessibility")
	fmt.Println("  • Empty alt text means the image is decorative")
	fmt.Println("  • Missing alt attribute = poor accessibility")
	fmt.Println()

	fmt.Println("✓ Use videoType/audioType for format detection")
	fmt.Println("  • Different formats have different browser support")
	fmt.Println("  • Provide multiple sources for compatibility")
	fmt.Println()

	fmt.Println("✓ Consider file size when extracting media")
	fmt.Println("  • Set MaxInputSize to prevent memory issues")
	fmt.Println("  • Large media files can slow down extraction")
	fmt.Println()

	fmt.Println("✓ Handle missing media gracefully")
	fmt.Println("  • Always check len() before accessing slices")
	fmt.Println("  • Media URLs might be relative or absolute")
	fmt.Println()

	fmt.Println("✓ Poster images are extracted separately")
	fmt.Println("  • Videos have poster metadata")
	fmt.Println("  • Poster is also in the Images slice")

}
