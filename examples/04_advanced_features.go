//go:build examples

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cybergodev/html"
)

// AdvancedFeatures demonstrates production features and standard library compatibility.
func main() {
	fmt.Println("=== Advanced Features ===\n ")

	// Part 1: Custom Configuration
	advancedConfig()

	fmt.Println("\n=== Standard Library Compatibility ===\n ")

	// Part 2: golang.org/x/net/html Compatibility
	compatibility()
}

func advancedConfig() {
	// Custom processor configuration
	fmt.Println("1. Custom processor configuration:")
	config := html.Config{
		MaxInputSize:       10 * 1024 * 1024, // 10MB
		ProcessingTimeout:  15 * time.Second,
		MaxCacheEntries:    500,
		CacheTTL:           30 * time.Minute,
		WorkerPoolSize:     8,
		EnableSanitization: true,
		MaxDepth:           50,
	}

	processor, err := html.New(config)
	if err != nil {
		log.Fatal(err)
	}
	defer processor.Close()

	fmt.Printf("   Max Input: %d bytes, Timeout: %v, Cache: %d entries, Workers: %d\n\n",
		config.MaxInputSize, config.ProcessingTimeout, config.MaxCacheEntries, config.WorkerPoolSize)

	// Caching performance
	fmt.Println("2. Caching performance:")
	testHTML := `<article><h1>Cache Test</h1><p>Testing caching performance.</p></article>`

	start := time.Now()
	processor.ExtractWithDefaults(testHTML)
	missTime := time.Since(start)

	start = time.Now()
	processor.ExtractWithDefaults(testHTML)
	hitTime := time.Since(start)

	fmt.Printf("   Cache miss: %v, Cache hit: %v (%.1fx faster)\n\n",
		missTime, hitTime, float64(missTime)/float64(hitTime))

	// Batch processing
	fmt.Println("3. Batch processing:")
	docs := []string{
		`<article><h1>Article 1</h1><p>Content about Go programming.</p></article>`,
		`<article><h1>Article 2</h1><p>Content about web development.</p></article>`,
		`<article><h1>Article 3</h1><p>Content about database design.</p></article>`,
	}

	start = time.Now()
	results, err := processor.ExtractBatch(docs, html.DefaultExtractConfig())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Processed %d documents in %v\n", len(results), time.Since(start))

	// Concurrent usage
	fmt.Println("\n4. Concurrent processing:")
	var wg sync.WaitGroup
	start = time.Now()

	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			html := fmt.Sprintf(`<article><h1>Article %d</h1><p>Content %d.</p></article>`, id, id)
			processor.ExtractWithDefaults(html)
		}(i)
	}

	wg.Wait()
	fmt.Printf("   Processed 10 documents concurrently in %v\n", time.Since(start))

	// Statistics
	fmt.Println("\n5. Statistics:")
	stats := processor.GetStatistics()
	fmt.Printf("   Total: %d, Cache Hits: %d (%.1f%%), Errors: %d\n",
		stats.TotalProcessed,
		stats.CacheHits,
		float64(stats.CacheHits)/float64(stats.TotalProcessed)*100,
		stats.ErrorCount)
}

func compatibility() {
	// Parse HTML
	fmt.Println("1. Parse HTML:")
	doc, err := html.Parse(strings.NewReader("<html><body><h1>Hello</h1></body></html>"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Document type: %v\n\n", doc.Type)

	// Render HTML
	fmt.Println("2. Render HTML:")
	var buf bytes.Buffer
	html.Render(&buf, doc)
	fmt.Printf("   Rendered: %s\n\n", buf.String()[:40]+"...")

	// Escape/Unescape
	fmt.Println("3. Escape HTML:")
	escaped := html.EscapeString("<script>alert('xss')</script>")
	fmt.Printf("   %s -> %s\n\n", "<script>alert('xss')</script>", escaped)

	fmt.Println("4. Unescape HTML:")
	unescaped := html.UnescapeString("&lt;html&gt; &copy; 2024")
	fmt.Printf("   %s -> %s\n\n", "&lt;html&gt; &copy; 2024", unescaped)

	// Tokenize
	fmt.Println("5. Tokenize HTML:")
	tokenizer := html.NewTokenizer(strings.NewReader("<p>Hello <strong>World</strong></p>"))
	count := 0
	for {
		if tokenizer.Next() == html.ErrorToken {
			break
		}
		count++
	}
	fmt.Printf("   Found %d tokens\n\n", count)

	// Node creation
	fmt.Println("6. Node creation:")
	node := &html.Node{
		Type: html.ElementNode,
		Data: "div",
		Attr: []html.Attribute{{Key: "class", Val: "container"}},
	}
	fmt.Printf("   Created node: <%s class=%q>\n\n", node.Data, node.Attr[0].Val)

	// Bonus: Enhanced extraction
	fmt.Println("7. Enhanced extraction (bonus feature):")
	p := html.NewWithDefaults()
	defer p.Close()

	result, err := p.ExtractWithDefaults(`<article><h1>Article</h1><p>Detailed content.</p></article>`)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Title: %s, Words: %d, Time: %v\n", result.Title, result.WordCount, result.ReadingTime)

	// Render to stdout
	fmt.Println("\n8. Render to stdout:")
	fmt.Print("   ")
	html.Render(os.Stdout, node)
	fmt.Println()

	fmt.Println("\n=== Summary ===")
	fmt.Println("✓ 100% compatible with golang.org/x/net/html")
	fmt.Println("✓ Drop-in replacement - just change import path")
	fmt.Println("✓ Plus enhanced content extraction features")
}
