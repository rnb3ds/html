# HTML Library - intelligent HTML content extraction

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/cybergodev/html.svg)](https://pkg.go.dev/github.com/cybergodev/html)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Security](https://img.shields.io/badge/Security-Production%20Ready-green.svg)](SECURITY.md)

**Production-grade Go library for intelligent HTML content extraction.** 100% compatible with `golang.org/x/net/html` — use it as a drop-in replacement, plus get powerful content extraction features.

#### **[📖 中文文档](README_zh-CN.md)** - User guide

## ✨ Key Features

### 🎯 Intelligent Content Extraction
- **Article Detection**: Identifies main content using scoring algorithms (text density, link density, semantic tags)
- **Smart Text Extraction**: Preserves structure, handles newlines, calculates word count and reading time
- **Media Extraction**: Images, videos, audio with full metadata (URL, dimensions, alt text, type detection)
- **Link Analysis**: External/internal detection, nofollow attributes, anchor text extraction

### 🚀 Production-Ready Performance
- **Content-Addressable Caching**: SHA256-based keys with TTL and LRU eviction
- **Batch Processing**: Parallel extraction with configurable worker pools
- **Thread-Safe**: Concurrent use without external synchronization 
- **Resource Limits**: Configurable input size, nesting depth, and timeout protection

### 📦 Zero Bloat

- **Single Dependency**: Only `golang.org/x/net/html` (no bloated dependency tree)
- **Minimal API Surface**: Simple, focused, easy to learn

### 🎯 Use Cases
- 📰 **News Aggregators**: Extract clean article content from various news sites
- 🤖 **Web Scrapers**: Get structured data from HTML pages efficiently
- 📝 **Content Management**: Convert HTML to Markdown or other formats
- 🔍 **Search Engines**: Index main content without navigation/ads noise
- 📊 **Data Analysis**: Extract and analyze web content at scale
- 📱 **RSS/Feed Generators**: Create feeds from HTML content
- 🎓 **Documentation Tools**: Convert HTML docs to other formats


## 📥 Installation

```bash
go get github.com/cybergodev/html
```

## 🚀 Quick Start

### Intelligent Content Extraction

Extract clean, structured content from complex HTML:

```go
import "github.com/cybergodev/html"

processor := html.NewWithDefaults()
defer processor.Close()

htmlContent := `
    <html>
    <body>
        <nav>Skip this navigation</nav>
        <article>
            <h1>10 Tips for Better Go Code</h1>
            <p>Go is a powerful language that emphasizes simplicity...</p>
            <img src="diagram.png" alt="Architecture Diagram" width="800">
            <p>The key principles include...</p>
        </article>
        <aside>Advertisement</aside>
    </body>
    </html>
`

result, err := processor.ExtractWithDefaults(htmlContent)
if err != nil {
    panic(err)
}

// Extracted content (navigation and ads removed automatically)
fmt.Println("Title:", result.Title)              // "10 Tips for Better Go Code"
fmt.Println("Text:", result.Text)                // Clean article text only
fmt.Println("Word Count:", result.WordCount)     // 8
fmt.Println("Reading Time:", result.ReadingTime) // 2.4s
fmt.Println("Images:", len(result.Images))       // 1

// Image metadata
for _, img := range result.Images {
    fmt.Printf("Image: %s (%s x %s)\n", img.URL, img.Width, img.Height)
    fmt.Printf("Alt: %s\n", img.Alt)
}
```

## 🎯 Core Features

### 1. Intelligent Article Detection

Automatically extracts main content while removing noise:

```go
processor := html.NewWithDefaults()
defer processor.Close()

// Complex page with navigation, ads, sidebars
htmlContent := `
    <html>
    <nav>Site Navigation</nav>
    <aside>Sidebar Ads</aside>
    <article>
        <h1>Main Article</h1>
        <p>This is the actual content users want to read...</p>
    </article>
    <footer>Footer Links</footer>
    </html>
`

config := html.ExtractConfig{
    ExtractArticle: true,  // Enable smart content detection
}

result, _ := processor.Extract(htmlContent, config)
// result.Text contains ONLY the article content
// Navigation, ads, sidebar, and footer are automatically removed
```

### 2. Rich Media Extraction

Extract all media with complete metadata:

```go
result, _ := processor.ExtractWithDefaults(htmlContent)

// Images with full metadata
for _, img := range result.Images {
    fmt.Printf("URL: %s\n", img.URL)
    fmt.Printf("Alt: %s\n", img.Alt)
    fmt.Printf("Size: %s x %s\n", img.Width, img.Height)
    fmt.Printf("Decorative: %v\n", img.IsDecorative)
}

// Videos URLs
for _, video := range result.Videos {
    fmt.Printf("Video: %s (type: %s)\n", video.URL, video.Type)
}

// Audio files
for _, audio := range result.Audios {
    fmt.Printf("Audio: %s (type: %s)\n", audio.URL, audio.Type)
}

// Links with analysis
for _, link := range result.Links {
    fmt.Printf("Link: %s -> %s\n", link.Text, link.URL)
    fmt.Printf("External: %v, NoFollow: %v\n", link.IsExternal, link.IsNoFollow)
}
```

### 3. Inline Image Formatting

Control how images appear in extracted text:

```go
htmlContent := `
    <article>
        <p>Introduction paragraph.</p>
        <img src="diagram.png" alt="System Architecture">
        <p>As shown in the diagram above...</p>
    </article>
`

// Markdown format
config := html.ExtractConfig{
    InlineImageFormat: "markdown",
}
result, _ := processor.Extract(htmlContent, config)
// Output: "Introduction paragraph.\n![System Architecture](diagram.png)\nAs shown..."

// HTML format
config.InlineImageFormat = "html"
result, _ = processor.Extract(htmlContent, config)
// Output: "Introduction paragraph.\n<img src=\"diagram.png\" alt=\"System Architecture\">\nAs shown..."

// Placeholder format
config.InlineImageFormat = "placeholder"
result, _ = processor.Extract(htmlContent, config)
// Output: "Introduction paragraph.\n[IMAGE:1]\nAs shown..."
```

**Formats:**
- `none`: Remove images from text (default)
- `placeholder`: Insert `[IMAGE:1]`, `[IMAGE:2]`, etc.
- `markdown`: Insert `![alt](url)` for Markdown conversion
- `html`: Insert `<img>` tags for HTML reconstruction

### 4. Comprehensive Link Extraction

Extract all types of resource links with automatic URL resolution:

```go
htmlContent := `
    <!DOCTYPE html>
    <html>
    <head>
        <base href="https://example.com/">
        <link rel="stylesheet" href="css/main.css">
        <script src="js/app.js"></script>
        <link rel="icon" href="/favicon.ico">
    </head>
    <body>
        <a href="/about">About</a>
        <a href="https://external.com">External</a>
        <img src="images/hero.jpg" alt="Hero">
        <video src="videos/demo.mp4"></video>
        <audio src="audio/music.mp3"></audio>
        <iframe src="https://youtube.com/embed/abc123"></iframe>
    </body>
    </html>
`

// Simple extraction (convenience function)
links, err := html.ExtractAllLinks(htmlContent)
if err != nil {
    log.Fatal(err)
}

// Group links by type using convenience function
linksByType := html.GroupLinksByType(links)

// Access specific types directly
cssLinks := linksByType["css"]
jsLinks := linksByType["js"]
contentLinks := linksByType["link"]
images := linksByType["image"]
```

**Advanced Configuration:**
```go
processor := html.NewWithDefaults()
defer processor.Close()

config := html.LinkExtractionConfig{
    ResolveRelativeURLs:  true,  // Auto-resolve relative URLs
    BaseURL:              "",    // Auto-detect or specify base URL
    IncludeImages:        true,  // Extract image resources
    IncludeVideos:        true,  // Extract video resources  
    IncludeAudios:        true,  // Extract audio resources
    IncludeCSS:           true,  // Extract CSS stylesheets
    IncludeJS:            true,  // Extract JavaScript files
    IncludeContentLinks:  true,  // Extract navigation links
    IncludeExternalLinks: true,  // Extract external domain links
    IncludeIcons:         true,  // Extract favicons and icons
}

links, err := processor.ExtractAllLinks(htmlContent, config)
```

**Features:**
- **Automatic URL Resolution**: Detects base URLs from `<base>` tags, canonical meta, or existing URLs
- **Resource Type Detection**: Images, videos, audio, CSS, JS, content links, external links, icons
- **Smart Deduplication**: Prevents duplicate links in results
- **Domain Classification**: Distinguishes internal vs external links
- **Comprehensive Coverage**: Extracts from all HTML elements including `<link>`, `<script>`, `<img>`, `<video>`, `<audio>`, `<iframe>`, `<embed>`, `<object>`

### 5. Batch Processing

Process multiple documents in parallel with worker pools:

```go
processor := html.NewWithDefaults()
defer processor.Close()

// Process multiple HTML strings
htmlContents := []string{
    "<html><body><h1>Page 1</h1><p>Content 1</p></body></html>",
    "<html><body><h1>Page 2</h1><p>Content 2</p></body></html>",
    "<html><body><h1>Page 3</h1><p>Content 3</p></body></html>",
}

config := html.DefaultExtractConfig()
results, err := processor.ExtractBatch(htmlContents, config)

for i, result := range results {
    if result != nil {
        fmt.Printf("Page %d: %s (%d words)\n", i+1, result.Title, result.WordCount)
    }
}

// Or process files directly
filePaths := []string{"page1.html", "page2.html", "page3.html"}
results, err = processor.ExtractBatchFiles(filePaths, config)
```

### 6. Performance & Caching

Built-in caching and monitoring:

```go
processor := html.NewWithDefaults()
defer processor.Close()

// Extract content (cached automatically)
result1, _ := processor.ExtractWithDefaults(htmlContent)

// Same content? Instant cache hit
result2, _ := processor.ExtractWithDefaults(htmlContent)

// Check statistics
stats := processor.GetStatistics()
fmt.Printf("Total Processed: %d\n", stats.TotalProcessed)
fmt.Printf("Cache Hits: %d (%.1f%%)\n", stats.CacheHits, 
    float64(stats.CacheHits)/float64(stats.TotalProcessed)*100)
fmt.Printf("Average Time: %v\n", stats.AverageProcessTime)
fmt.Printf("Errors: %d\n", stats.ErrorCount)

// Clear cache if needed
processor.ClearCache()
```

**Caching features:**
- SHA256-based content-addressable keys (collision-resistant)
- TTL-based expiration (default: 1 hour)
- LRU eviction when cache is full
- Thread-safe with minimal lock contention

## ⚙️ Configuration

### Processor Configuration

Customize resource limits and behavior:

```go
config := html.Config{
    MaxInputSize:       50 * 1024 * 1024,   // 50MB max input size
    ProcessingTimeout:  30 * time.Second,   // 30s processing timeout
    MaxCacheEntries:    1000,               // Cache up to 1000 results
    CacheTTL:           time.Hour,          // 1 hour cache TTL
    WorkerPoolSize:     4,                  // 4 parallel workers for batch
    EnableSanitization: true,               // Sanitize HTML input
    MaxDepth:           100,                // Max HTML nesting depth
}

processor, err := html.New(config)
if err != nil {
    log.Fatal(err)
}
defer processor.Close()
```

**Default values** (via `html.NewWithDefaults()`):
- MaxInputSize: 50MB
- ProcessingTimeout: 30s
- MaxCacheEntries: 1000
- CacheTTL: 1 hour
- WorkerPoolSize: 4
- EnableSanitization: true
- MaxDepth: 100

### Extraction Configuration

Control what to extract and how:

```go
config := html.ExtractConfig{
    ExtractArticle:    true,        // Enable intelligent article detection
    PreserveImages:    true,        // Extract image metadata
    PreserveLinks:     true,        // Extract link metadata
    PreserveVideos:    true,        // Extract video metadata
    PreserveAudios:    true,        // Extract audio metadata
    InlineImageFormat: "markdown",  // none, placeholder, markdown, html
}

result, err := processor.Extract(htmlContent, config)
```

**Quick defaults:**
```go
// All features enabled, no inline images
config := html.DefaultExtractConfig()

// Or use the shorthand
result, _ := processor.ExtractWithDefaults(htmlContent)
```

## 📚 API Reference

### Processor Methods

```go
// Create processor
processor := html.NewWithDefaults()
processor, err := html.New(config)
defer processor.Close()

// Extract content
result, err := processor.Extract(htmlContent, config)
result, err := processor.ExtractWithDefaults(htmlContent)
result, err := processor.ExtractFromFile("page.html", config)

// Batch processing
results, err := processor.ExtractBatch(htmlContents, config)
results, err := processor.ExtractBatchFiles(filePaths, config)

// Monitoring
stats := processor.GetStatistics()
processor.ClearCache()
```

### Result Structure

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
```

### Media Types

```go
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

### Statistics

```go
type Statistics struct {
    TotalProcessed     int64         // Total extractions performed
    CacheHits          int64         // Cache hits
    CacheMisses        int64         // Cache misses
    ErrorCount         int64         // Total errors
    AverageProcessTime time.Duration // Average processing time
}
```

## 💡 Usage Examples

See the [examples/](examples) directory for complete, runnable examples:

- **[01_quick_start.go](examples/01_quick_start.go)** - Quick start with convenience functions
- **[02_content_extraction.go](examples/02_content_extraction.go)** - Content extraction with article detection and inline images
- **[03_link_extraction.go](examples/03_link_extraction.go)** - Comprehensive link extraction with URL resolution
- **[04_media_extraction.go](examples/04_media_extraction.go)** - Extract images, videos, audio, and links with metadata
- **[05_advanced_usage.go](examples/05_advanced_usage.go)** - Advanced features: custom config, batch processing, caching, concurrency
- **[06_compatibility.go](examples/06_compatibility.go)** - 100% compatibility with golang.org/x/net/html

## 🔒 Thread Safety

The `Processor` is **safe for concurrent use** by multiple goroutines without external synchronization:

```go
processor := html.NewWithDefaults()
defer processor.Close()

// Safe to call from multiple goroutines
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        result, _ := processor.ExtractWithDefaults(htmlContent)
        fmt.Printf("Goroutine %d: %s\n", id, result.Title)
    }(i)
}
wg.Wait()
```

## ⚡ Performance Tips

1. **Reuse Processor**: Create once, use many times (avoid per-request creation)
2. **Enable Caching**: Default settings work well (1000 entries, 1 hour TTL)
3. **Batch Processing**: Use `ExtractBatch()` for multiple documents (parallel workers)
4. **Tune Limits**: Adjust `MaxInputSize` based on your content (default: 50MB)
5. **Worker Pool**: Set `WorkerPoolSize` to match CPU cores (default: 4)

## 🔄 Compatibility with golang.org/x/net/html

### Standard HTML Parsing (100% Compatible)

This library is a **100% compatible drop-in replacement** for `golang.org/x/net/html`:

```go
// Before
import "golang.org/x/net/html"

// After  
import "github.com/cybergodev/html"

// Parse HTML documents
doc, err := html.Parse(strings.NewReader(htmlContent))

// Render to HTML
html.Render(os.Stdout, doc)

// Escape/Unescape HTML entities
escaped := html.EscapeString("<script>alert('xss')</script>")
unescaped := html.UnescapeString("&lt;html&gt; &copy; 2024")

// Tokenize HTML
tokenizer := html.NewTokenizer(strings.NewReader("<p>Test</p>"))
```

All `golang.org/x/net/html` APIs work identically — just change the import:

**What's re-exported:**
- All types: `Node`, `Token`, `Tokenizer`, `Attribute`, `NodeType`, `TokenType`
- All functions: `Parse()`, `ParseFragment()`, `Render()`, `EscapeString()`, `UnescapeString()`, `NewTokenizer()`
- All constants: `ElementNode`, `TextNode`, `DocumentNode`, `CommentNode`, `DoctypeNode`, etc.

**Migration cost:** Zero. Just change the import path.

See [COMPATIBILITY.md](COMPATIBILITY.md) for detailed compatibility information.


---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

## 🌟 Star History

If you find this project useful, please consider giving it a star! ⭐

---

**Made with ❤️ by the CyberGoDev team**
