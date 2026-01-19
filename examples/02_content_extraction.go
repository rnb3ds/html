//go:build examples

package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// ContentExtraction demonstrates comprehensive content extraction features
// including article detection, inline images, and metadata extraction.
func main() {
	processor := html.NewWithDefaults()
	defer processor.Close()

	// Realistic blog post HTML with navigation, sidebar, ads
	blogHTML := `
		<html>
		<head>
			<title>Understanding Go Interfaces | Tech Blog</title>
		</head>
		<body>
			<nav>
				<ul>
					<li><a href="/">Home</a></li>
					<li><a href="/about">About</a></li>
					<li><a href="/contact">Contact</a></li>
				</ul>
			</nav>
			
			<aside class="sidebar">
				<div class="ad">
					<h3>Advertisement</h3>
					<p>Buy our amazing product now!</p>
				</div>
				<div class="related">
					<h3>Related Posts</h3>
					<ul>
						<li><a href="/post1">Post 1</a></li>
						<li><a href="/post2">Post 2</a></li>
					</ul>
				</div>
			</aside>
			
			<main>
				<article>
					<h1>Understanding Go Interfaces</h1>
					<p class="meta">Published on January 15, 2024 by John Doe</p>
					
					<p>Interfaces are one of Go's most powerful features. They provide a way to specify the behavior of an object.</p>
					
					<img src="https://example.com/interface-diagram.png" alt="Go Interface Diagram" width="800" height="400">
					
					<p>As shown in the diagram above, interfaces provide abstraction and polymorphism in Go.</p>
					
					<h2>Key Concepts</h2>
					<p>An interface type is defined as a set of method signatures. A value of interface type can hold any value that implements those methods.</p>
					
					<p>For more information, visit the <a href="https://golang.org/doc/effective_go#interfaces">official Go documentation</a>.</p>
				</article>
			</main>
			
			<footer>
				<p>© 2024 Tech Blog. All rights reserved.</p>
				<p><a href="/privacy">Privacy Policy</a> | <a href="/terms">Terms of Service</a></p>
			</footer>
		</body>
		</html>
	`

	fmt.Println("=== Content Extraction Example ===\n ")

	// Example 1: Smart article detection (removes navigation, ads, sidebar, footer)
	fmt.Println("1. Smart article detection:")
	config1 := html.ExtractConfig{
		ExtractArticle: true, // Automatically removes noise
		PreserveImages: true,
		PreserveLinks:  true,
	}

	result1, err := processor.Extract(blogHTML, config1)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Title: %s\n", result1.Title)
	fmt.Printf("   Word Count: %d\n", result1.WordCount)
	fmt.Printf("   Reading Time: %v\n", result1.ReadingTime)
	fmt.Printf("   Images: %d, Links: %d\n\n", len(result1.Images), len(result1.Links))

	// Example 2: Inline images with Markdown format
	fmt.Println("2. Inline images (Markdown format):")
	config2 := html.ExtractConfig{
		ExtractArticle:    true,
		PreserveImages:    true,
		InlineImageFormat: "markdown",
	}

	result2, err := processor.Extract(blogHTML, config2)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   Content with inline images:")
	fmt.Println("   " + result2.Text[:300] + "...\n")

	// Example 3: Inline images with HTML format
	fmt.Println("3. Inline images (HTML format):")
	config3 := html.ExtractConfig{
		ExtractArticle:    true,
		PreserveImages:    true,
		InlineImageFormat: "html",
	}

	result3, err := processor.Extract(blogHTML, config3)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   Content with HTML images:")
	fmt.Println("   " + result3.Text[:300] + "...\n")

	// Example 4: Inline images with placeholder format
	fmt.Println("4. Inline images (Placeholder format):")
	config4 := html.ExtractConfig{
		ExtractArticle:    true,
		PreserveImages:    true,
		InlineImageFormat: "placeholder",
	}

	result4, err := processor.Extract(blogHTML, config4)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   Content with placeholders:")
	fmt.Println("   " + result4.Text[:300] + "...\n")

	// Example 5: Minimal extraction (text only, no metadata)
	fmt.Println("5. Minimal extraction (text only):")
	config5 := html.ExtractConfig{
		ExtractArticle: false,
		PreserveImages: false,
		PreserveLinks:  false,
		PreserveVideos: false,
		PreserveAudios: false,
	}

	result5, err := processor.Extract(blogHTML, config5)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Text: %s\n", result5.Text[:300]+"...")
	fmt.Printf("   Images: %d, Links: %d\n\n", len(result5.Images), len(result5.Links))

	fmt.Println("✓ Content extraction complete!")
}
