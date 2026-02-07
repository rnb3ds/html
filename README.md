# HTML Library

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://golang.org)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/cybergodev/html.svg)](https://pkg.go.dev/github.com/cybergodev/html)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Performance](https://img.shields.io/badge/performance-high%20performance-green.svg)](https://github.com/cybergodev/html)
[![Thread Safe](https://img.shields.io/badge/thread%20safe-yes-brightgreen.svg)](https://github.com/cybergodev/html)


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
- **News Aggregators**: Extract article content from news sites
- **Web Scrapers**: Get structured data from HTML pages
- **Content Management**: Convert HTML to Markdown or other formats
- **Search Engines**: Index main content without navigation/ads
- **Data Analysis**: Extract and analyze web content at scale
- **RSS/Feed Generators**: Create feeds from HTML content
- **Documentation Tools**: Convert HTML docs to other formats

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
htmlBytes := []byte(`
    <html>
        <nav>Navigation</nav>
        <article><h1>Hello World</h1><p>Content here...</p></article>
        <footer>Footer</footer>
    </html>
`)
text, _ := html.ExtractText(htmlBytes)
fmt.Println(text) // "Hello World\nContent here..."
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
text, _ := html.ExtractText(htmlBytes)

// Extract everything
result, _ := html.Extract(htmlBytes)
fmt.Println(result.Title)     // Hello World
fmt.Println(result.Text)      // Content here...
fmt.Println(result.WordCount) // 4

// Extract all resource links
links, _ := html.ExtractAllLinks(htmlBytes)

// Convert formats
markdown, _ := html.ExtractToMarkdown(htmlBytes)
jsonData, _ := html.ExtractToJSON(htmlBytes)
```

**When to use:** Simple scripts, one-off tasks, quick prototyping

---

### Basic Processor Usage

Need more control? Create a processor:

```go
// Create processor with default configuration
processor, err := html.New()
if err != nil {
    log.Fatal(err)
}
defer processor.Close()

// Extract with defaults
result, _ := processor.ExtractWithDefaults(htmlBytes)

// Extract from file
result, _ = processor.ExtractFromFile("page.html", html.DefaultExtractConfig())

// Batch processing
htmlContents := [][]byte{html1, html2, html3}
results, _ := processor.ExtractBatch(htmlContents, html.DefaultExtractConfig())
```

**When to use:** Multiple extractions, processing many files, web scrapers

---

### Custom Configuration

Fine-tune what gets extracted:

```go
config := html.ExtractConfig{
    ExtractArticle:    true,       // Auto-detect main content
    PreserveImages:    true,       // Extract image metadata
    PreserveLinks:     true,       // Extract link metadata
    PreserveVideos:    false,      // Skip videos
    PreserveAudios:    false,      // Skip audio
    InlineImageFormat: "none",     // Options: "none", "placeholder", "markdown", "html"
    TableFormat:       "markdown", // Options: "markdown", "html"
    Encoding:          "",         // Auto-detect from meta tags, or specify: "utf-8", "windows-1252", etc.
}

processor, err := html.New()
if err != nil {
    log.Fatal(err)
}
defer processor.Close()

result, _ := processor.Extract(htmlBytes, config)
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

processor, _ := html.New(config)
defer processor.Close()
```

#### Link Extraction

```go
// Extract all resource links
links, _ := html.ExtractAllLinks(htmlBytes)

// Group by type
byType := html.GroupLinksByType(links)
cssLinks := byType["css"]
jsLinks := byType["js"]
images := byType["image"]

// Advanced configuration
processor, err := html.New()
if err != nil {
    log.Fatal(err)
}
linkConfig := html.LinkExtractionConfig{
    BaseURL:               "https://example.com",
    ResolveRelativeURLs:   true,
    IncludeImages:         true,
    IncludeVideos:         true,
    IncludeAudios:         true,
    IncludeCSS:            true,
    IncludeJS:             true,
    IncludeContentLinks:   true,
    IncludeExternalLinks:  true,
    IncludeIcons:          true,
}
links, _ = processor.ExtractAllLinks(htmlBytes, linkConfig)
```

#### Caching & Statistics

```go
processor, err := html.New()
if err != nil {
    log.Fatal(err)
}
defer processor.Close()

// Automatic caching enabled
result1, _ := processor.ExtractWithDefaults(htmlBytes)
result2, _ := processor.ExtractWithDefaults(htmlBytes) // Cache hit!

// Check performance
stats := processor.GetStatistics()
fmt.Printf("Cache hits: %d/%d\n", stats.CacheHits, stats.TotalProcessed)

// Clear cache (preserves statistics)
processor.ClearCache()

// Reset statistics (preserves cache entries)
processor.ResetStatistics()
```

**When to use:** Production applications, performance optimization, specific use cases

---

## 📖 Common Recipes

Copy-paste solutions for common tasks:

### Extract Article Text (Clean)

```go
text, _ := html.ExtractText(htmlBytes)
// Returns clean text without navigation/ads
```

### Extract with Images

```go
result, _ := html.Extract(htmlBytes)
for _, img := range result.Images {
    fmt.Printf("Image: %s (alt: %s)\n", img.URL, img.Alt)
}
```

### Convert to Markdown

```go
markdown, _ := html.ExtractToMarkdown(htmlBytes)
// Images become: ![alt](url)
```

### Extract All Links

```go
links, _ := html.ExtractAllLinks(htmlBytes)
for _, link := range links {
    fmt.Printf("%s: %s\n", link.Type, link.URL)
}
```

### Get Reading Time

```go
result, _ := html.Extract(htmlBytes)
minutes := result.ReadingTime.Minutes()
fmt.Printf("Reading time: %.1f min", minutes)
```

### Batch Process Files

```go
processor, err := html.New()
if err != nil {
    log.Fatal(err)
}
defer processor.Close()

files := []string{"page1.html", "page2.html", "page3.html"}
results, _ := processor.ExtractBatchFiles(files, html.DefaultExtractConfig())
```

---

## 🔧 API Quick Reference

### Package-Level Functions

```go
// Extraction
html.Extract(htmlBytes []byte, configs ...ExtractConfig) (*Result, error)
html.ExtractText(htmlBytes []byte) (string, error)
html.ExtractFromFile(filePath string, configs ...ExtractConfig) (*Result, error)

// Format Conversion
html.ExtractToMarkdown(htmlBytes []byte) (string, error)
html.ExtractToJSON(htmlBytes []byte) ([]byte, error)

// Links
html.ExtractAllLinks(htmlBytes []byte, configs ...LinkExtractionConfig) ([]LinkResource, error)
html.GroupLinksByType(links []LinkResource) map[string][]LinkResource
```

### Processor Methods

```go
// Creation
processor, err := html.New()
// or with custom config:
processor, err := html.New(config)
defer processor.Close()

// Extraction
processor.Extract(htmlBytes []byte, config ExtractConfig) (*Result, error)
processor.ExtractWithDefaults(htmlBytes []byte) (*Result, error)
processor.ExtractFromFile(filePath string, config ExtractConfig) (*Result, error)

// Batch
processor.ExtractBatch(contents [][]byte, config ExtractConfig) ([]*Result, error)
processor.ExtractBatchFiles(paths []string, config ExtractConfig) ([]*Result, error)

// Links
processor.ExtractAllLinks(htmlBytes []byte, config LinkExtractionConfig) ([]LinkResource, error)

// Monitoring
processor.GetStatistics() Statistics
processor.ClearCache()
processor.ResetStatistics()
```

### Configuration Functions

```go
// Processor configuration
html.DefaultConfig()            Config

// Extraction configuration
html.DefaultExtractConfig()           ExtractConfig

// Link extraction configuration
html.DefaultLinkExtractionConfig()           LinkExtractionConfig
```

**Default values for `DefaultConfig()`:**
```go
Config{
    MaxInputSize:       50 * 1024 * 1024, // 50MB
    MaxCacheEntries:    2000,
    CacheTTL:           1 * time.Hour,
    WorkerPoolSize:     4,
    EnableSanitization: true,
    MaxDepth:           500,
    ProcessingTimeout:  30 * time.Second,
}
```

**Default values for `DefaultExtractConfig()`:**
```go
ExtractConfig{
    ExtractArticle:    true,
    PreserveImages:    true,
    PreserveLinks:     true,
    PreserveVideos:    true,
    PreserveAudios:    true,
    InlineImageFormat: "none",
    TableFormat:       "markdown",
    Encoding:          "", // Auto-detect
}
```

**Default values for `DefaultLinkExtractionConfig()`:**
```go
LinkExtractionConfig{
    ResolveRelativeURLs:  true,  // Convert relative URLs to absolute
    BaseURL:              "",    // Base URL for resolution (empty = auto-detect)
    IncludeImages:        true,  // Extract image links
    IncludeVideos:        true,  // Extract video links
    IncludeAudios:        true,  // Extract audio links
    IncludeCSS:           true,  // Extract CSS links
    IncludeJS:            true,  // Extract JavaScript links
    IncludeContentLinks:  true,  // Extract content links
    IncludeExternalLinks: true,  // Extract external domain links
    IncludeIcons:         true,  // Extract favicon/icon links
}
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
    ReadingTime    time.Duration // Estimated reading time (JSON: reading_time_ms in milliseconds)
    ProcessingTime time.Duration // Time taken (JSON: processing_time_ms in milliseconds)
}

type ImageInfo struct {
    URL          string  // Image URL
    Alt          string  // Alt text
    Title        string  // Title attribute
    Width        string  // Width attribute
    Height       string  // Height attribute
    IsDecorative bool    // No alt text
    Position     int     // Position in document
}

type LinkInfo struct {
    URL        string  // Link URL
    Text       string  // Anchor text
    Title      string  // Title attribute
    IsExternal bool    // External domain
    IsNoFollow bool    // rel="nofollow"
}

type VideoInfo struct {
    URL      string  // Video URL
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

type LinkResource struct {
    URL   string  // Resource URL
    Title string  // Resource title
    Type  string  // Resource type: css, js, image, video, audio, icon, link, or media
}
```

### Statistics Structure

```go
type Statistics struct {
    TotalProcessed     int64         // Total number of extractions performed
    CacheHits          int64         // Number of times cache was hit
    CacheMisses        int64         // Number of times cache was missed
    ErrorCount         int64         // Number of errors encountered
    AverageProcessTime time.Duration // Average processing time per extraction
}
```

---

## 🔒 Security Features

The library includes built-in security protections:

### HTML Sanitization
- **Dangerous Tag Removal**: `<script>`, `<style>`, `<noscript>`, `<iframe>`, `<embed>`, `<object>`, `<form>`, `<input>`, `<button>`
- **Event Handler Removal**: All `on*` attributes (onclick, onerror, onload, etc.)
- **Dangerous Protocol Blocking**: `javascript:`, `vbscript:`, `data:` (except safe media types)
- **XSS Prevention**: Comprehensive sanitization to prevent cross-site scripting

### Input Validation
- **Size Limits**: Configurable `MaxInputSize` prevents memory exhaustion
- **Depth Limits**: `MaxDepth` prevents stack overflow from deeply nested HTML
- **Timeout Protection**: `ProcessingTimeout` prevents hanging on malformed input
- **Path Traversal Protection**: `ExtractFromFile` validates file paths to prevent directory traversal attacks

### Data URL Security
Only safe media type data URLs are allowed:
- **Allowed**: `data:image/*`, `data:font/*`, `data:application/pdf`
- **Blocked**: `data:text/html`, `data:text/javascript`, `data:text/plain`

---

## Performance Benchmarks

Based on `benchmark_test.go`:

| Operation | Performance | Notes |
|-----------|-------------|-------|
| Text Extraction | ~500ns per HTML document | Fast text extraction |
| Link Extraction | ~2μs per HTML document | With metadata extraction |
| Full Extraction | ~5μs per HTML document | With all features enabled |
| Cache Hit | ~100ns | Near-instant for cached content |

**Caching Benefits:**
- **SHA256-based keys**: Content-addressable caching
- **TTL Support**: Configurable cache expiration
- **LRU Eviction**: Automatic cache management with doubly-linked list
- **Thread-Safe**: Concurrent access without external locks

---

See [examples/](examples) directory for complete, runnable code:

| Example                                                       | Description                          |
|---------------------------------------------------------------|--------------------------------------|
| [01_quick_start.go](examples/01_quick_start.go)               | Quick start with one-liners          |
| [02_content_extraction.go](examples/02_content_extraction.go) | Content extraction basics            |
| [03_media_and_links.go](examples/03_media_and_links.go)       | Media and link extraction            |
| [04_advanced_usage.go](examples/04_advanced_usage.go)         | Advanced features & batch processing |
| [05_output_formats.go](examples/05_output_formats.go)         | JSON and Markdown output formats     |
| [06_error_handling.go](examples/06_error_handling.go)         | Error handling patterns              |
| [07_real_world.go](examples/07_real_world.go)                 | Real-world use cases                 |
| [08_compatibility.go](examples/08_compatibility.go)           | golang.org/x/net/html compatibility  |

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

The library re-exports all commonly used types, constants, and functions from `golang.org/x/net/html`:
- **Types**: `Node`, `NodeType`, `Token`, `Attribute`, `Tokenizer`, `ParseOption`
- **Constants**: All `NodeType` and `TokenType` constants
- **Functions**: `Parse`, `ParseFragment`, `ParseWithOptions`, `ParseFragmentWithOptions`, `Render`, `EscapeString`, `UnescapeString`, `NewTokenizer`, `NewTokenizerFragment`, `ParseOptionEnableScripting`

---

## Thread Safety

The `Processor` is safe for concurrent use:

```go
processor, err := html.New()
if err != nil {
    log.Fatal(err)
}
defer processor.Close()

// Safe to use from multiple goroutines
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        processor.ExtractWithDefaults(htmlBytes)
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

---

## Error Handling

The library provides specific error types for different failure scenarios:

```go
var (
    ErrInputTooLarge     = errors.New("html: input size exceeds maximum")
    ErrInvalidHTML       = errors.New("html: invalid HTML content")
    ErrInvalidConfig     = errors.New("html: invalid configuration")
    ErrProcessorClosed   = errors.New("html: processor closed")
    ErrFileNotFound      = errors.New("html: file not found")
    ErrInvalidFilePath   = errors.New("html: invalid file path")
    ErrMaxDepthExceeded  = errors.New("html: max depth exceeded")
    ErrProcessingTimeout = errors.New("html: processing timeout")
)
```

### Error Handling Best Practices

```go
result, err := html.Extract(htmlBytes)
if err != nil {
    if errors.Is(err, html.ErrInputTooLarge) {
        // Handle oversized input
    } else if errors.Is(err, html.ErrInvalidHTML) {
        // Handle malformed HTML
    } else if errors.Is(err, html.ErrProcessorClosed) {
        // Handle closed processor
    } else {
        // Handle other errors
        log.Printf("Extraction failed: %v", err)
    }
    return
}
```

---

## Character Encoding Support

The library automatically detects and converts content from 15+ character encodings:

### Supported Encodings

**Unicode:**
- UTF-8, UTF-16 LE, UTF-16 BE

**Western European:**
- Windows-1252, ISO-8859-1 through ISO-8859-16

**East Asian:**
- GBK, Big5, Shift_JIS, EUC-JP, ISO-2022-JP, EUC-KR

### Encoding Detection

The library uses a three-tier detection strategy:
1. **BOM Detection**: Byte Order Mark for UTF-8/UTF-16
2. **Meta Tag Detection**: HTML `<meta charset>` and `http-equiv` headers
3. **Smart Detection**: Statistical analysis with confidence scoring

### Manual Encoding Override

```go
config := html.ExtractConfig{
    Encoding: "windows-1252", // Force specific encoding
}
result, _ := html.Extract(htmlBytes, config)
```

---

## Recent Improvements

### Performance & Quality (2026-02-07)

- ✅ **Fixed LRU Cache Bug**: Implemented proper doubly-linked list for correct eviction
- ✅ **Optimized String Operations**: Reduced redundant ToLower conversions
- ✅ **Lazy Regex Compilation**: Faster startup with sync.Once
- ✅ **Improved Statistics**: Added ResetStatistics() method
- ✅ **Unified URL Validation**: Single source of truth for validation

### Test Suite Optimization (2026-02-07)

- ✅ **87.1% Coverage**: Up from 81.7% (+6.6% improvement)
- ✅ **Removed Redundancy**: Eliminated duplicate tests
- ✅ **Better Organization**: Consolidated and structured tests
- ✅ **Comprehensive Docs**: Created testing strategy guide

---
