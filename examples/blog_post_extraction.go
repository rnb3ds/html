package main

import (
	"fmt"
	"log"

	"github.com/cybergodev/html"
)

// BlogPostExtraction demonstrates extracting clean article content
// from a typical blog post with navigation, sidebar, and ads.
func main() {
	processor := html.NewWithDefaults()
	defer processor.Close()

	// Typical blog HTML with navigation, sidebar, ads
	blogHTML := `
		<html>
		<head>
			<title>Understanding Go Interfaces | My Tech Blog</title>
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
					
					<p>Interfaces are one of Go's most powerful features. They provide a way to specify the behavior of an object: if something can do this, then it can be used here.</p>
					
					<img src="https://example.com/interface-diagram.png" alt="Go Interface Diagram" width="800" height="400">
					
					<p>As shown in the diagram above, interfaces provide abstraction and polymorphism in Go. Unlike many other languages, Go's interfaces are implemented implicitly.</p>
					
					<h2>Key Concepts</h2>
					<p>An interface type is defined as a set of method signatures. A value of interface type can hold any value that implements those methods.</p>
					
					<p>For more information, visit the <a href="https://golang.org/doc/effective_go#interfaces">official Go documentation</a>.</p>
				</article>
			</main>
			
			<footer>
				<p>© 2024 My Tech Blog. All rights reserved.</p>
				<p><a href="/privacy">Privacy Policy</a> | <a href="/terms">Terms of Service</a></p>
			</footer>
		</body>
		</html>
	`

	config := html.ExtractConfig{
		ExtractArticle:    true,       // Remove navigation, ads, sidebar
		PreserveImages:    true,       // Keep image metadata
		PreserveLinks:     true,       // Keep links
		InlineImageFormat: "markdown", // Convert to Markdown
	}

	result, err := processor.Extract(blogHTML, config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Blog Post Extraction Example ===\n ")
	fmt.Println("Title:", result.Title)
	fmt.Println("\nExtracted Content (Markdown format):")
	fmt.Println(result.Text)
	fmt.Println("\n=== Metadata ===")
	fmt.Printf("Word Count: %d\n", result.WordCount)
	fmt.Printf("Reading Time: %v\n", result.ReadingTime)
	fmt.Printf("Processing Time: %v\n", result.ProcessingTime)

	fmt.Println("\n=== Images ===")
	for i, img := range result.Images {
		fmt.Printf("[%d] %s\n", i+1, img.URL)
		fmt.Printf("    Alt: %s\n", img.Alt)
		fmt.Printf("    Size: %s x %s\n", img.Width, img.Height)
	}

	fmt.Println("\n=== Links ===")
	for i, link := range result.Links {
		fmt.Printf("[%d] %s -> %s\n", i+1, link.Text, link.URL)
		fmt.Printf("    External: %v\n", link.IsExternal)
	}
}
