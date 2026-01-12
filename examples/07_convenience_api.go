// Package main demonstrates the enhanced convenience API of the html library.
// This example shows how to use the new package-level convenience functions
// for common HTML processing tasks.
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/cybergodev/html"
)

func main() {
	// Sample HTML content demonstrating various use cases
	sampleHTML := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Convenience API Demo</title>
	</head>
	<body>
		<h1>Introduction to Convenience API</h1>
		<p>The html library provides many convenient functions for HTML processing.</p>

		<h2>Format Conversion</h2>
		<p>Convert HTML to various formats like Markdown or JSON with simple function calls.</p>
		<img src="https://example.com/image1.jpg" alt="Demo Image 1" width="800" height="600">

		<h2>Media Extraction</h2>
		<p>Easily extract images, videos, links, and audio from HTML content.</p>
		<a href="https://example.com/page1">Link 1</a>
		<a href="https://example.com/page2">Link 2</a>

		<video src="https://example.com/video.mp4" width="1280" height="720" poster="https://example.com/poster.jpg"></video>

		<h2>Text Processing</h2>
		<p>Generate summaries, clean text, and get reading time estimates.</p>
		<p>This is a longer paragraph with multiple sentences. It demonstrates how the summarization
		function can limit the word count. The text will be truncated at the specified limit with
		an ellipsis added to indicate continuation.</p>
	</body>
	</html>
	`

	fmt.Println("=== Convenience API Demo ===\n ")

	// Example 1: Quick Markdown conversion
	fmt.Println("1. Extract to Markdown")
	fmt.Println(strings.Repeat("-", 50))
	markdown, err := html.ExtractToMarkdown(sampleHTML)
	if err != nil {
		log.Fatalf("ExtractToMarkdown failed: %v", err)
	}
	fmt.Println(markdown)
	fmt.Println()

	// Example 2: Extract to JSON
	fmt.Println("2. Extract to JSON")
	fmt.Println(strings.Repeat("-", 50))
	jsonData, err := html.ExtractToJSON(sampleHTML)
	if err != nil {
		log.Fatalf("ExtractToJSON failed: %v", err)
	}
	fmt.Println(string(jsonData))
	fmt.Println()

	// Example 3: Extract title and text together
	fmt.Println("3. Extract Title and Text")
	fmt.Println(strings.Repeat("-", 50))
	title, text, err := html.ExtractWithTitle(sampleHTML)
	if err != nil {
		log.Fatalf("ExtractWithTitle failed: %v", err)
	}
	fmt.Printf("Title: %s\n", title)
	fmt.Printf("Text preview: %s\n", truncate(text, 150))
	fmt.Println()

	// Example 4: Extract only images
	fmt.Println("4. Extract Images Only")
	fmt.Println(strings.Repeat("-", 50))
	images, err := html.ExtractImages(sampleHTML)
	if err != nil {
		log.Fatalf("ExtractImages failed: %v", err)
	}
	fmt.Printf("Found %d images:\n", len(images))
	for i, img := range images {
		fmt.Printf("  %d. %s (alt: %s, %sx%s)\n",
			i+1, img.URL, img.Alt, img.Width, img.Height)
	}
	fmt.Println()

	// Example 5: Extract only videos
	fmt.Println("5. Extract Videos Only")
	fmt.Println(strings.Repeat("-", 50))
	videos, err := html.ExtractVideos(sampleHTML)
	if err != nil {
		log.Fatalf("ExtractVideos failed: %v", err)
	}
	fmt.Printf("Found %d videos:\n", len(videos))
	for i, vid := range videos {
		fmt.Printf("  %d. %s (poster: %s, %sx%s)\n",
			i+1, vid.URL, vid.Poster, vid.Width, vid.Height)
	}
	fmt.Println()

	// Example 6: Extract only links
	fmt.Println("6. Extract Links Only")
	fmt.Println(strings.Repeat("-", 50))
	links, err := html.ExtractLinks(sampleHTML)
	if err != nil {
		log.Fatalf("ExtractLinks failed: %v", err)
	}
	fmt.Printf("Found %d links:\n", len(links))
	for i, link := range links {
		fmt.Printf("  %d. %s - %s (external: %v)\n",
			i+1, link.URL, link.Text, link.IsExternal)
	}
	fmt.Println()

	// Example 7: Summarize content
	fmt.Println("7. Summarize Content (20 words)")
	fmt.Println(strings.Repeat("-", 50))
	summary, err := html.Summarize(sampleHTML, 20)
	if err != nil {
		log.Fatalf("Summarize failed: %v", err)
	}
	fmt.Println(summary)
	fmt.Println()

	// Example 8: Clean text extraction
	fmt.Println("8. Extract and Clean Text")
	fmt.Println(strings.Repeat("-", 50))
	clean, err := html.ExtractAndClean(sampleHTML)
	if err != nil {
		log.Fatalf("ExtractAndClean failed: %v", err)
	}
	fmt.Println(truncate(clean, 200))
	fmt.Println()

	// Example 9: Get reading time
	fmt.Println("9. Get Reading Time")
	fmt.Println(strings.Repeat("-", 50))
	readingTime, err := html.GetReadingTime(sampleHTML)
	if err != nil {
		log.Fatalf("GetReadingTime failed: %v", err)
	}
	fmt.Printf("Estimated reading time: %.1f minutes\n", readingTime)
	fmt.Println()

	// Example 10: Get word count
	fmt.Println("10. Get Word Count")
	fmt.Println(strings.Repeat("-", 50))
	wordCount, err := html.GetWordCount(sampleHTML)
	if err != nil {
		log.Fatalf("GetWordCount failed: %v", err)
	}
	fmt.Printf("Total words: %d\n", wordCount)
	fmt.Println()

	// Example 11: Extract only title
	fmt.Println("11. Extract Only Title")
	fmt.Println(strings.Repeat("-", 50))
	onlyTitle, err := html.ExtractTitle(sampleHTML)
	if err != nil {
		log.Fatalf("ExtractTitle failed: %v", err)
	}
	fmt.Printf("Title: %s\n", onlyTitle)
	fmt.Println()

	// Example 12: Using configuration presets
	fmt.Println("12. Using Configuration Presets")
	fmt.Println(strings.Repeat("-", 50))

	// RSS preset - faster processing, no article detection
	processor := html.NewWithDefaults()
	defer processor.Close()

	rssResult, err := processor.Extract(sampleHTML, html.ConfigForRSS())
	if err != nil {
		log.Fatalf("ConfigForRSS failed: %v", err)
	}
	fmt.Printf("RSS config - Images: %d, Links: %d\n", len(rssResult.Images), len(rssResult.Links))

	// Summary preset - text only, no media
	summaryResult, err := processor.Extract(sampleHTML, html.ConfigForSummary())
	if err != nil {
		log.Fatalf("ConfigForSummary failed: %v", err)
	}
	fmt.Printf("Summary config - Text length: %d chars, Media: 0\n", len(summaryResult.Text))

	// Search index preset - all metadata
	searchResult, err := processor.Extract(sampleHTML, html.ConfigForSearchIndex())
	if err != nil {
		log.Fatalf("ConfigForSearchIndex failed: %v", err)
	}
	fmt.Printf("Search config - Images: %d, Links: %d, Videos: %d, Audios: %d\n",
		len(searchResult.Images), len(searchResult.Links),
		len(searchResult.Videos), len(searchResult.Audios))
}

// truncate truncates a string to the specified length and adds "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
