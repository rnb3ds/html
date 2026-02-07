# 100% Compatibility with golang.org/x/net/html

This library is **100% API-compatible** with `golang.org/x/net/html` and serves as a **drop-in replacement** with enhanced content extraction features.

## What Does 100% Compatible Mean?

All types, functions, and constants from `golang.org/x/net/html` are re-exported:

### Types
- `Node` - HTML parse tree node
- `NodeType` - Type of node (ErrorNode, TextNode, DocumentNode, ElementNode, CommentNode, DoctypeNode)
- `Attribute` - HTML element attribute
- `Token` - HTML token
- `TokenType` - Type of token (ErrorToken, TextToken, StartTagToken, EndTagToken, SelfClosingTagToken, CommentToken, DoctypeToken)
- `Tokenizer` - HTML tokenizer

### Functions
- `Parse(r io.Reader) (*Node, error)` - Parse HTML document
- `ParseFragment(r io.Reader, context *Node) ([]*Node, error)` - Parse HTML fragment
- `Render(w io.Writer, n *Node) error` - Render HTML tree
- `EscapeString(s string) string` - Escape HTML special characters
- `UnescapeString(s string) string` - Unescape HTML entities (all 2,231 entities)
- `NewTokenizer(r io.Reader) *Tokenizer` - Create HTML tokenizer

## Migration Guide

### From golang.org/x/net/html

Simply change your import statement:

```go
// Before
import "golang.org/x/net/html"

// After
import "github.com/cybergodev/html"
```

**That's it!** All your existing code works unchanged.

### Real-World Migration Example

**Before (using golang.org/x/net/html):**
```go
package main

import (
    "fmt"
    "strings"
    "golang.org/x/net/html"
)

func main() {
    htmlContent := "<html><body><h1>Hello</h1><p>World</p></body></html>"
    
    doc, err := html.Parse(strings.NewReader(htmlContent))
    if err != nil {
        panic(err)
    }
    
    // Walk the tree and extract text
    var extractText func(*html.Node) string
    extractText = func(n *html.Node) string {
        if n.Type == html.TextNode {
            return n.Data
        }
        var text string
        for c := n.FirstChild; c != nil; c = c.NextSibling {
            text += extractText(c)
        }
        return text
    }
    
    text := extractText(doc)
    fmt.Println(text)
}
```

**After (using github.com/cybergodev/html):**
```go
package main

import (
    "fmt"
    "strings"
    "github.com/cybergodev/html"  // Only change: import path
)

func main() {
    htmlContent := "<html><body><h1>Hello</h1><p>World</p></body></html>"
    
    // Option 1: Use standard parsing (100% compatible)
    doc, err := html.Parse(strings.NewReader(htmlContent))
    if err != nil {
        panic(err)
    }
    
    // Same tree walking code works identically
    var extractText func(*html.Node) string
    extractText = func(n *html.Node) string {
        if n.Type == html.TextNode {
            return n.Data
        }
        var text string
        for c := n.FirstChild; c != nil; c = c.NextSibling {
            text += extractText(c)
        }
        return text
    }
    
    text := extractText(doc)
    fmt.Println(text)
    
    // Option 2: Or use enhanced extraction (simpler!)
    processor, err := html.New()
	if err != nil {
		log.Fatal(err)
	}
    defer processor.Close()
    
    result, err := processor.ExtractWithDefaults(htmlContent)
    if err != nil {
        panic(err)
    }
    
    fmt.Println("Title:", result.Title)
    fmt.Println("Text:", result.Text)
    fmt.Println("Words:", result.WordCount)
}
```

### Example: Standard HTML Parsing

```go
package main

import (
    "bytes"
    "fmt"
    "strings"
    
    "github.com/cybergodev/html"  // Drop-in replacement
)

func main() {
    // Parse HTML
    doc, err := html.Parse(strings.NewReader("<html><body><h1>Hello</h1></body></html>"))
    if err != nil {
        panic(err)
    }
    
    // Render HTML
    var buf bytes.Buffer
    html.Render(&buf, doc)
    fmt.Println(buf.String())
    
    // Escape/Unescape
    escaped := html.EscapeString("<script>alert('xss')</script>")
    fmt.Println(escaped)
    
    unescaped := html.UnescapeString("&lt;html&gt; &copy; 2024")
    fmt.Println(unescaped)
    
    // Tokenize
    tokenizer := html.NewTokenizer(strings.NewReader("<p>Test</p>"))
    for {
        tt := tokenizer.Next()
        if tt == html.ErrorToken {
            break
        }
        fmt.Printf("Token: %v\n", tt)
    }
    
    // Create nodes
    node := &html.Node{
        Type: html.ElementNode,
        Data: "div",
        Attr: []html.Attribute{
            {Key: "class", Val: "container"},
            {Key: "id", Val: "main"},
        },
    }
}
```

## Enhanced Features (Bonus)

In addition to standard HTML parsing, this library provides advanced content extraction:

```go
package main

import (
    "fmt"
    "strings"
    "github.com/cybergodev/html"
)

func main() {
    // Standard parsing still works
    doc, _ := html.Parse(strings.NewReader(htmlContent))
    
    // Plus enhanced extraction
    processor, err := html.New()
	if err != nil {
		log.Fatal(err)
	}
    defer processor.Close()
    
    result, err := processor.ExtractWithDefaults(`
        <article>
            <h1>Article Title</h1>
            <p>Main content here.</p>
            <img src="image.jpg" alt="Test">
        </article>
    `)
    
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Title: %s\n", result.Title)
    fmt.Printf("Text: %s\n", result.Text)
    fmt.Printf("Word Count: %d\n", result.WordCount)
    fmt.Printf("Images: %d\n", len(result.Images))
    fmt.Printf("Processing Time: %v\n", result.ProcessingTime)
}
```

## Two-Tier API Design

### Tier 1: Standard HTML Parsing (100% Compatible)

Use all `golang.org/x/net/html` APIs directly:

```go
doc, _ := html.Parse(r)
html.Render(w, doc)
escaped := html.EscapeString(s)
tokenizer := html.NewTokenizer(r)
```

### Tier 2: Enhanced Content Extraction (Library-Specific)

Use `Processor` for advanced features:

```go
processor, err := html.New()
	if err != nil {
		log.Fatal(err)
	}
defer processor.Close()

result, _ := processor.ExtractWithDefaults(htmlContent)
// Access: result.Title, result.Text, result.Images, result.Links, etc.

// Or with custom configuration
config := html.ExtractConfig{
    ExtractArticle:    true,
    PreserveImages:    true,
    PreserveLinks:     true,
    InlineImageFormat: "markdown",
}
result, _ = processor.Extract(htmlContent, config)
```

## Verification

All compatibility is verified with automated tests:

```bash
# Run compatibility tests
go test -v -run Compatibility

# Run examples
go run examples/compatibility.go
```

### Test Coverage

The library includes comprehensive compatibility tests that verify:

- ✓ **Node Types** - All 6 node types (ErrorNode, TextNode, DocumentNode, ElementNode, CommentNode, DoctypeNode)
- ✓ **Token Types** - All 7 token types (ErrorToken, TextToken, StartTagToken, EndTagToken, SelfClosingTagToken, CommentToken, DoctypeToken)
- ✓ **Parse Function** - Identical parsing behavior
- ✓ **Render Function** - Identical rendering output
- ✓ **EscapeString** - All HTML escape sequences
- ✓ **UnescapeString** - All 2,231 HTML entities
- ✓ **Tokenizer** - Identical tokenization behavior
- ✓ **Node Structure** - Identical node tree structure
- ✓ **ParseFragment** - Identical fragment parsing

Run `go test -v ./...` to verify all tests pass.

## Enhanced API Methods

Beyond standard HTML parsing, the `Processor` provides these additional methods:

### Content Extraction
```go
// Extract with default configuration
result, err := processor.ExtractWithDefaults(htmlContent)

// Extract with custom configuration
config := html.ExtractConfig{
    ExtractArticle:    true,       // Enable intelligent article detection
    PreserveImages:    true,       // Extract image metadata
    PreserveLinks:     true,       // Extract link metadata
    PreserveVideos:    true,       // Extract video metadata
    PreserveAudios:    true,       // Extract audio metadata
    InlineImageFormat: "markdown", // none, placeholder, markdown, html
}
result, err := processor.Extract(htmlContent, config)

// Extract from file
result, err := processor.ExtractFromFile("page.html", config)
```

### Batch Processing
```go
// Process multiple HTML strings in parallel
htmlContents := []string{html1, html2, html3}
results, err := processor.ExtractBatch(htmlContents, config)

// Process multiple files in parallel
filePaths := []string{"page1.html", "page2.html", "page3.html"}
results, err := processor.ExtractBatchFiles(filePaths, config)
```

### Monitoring & Cache Management
```go
// Get processing statistics
stats := processor.GetStatistics()
fmt.Printf("Total Processed: %d\n", stats.TotalProcessed)
fmt.Printf("Cache Hits: %d\n", stats.CacheHits)
fmt.Printf("Cache Misses: %d\n", stats.CacheMisses)
fmt.Printf("Average Time: %v\n", stats.AverageProcessTime)

// Clear cache
processor.ClearCache()

// Always close when done
defer processor.Close()
```

### Configuration Options
```go
// Create processor with custom configuration
config := html.Config{
    MaxInputSize:       50 * 1024 * 1024,  // 50MB max input
    ProcessingTimeout:  30 * time.Second,   // 30s timeout
    MaxCacheEntries:    1000,               // Cache 1000 results
    CacheTTL:           time.Hour,          // 1 hour TTL
    WorkerPoolSize:     4,                  // 4 parallel workers
    EnableSanitization: true,               // Sanitize HTML
    MaxDepth:           100,                // Max nesting depth
}
processor, err := html.New(config)

// Or use defaults
processor, err := html.New()
	if err != nil {
		log.Fatal(err)
	}
```

## Key Benefits

- ✅ **Zero migration cost** - Just change import path
- ✅ **100% compatible** - All APIs work identically
- ✅ **Plus enhanced features** - Advanced content extraction
- ✅ **Single dependency** - Only depends on `golang.org/x/net/html`
- ✅ **Fully tested** - All compatibility verified
- ✅ **Production-ready** - Thread-safe, secure, performant

## Result Structure

The `Result` type provides comprehensive extraction data:

```go
type Result struct {
    Text           string        // Extracted clean text
    Title          string        // Page/article title
    Images         []ImageInfo   // Image metadata
    Links          []LinkInfo    // Link metadata
    Videos         []VideoInfo   // Video metadata
    Audios         []AudioInfo   // Audio metadata
    ProcessingTime time.Duration // Processing duration
    WordCount      int           // Word count
    ReadingTime    time.Duration // Estimated reading time (200 WPM)
}

type ImageInfo struct {
    URL          string  // Image URL
    Alt          string  // Alt text
    Title        string  // Title attribute
    Width        string  // Width attribute
    Height       string  // Height attribute
    IsDecorative bool    // True if alt text is empty
    Position     int     // Position in text (for inline formatting)
}

type LinkInfo struct {
    URL        string  // Link URL
    Text       string  // Anchor text
    Title      string  // Title attribute
    IsExternal bool    // True if external domain
    IsNoFollow bool    // True if rel="nofollow"
}

type VideoInfo struct {
    URL      string  // Video URL (native, YouTube, Vimeo, direct)
    Type     string  // MIME type or "embed"
    Poster   string  // Poster image URL
    Width    string  // Width attribute
    Height   string  // Height attribute
    Duration string  // Duration attribute
}

type AudioInfo struct {
    URL      string  // Audio URL
    Type     string  // MIME type
    Duration string  // Duration attribute
}
```

## FAQ

### Q: Will my existing code break?
**A:** No. All `golang.org/x/net/html` APIs work identically.

### Q: Do I need to use the enhanced features?
**A:** No. They're optional. Standard parsing works as before.

### Q: What's the performance impact?
**A:** Zero for standard parsing. Enhanced features are opt-in via `Processor`.

### Q: Are all HTML entities supported?
**A:** Yes. All 2,231 HTML entities from `golang.org/x/net/html` are supported.

### Q: Is it thread-safe?
**A:** Yes. Both standard parsing and enhanced extraction are thread-safe. The `Processor` can be safely used by multiple goroutines concurrently.

### Q: What about dependencies?
**A:** Single dependency: `golang.org/x/net/html` (same as before).

### Q: How does caching work?
**A:** Content-addressable caching using SHA256 keys. Cache entries expire based on TTL (default: 1 hour) and are evicted using LRU when the cache is full (default: 1000 entries).

## Performance Considerations

### Standard Parsing
Using standard `html.Parse()`, `html.Render()`, etc. has **zero performance overhead** compared to `golang.org/x/net/html` - they are direct re-exports.

### Enhanced Extraction
The `Processor` adds intelligent features with minimal overhead:

- **First extraction**: Parses and analyzes content (~1-5ms for typical pages)
- **Cached extractions**: Near-instant retrieval from cache (~0.1ms)
- **Batch processing**: Parallel workers maximize throughput
- **Memory efficient**: Lazy cleanup, no background goroutines

**Caching benefits:**
```go
processor, err := html.New()
	if err != nil {
		log.Fatal(err)
	}
defer processor.Close()

// First call: ~2ms (parse + analyze)
result1, _ := processor.ExtractWithDefaults(htmlContent)

// Second call: ~0.1ms (cache hit)
result2, _ := processor.ExtractWithDefaults(htmlContent)

stats := processor.GetStatistics()
fmt.Printf("Cache hit rate: %.1f%%\n", 
    float64(stats.CacheHits)/float64(stats.TotalProcessed)*100)
```

## Best Practices

### 1. Reuse Processor Instances
```go
// ✅ Good: Create once, reuse many times
processor, err := html.New()
	if err != nil {
		log.Fatal(err)
	}
defer processor.Close()

for _, content := range htmlContents {
    result, _ := processor.ExtractWithDefaults(content)
    // Process result...
}

// ❌ Bad: Creating new processor per request
for _, content := range htmlContents {
    processor, err := html.New()
	if err != nil {
		log.Fatal(err)
	}
    result, _ := processor.ExtractWithDefaults(content)
    processor.Close()
}
```

### 2. Always Close Processor
```go
// ✅ Good: Use defer to ensure cleanup
processor, err := html.New()
	if err != nil {
		log.Fatal(err)
	}
defer processor.Close()

// ❌ Bad: Forgetting to close
processor, err := html.New()
	if err != nil {
		log.Fatal(err)
	}
result, _ := processor.ExtractWithDefaults(content)
// Processor never closed - cache not cleaned up
```

### 3. Use Batch Processing for Multiple Documents
```go
// ✅ Good: Parallel processing with worker pool
results, _ := processor.ExtractBatch(htmlContents, config)

// ❌ Bad: Sequential processing
for _, content := range htmlContents {
    result, _ := processor.Extract(content, config)
}
```

### 4. Configure Limits Appropriately
```go
// ✅ Good: Set limits based on your use case
config := html.Config{
    MaxInputSize:    10 * 1024 * 1024,  // 10MB for blog posts
    MaxCacheEntries: 500,                // Cache 500 recent pages
    WorkerPoolSize:  8,                  // 8 workers for batch processing
}

// ❌ Bad: Using unlimited or excessive values
config := html.Config{
    MaxInputSize:    1024 * 1024 * 1024, // 1GB - too large
    MaxCacheEntries: 1000000,            // 1M entries - excessive memory
}
```

## Support

- **Documentation**: See `docs/` directory and [README.md](README.md)
- **Examples**: See `examples/` directory
- **Tests**: Run `go test -v ./...`
- **Issues**: Report on GitHub

## License

MIT License - See [LICENSE](LICENSE) file for details.
