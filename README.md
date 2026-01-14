# HTML Library

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://golang.org)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/cybergodev/html.svg)](https://pkg.go.dev/github.com/cybergodev/html)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Performance](https://img.shields.io/badge/performance-high%20performance-green.svg)](https://github.com/cybergodev/json)
[![Thread Safe](https://img.shields.io/badge/thread%20safe-yes-brightgreen.svg)](https://github.com/cybergodev/json)


**A Go library for intelligent HTML content extraction.** Compatible with `golang.org/x/net/html` — use it as a drop-in replacement, plus get enhanced content extraction features.

#### **[📖 中文文档](README_zh-CN.md)** - User guide

## ✨ Core Features

### 🎯 Content Extraction
- **Article Detection**: Identifies main content using scoring algorithms (text density, link density, semantic tags)
- **Smart Text Extraction**: Preserves structure, handles newlines, calculates word count and reading time
- **Media Extraction**: Images, videos, audio with metadata (URL, dimensions, alt text, type detection)
- **Link Analysis**: External/internal detection, nofollow attributes, anchor text extraction

### ⚡ Performance
- **Content-Addressable Caching**: SHA256-based keys with TTL and LRU eviction
- **Batch Processing**: Parallel extraction with configurable worker pools
- **Thread-Safe**: Concurrent use without external synchronization
- **Resource Limits**: Configurable input size, nesting depth, and timeout protection

### 📖 Use Cases
- 📰 **News Aggregators**: Extract article content from news sites
- 🤖 **Web Scrapers**: Get structured data from HTML pages
- 📝 **Content Management**: Convert HTML to Markdown or other formats
- 🔍 **Search Engines**: Index main content without navigation/ads
- 📊 **Data Analysis**: Extract and analyze web content at scale
- 📱 **RSS/Feed Generators**: Create feeds from HTML content
- 🎓 **Documentation Tools**: Convert HTML docs to other formats

---

## 📦 Installation

```bash
go get github.com/cybergodev/html
```

---

## ⚡ 5-Minute Quick Start

```go
import "github.com/cybergodev/html"

// Extract clean text from HTML
htmlContent, _ := html.ExtractText(`
    <html>
        <nav>Navigation</nav>
        <article><h1>Hello World</h1><p>Content here...</p></article>
        <footer>Footer</footer>
    </html>
`)
fmt.Println(htmlContent) // "Hello World\nContent here..."
```

**That's it!** The library automatically:
- Removes navigation, footers, ads
- Extracts main content
- Cleans up whitespace

---

## 🚀 Quick Guide

### One-Liner Functions

Just want to get something done? Use these package-level functions:

```go
// Extract text only
text, _ := html.ExtractText(htmlContent)

// Extract everything
result, _ := html.Extract(htmlContent)
fmt.Println(result.Title)     // Hello World
fmt.Println(result.Text)      // Content here...
fmt.Println(result.WordCount) // 4

// Extract only specific elements
title, err := html.ExtractTitle(htmlContent)
images, err := html.ExtractImages(htmlContent)
links, err := html.ExtractLinks(htmlContent)

// Convert formats
markdown, err := html.ExtractToMarkdown(htmlContent)
jsonData, err := html.ExtractToJSON(htmlContent)

// Content analysis
wordCount, err := html.GetWordCount(htmlContent)
readingTime, err := html.GetReadingTime(htmlContent)
summary, err := html.Summarize(htmlContent, 50) // max 50 words
```

**When to use:** Simple scripts, one-off tasks, quick prototyping

---

### Basic Processor Usage

Need more control, Create a processor:

```go
processor := html.NewWithDefaults()
defer processor.Close()

// Extract with defaults
result, err := processor.ExtractWithDefaults(htmlContent)

// Extract from file
result, err = processor.ExtractFromFile("page.html", html.DefaultExtractConfig())

// Batch processing
htmlContents := []string{html1, html2, html3}
results, err := processor.ExtractBatch(htmlContents, html.DefaultExtractConfig())
```

**When to use:** Multiple extractions, processing many files, web scrapers

---

### Custom Configuration

Fine-tune what gets extracted:

```go
config := html.ExtractConfig{
    ExtractArticle:    true,   // Auto-detect main content
    PreserveImages:    true,   // Extract image metadata
    PreserveLinks:     true,   // Extract link metadata
    PreserveVideos:    false,  // Skip videos
    PreserveAudios:    false,  // Skip audio
    InlineImageFormat: "none", // Options: "none", "placeholder", "markdown", "html"
}

processor := html.NewWithDefaults()
defer processor.Close()

result, err := processor.Extract(htmlContent, config)
```

**When to use:** Specific extraction needs, format conversion, custom output

---

### Advanced Features

#### Custom Processor Configuration

```go
config := html.Config{
    MaxInputSize:       10 * 1024 * 1024, // 10MB limit
    ProcessingTimeout:  30 * time.Second,
    MaxCacheEntries:    500,
    CacheTTL:           30 * time.Minute,
    WorkerPoolSize:     8,
    EnableSanitization: true,  // Remove <script>, <style> tags
    MaxDepth:           50,    // Prevent deep nesting attacks
}

processor, err := html.New(config)
defer processor.Close()
```

#### Link Extraction

```go
// Extract all resource links
links, err := html.ExtractAllLinks(htmlContent)

// Group by type
byType := html.GroupLinksByType(links)
cssLinks := byType["css"]
jsLinks := byType["js"]
images := byType["image"]

// Advanced configuration
processor := html.NewWithDefaults()
linkConfig := html.LinkExtractionConfig{
    BaseURL:              "https://example.com",
    ResolveRelativeURLs:  true,
    IncludeImages:        true,
    IncludeVideos:        true,
    IncludeCSS:           true,
    IncludeJS:            true,
}
links, err = processor.ExtractAllLinks(htmlContent, linkConfig)
```

#### Caching & Statistics

```go
processor := html.NewWithDefaults()
defer processor.Close()

// Automatic caching enabled
result1, err := processor.ExtractWithDefaults(htmlContent)
result2, err := processor.ExtractWithDefaults(htmlContent) // Cache hit!

// Check performance
stats := processor.GetStatistics()
fmt.Printf("Cache hits: %d/%d\n", stats.CacheHits, stats.TotalProcessed)

// Clear cache if needed
processor.ClearCache()
```

#### Configuration Presets

```go
processor := html.NewWithDefaults()
defer processor.Close()

// RSS feed generation
result, err := processor.Extract(htmlContent, html.ConfigForRSS())

// Summary generation (text only)
result, err = processor.Extract(htmlContent, html.ConfigForSummary())

// Search indexing (all metadata)
result, err = processor.Extract(htmlContent, html.ConfigForSearchIndex())

// Markdown output
result, err = processor.Extract(htmlContent, html.ConfigForMarkdown())
```

**When to use:** Production applications, performance optimization, specific use cases

---

## 📖 Common Recipes

Copy-paste solutions for common tasks:

### Extract Article Text (Clean)

```go
text, err := html.ExtractText(htmlContent)
// Returns clean text without navigation/ads
```

### Extract with Images

```go
result, err := html.Extract(htmlContent)
for _, img := range result.Images {
    fmt.Printf("Image: %s (alt: %s)\n", img.URL, img.Alt)
}
```

### Convert to Markdown

```go
markdown, err := html.ExtractToMarkdown(htmlContent)
// Images become: ![alt](url)
```

### Extract All Links

```go
links, err := html.ExtractAllLinks(htmlContent)
for _, link := range links {
    fmt.Printf("%s: %s\n", link.Type, link.URL)
}
```

### Get Reading Time

```go
minutes, err := html.GetReadingTime(htmlContent)
fmt.Printf("Reading time: %.1f min", minutes)
```

### Batch Process Files

```go
processor := html.NewWithDefaults()
defer processor.Close()

files := []string{"page1.html", "page2.html", "page3.html"}
results, err := processor.ExtractBatchFiles(files, html.DefaultExtractConfig())
```

### Create RSS Feed Content

```go
processor := html.NewWithDefaults()
defer processor.Close()

result, err := processor.Extract(htmlContent, html.ConfigForRSS())
// Optimized for RSS: fast, includes images/links, no article detection
```

---

## 🔧 API Quick Reference

### Package-Level Functions

```go
// Extraction
Extract(htmlContent string) (*Result, error)
ExtractText(htmlContent string) (string, error)
ExtractFromFile(path string) (*Result, error)

// Format Conversion
ExtractToMarkdown(htmlContent string) (string, error)
ExtractToJSON(htmlContent string) ([]byte, error)

// Specific Elements
ExtractTitle(htmlContent string) (string, error)
ExtractImages(htmlContent string) ([]ImageInfo, error)
ExtractVideos(htmlContent string) ([]VideoInfo, error)
ExtractAudios(htmlContent string) ([]AudioInfo, error)
ExtractLinks(htmlContent string) ([]LinkInfo, error)
ExtractWithTitle(htmlContent string) (string, string, error)

// Analysis
GetWordCount(htmlContent string) (int, error)
GetReadingTime(htmlContent string) (float64, error)
Summarize(htmlContent string, maxWords int) (string, error)
ExtractAndClean(htmlContent string) (string, error)

// Links
ExtractAllLinks(htmlContent string, baseURL ...string) ([]LinkResource, error)
GroupLinksByType(links []LinkResource) map[string][]LinkResource
```

### Processor Methods

```go
// Creation
NewWithDefaults() *Processor
New(config Config) (*Processor, error)
processor.Close()

// Extraction
processor.Extract(htmlContent string, config ExtractConfig) (*Result, error)
processor.ExtractWithDefaults(htmlContent string) (*Result, error)
processor.ExtractFromFile(path string, config ExtractConfig) (*Result, error)

// Batch
processor.ExtractBatch(contents []string, config ExtractConfig) ([]*Result, error)
processor.ExtractBatchFiles(paths []string, config ExtractConfig) ([]*Result, error)

// Links
processor.ExtractAllLinks(htmlContent string, config LinkExtractionConfig) ([]LinkResource, error)

// Monitoring
processor.GetStatistics() Statistics
processor.ClearCache()
```

### Configuration Presets

```go
DefaultExtractConfig()      ExtractConfig
ConfigForRSS()               ExtractConfig
ConfigForSummary()           ExtractConfig
ConfigForSearchIndex()       ExtractConfig
ConfigForMarkdown()          ExtractConfig
DefaultLinkExtractionConfig() LinkExtractionConfig
```

---

## Result Structure

```go
type Result struct {
    Text           string        // Clean text content
    Title          string        // Page/article title
    Images         []ImageInfo   // Image metadata
    Links          []LinkInfo    // Link metadata
    Videos         []VideoInfo   // Video metadata
    Audios         []AudioInfo   // Audio metadata
    WordCount      int           // Total words
    ReadingTime    time.Duration // Estimated reading time
    ProcessingTime time.Duration // Time taken
}

type ImageInfo struct {
    URL          string  // Image URL
    Alt          string  // Alt text
    Title        string  // Title attribute
    Width        string  // Width attribute
    Height       string  // Height attribute
    IsDecorative bool    // No alt text
}

type LinkInfo struct {
    URL        string  // Link URL
    Text       string  // Anchor text
    IsExternal bool    // External domain
    IsNoFollow bool    // rel="nofollow"
}
```

---

## Examples

See [examples/](examples) directory for complete, runnable code:

| Example | Description |
|---------|-------------|
| [01_quick_start.go](examples/01_quick_start.go) | Quick start with one-liners |
| [02_content_extraction.go](examples/02_content_extraction.go) | Content extraction basics |
| [03_link_extraction.go](examples/03_link_extraction.go) | Link extraction patterns |
| [04_media_extraction.go](examples/04_media_extraction.go) | Media (images/videos/audio) |
| [04_advanced_features.go](examples/04_advanced_features.go) | Advanced features & compatibility |
| [05_advanced_usage.go](examples/05_advanced_usage.go) | Batch processing & performance |
| [06_compatibility.go](examples/06_compatibility.go) | golang.org/x/net/html compatibility |
| [07_convenience_api.go](examples/07_convenience_api.go) | Package-level convenience API |

---

## Compatibility

This library is a **drop-in replacement** for `golang.org/x/net/html`:

```go
// Just change the import
- import "golang.org/x/net/html"
+ import "github.com/cybergodev/html"

// All existing code works
doc, err := html.Parse(reader)
html.Render(writer, doc)
escaped := html.EscapeString("<script>")
```

See [COMPATIBILITY.md](COMPATIBILITY.md) for details.

---

## Thread Safety

The `Processor` is safe for concurrent use:

```go
processor := html.NewWithDefaults()
defer processor.Close()

// Safe to use from multiple goroutines
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        processor.ExtractWithDefaults(htmlContent)
    }()
}
wg.Wait()
```

---

## 🤝 Contributing

Contributions, issue reports, and suggestions are welcome!

## 📄 License

MIT License - See [LICENSE](LICENSE) file for details.

---

**Crafted with care for the Go community** ❤️ | If this project helps you, please give it a ⭐️ Star!
