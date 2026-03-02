# HTML Library

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://golang.org)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/cybergodev/html.svg)](https://pkg.go.dev/github.com/cybergodev/html)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Performance](https://img.shields.io/badge/performance-high%20performance-green.svg)](https://github.com/cybergodev/html)
[![Thread Safe](https://img.shields.io/badge/thread%20safe-yes-brightgreen.svg)](https://github.com/cybergodev/html)

**A Go library for intelligent HTML content extraction.** Compatible with `golang.org/x/net/html` — use as a drop-in replacement with enhanced content extraction capabilities.

**[📖 中文文档](README_zh-CN.md)** - Chinese Documentation

## ✨ Core Features

### Content Extraction
- **Article Detection**: Identifies main content using scoring algorithms (text density, link density, semantic tags)
- **Smart Text Extraction**: Preserves structure, handles line breaks, calculates word count and reading time
- **Media Extraction**: Extracts images, videos, audio with metadata (URL, dimensions, alt text, type detection)
- **Link Analysis**: External/internal link detection, nofollow attribute recognition, anchor text extraction

### Performance Advantages
- **Content-Addressed Caching**: FNV-128a based keys with TTL and LRU eviction
- **Batch Processing**: Parallel extraction with configurable Worker Pool and Context support
- **Thread-Safe**: Concurrent use without external synchronization
- **Resource Limits**: Configurable input size, nesting depth, and timeout protection

### Security Features
- **HTML Sanitization**: Removes dangerous tags and attributes
- **Audit Logging**: Tracks security events for compliance requirements
- **Input Validation**: Size limits, depth limits, path traversal protection

### Use Cases
- **News Aggregators**: Extract article content from news websites
- **Web Crawlers**: Fetch structured data from HTML pages
- **Content Management**: Convert HTML to Markdown or other formats
- **Search Engines**: Index main content, excluding navigation and ads
- **Data Analysis**: Extract and analyze web content at scale

---

## 📦 Installation

```bash
go get github.com/cybergodev/html
```

---

## ⚡ 5-Minute Quick Start

```go
package main

import (
    "fmt"
    "github.com/cybergodev/html"
)

func main() {
    // Extract plain text from HTML
    htmlBytes := []byte(`
        <html>
            <nav>Navigation Bar</nav>
            <article><h1>Hello World</h1><p>Content here...</p></article>
            <footer>Footer</footer>
        </html>
    `)
    text, err := html.ExtractText(htmlBytes)
    if err != nil {
        panic(err)
    }
    fmt.Println(text) // "Hello World\nContent here..."
}
```

**That's it!** The library automatically:
- Removes navigation, footers, ads
- Extracts main content
- Cleans up whitespace

---

## 🚀 Quick Guide

### One-Liner Functions

Just want to get things done quickly? Use these package-level functions:

```go
package main

import (
    "fmt"
    "github.com/cybergodev/html"
)

func main() {
    htmlBytes := []byte(`<html><body><h1>Title</h1><p>Content here...</p></body></html>`)

    // Extract text only
    text, _ := html.ExtractText(htmlBytes)

    // Extract all content
    result, _ := html.Extract(htmlBytes)
    fmt.Println(result.Title)     // Title
    fmt.Println(result.Text)      // Content here...
    fmt.Println(result.WordCount) // 2

    // Extract all resource links
    links, _ := html.ExtractAllLinks(htmlBytes)

    // Format conversion
    markdown, _ := html.ExtractToMarkdown(htmlBytes)
    jsonData, _ := html.ExtractToJSON(htmlBytes)
}
```

**Best for:** Simple scripts, one-time tasks, rapid prototyping

---

### Basic Processor Usage

Need more control? Create a Processor:

```go
package main

import (
    "fmt"
    "log"
    "github.com/cybergodev/html"
)

func main() {
    // Create Processor with default configuration
    processor, err := html.New()
    if err != nil {
        log.Fatal(err)
    }
    defer processor.Close()

    htmlBytes := []byte(`<html><body><h1>Title</h1><p>Content</p></body></html>`)

    // Extract with default configuration
    result, _ := processor.Extract(htmlBytes, html.DefaultExtractConfig())

    // Extract from file
    result, _ = processor.ExtractFromFile("page.html", html.DefaultExtractConfig())

    // Batch processing
    htmlContents := [][]byte{htmlBytes, htmlBytes, htmlBytes}
    results, _ := processor.ExtractBatch(htmlContents, html.DefaultExtractConfig())

    fmt.Printf("Processed %d documents\n", len(results))
}
```

**Best for:** Multiple extractions, processing large files, web crawlers

---

### Custom Configuration

Fine-tune what gets extracted:

```go
package main

import (
    "fmt"
    "github.com/cybergodev/html"
)

func main() {
    htmlBytes := []byte(`<html><body><h1>Title</h1><img src="img.jpg"><p>Content</p></body></html>`)

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

    result, _ := html.Extract(htmlBytes, config)
    fmt.Printf("Found %d images\n", len(result.Images))
}
```

**Best for:** Specific extraction needs, format conversion, custom output

---

### Advanced Features

#### Custom Processor Configuration

```go
package main

import (
    "time"
    "github.com/cybergodev/html"
)

func main() {
    // Method 1: Using Config struct
    config := html.Config{
        MaxInputSize:       10 * 1024 * 1024, // 10MB limit
        ProcessingTimeout:  30 * time.Second,
        MaxCacheEntries:    500,
        CacheTTL:           30 * time.Minute,
        WorkerPoolSize:     8,
        EnableSanitization: true,  // Remove <script>, <style> tags
        MaxDepth:           50,    // Prevent deeply nested attacks
    }
    processor, _ := html.New(config)
    defer processor.Close()

    // Method 2: Using high-security preset
    processor2, _ := html.New(html.HighSecurityConfig())
    defer processor2.Close()
}
```

#### Link Extraction

```go
package main

import (
    "fmt"
    "github.com/cybergodev/html"
)

func main() {
    htmlBytes := []byte(`
        <html>
        <head><link rel="stylesheet" href="style.css"></head>
        <body>
            <img src="image.jpg">
            <a href="https://example.com">Link</a>
            <script src="app.js"></script>
        </body>
        </html>
    `)

    // Extract all resource links
    links, _ := html.ExtractAllLinks(htmlBytes)

    // Group by type
    byType := html.GroupLinksByType(links)
    cssLinks := byType["css"]
    jsLinks := byType["js"]
    images := byType["image"]

    fmt.Printf("CSS: %d, JS: %d, Images: %d\n", len(cssLinks), len(jsLinks), len(images))

    // Advanced configuration
    processor, _ := html.New()
    defer processor.Close()

    linkConfig := html.LinkExtractionConfig{
        BaseURL:              "https://example.com",
        ResolveRelativeURLs:  true,
        IncludeImages:        true,
        IncludeVideos:        true,
        IncludeAudios:        true,
        IncludeCSS:           true,
        IncludeJS:            true,
        IncludeContentLinks:  true,
        IncludeExternalLinks: true,
        IncludeIcons:         true,
    }
    links, _ = processor.ExtractAllLinks(htmlBytes, linkConfig)
}
```

#### Caching & Statistics

```go
package main

import (
    "fmt"
    "github.com/cybergodev/html"
)

func main() {
    processor, _ := html.New()
    defer processor.Close()

    htmlBytes := []byte(`<html><body><p>Content</p></body></html>`)

    // Caching is automatically enabled
    processor.Extract(htmlBytes, html.DefaultExtractConfig())
    processor.Extract(htmlBytes, html.DefaultExtractConfig()) // Cache hit!

    // View performance statistics
    stats := processor.GetStatistics()
    fmt.Printf("Cache hits: %d/%d\n", stats.CacheHits, stats.TotalProcessed)

    // Clear cache (preserves statistics)
    processor.ClearCache()

    // Reset statistics (preserves cache entries)
    processor.ResetStatistics()
}
```

**Best for:** Production applications, performance optimization, specific use cases

---

## 📖 Common Examples

Copy-paste solutions for common tasks:

### Extract Article Text (Clean)

```go
text, _ := html.ExtractText(htmlBytes)
// Returns clean text without navigation/ads
```

### Extract from File

```go
// Extract text from file
text, _ := html.ExtractTextFromFile("page.html")

// Extract full result from file
result, _ := html.ExtractFromFile("page.html")

// Extract links from file
links, _ := html.ExtractAllLinksFromFile("page.html")

// Convert file to Markdown
markdown, _ := html.ExtractToMarkdownFromFile("page.html")

// Convert file to JSON
jsonData, _ := html.ExtractToJSONFromFile("page.html")
```

### Extract Content with Images

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
fmt.Printf("Reading time: %.1f minutes", minutes)
```

### Batch Process Files

```go
processor, _ := html.New()
defer processor.Close()

files := []string{"page1.html", "page2.html", "page3.html"}
results, _ := processor.ExtractBatchFiles(files, html.DefaultExtractConfig())
```

### Batch Processing with Context (Cancellable)

```go
package main

import (
    "context"
    "time"
    "github.com/cybergodev/html"
)

func main() {
    processor, _ := html.New()
    defer processor.Close()

    files := []string{"page1.html", "page2.html", "page3.html"}

    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Cancellable processing
    result := processor.ExtractBatchFilesWithContext(ctx, files, html.DefaultExtractConfig())

    fmt.Printf("Success: %d, Failed: %d, Cancelled: %d\n",
        result.Success, result.Failed, result.Cancelled)
}
```

### Using Preset Extraction Configurations

```go
// Text only - create text-only configuration
textConfig := html.ExtractConfig{
    ExtractArticle:    true,
    PreserveImages:    false,
    PreserveLinks:     false,
    PreserveVideos:    false,
    PreserveAudios:    false,
    InlineImageFormat: "none",
}
result, _ := html.Extract(htmlBytes, textConfig)

// Full content - all media, images in markdown format
fullConfig := html.ExtractConfig{
    ExtractArticle:    true,
    PreserveImages:    true,
    PreserveLinks:     true,
    PreserveVideos:    true,
    PreserveAudios:    true,
    InlineImageFormat: "markdown",
}
result, _ = html.Extract(htmlBytes, fullConfig)
```

---

## 🔧 API Quick Reference

### Package-Level Functions

```go
// Extract (from bytes)
html.Extract(htmlBytes []byte, configs ...ExtractConfig) (*Result, error)
html.ExtractText(htmlBytes []byte, configs ...ExtractConfig) (string, error)

// Extract (from file)
html.ExtractFromFile(filePath string, configs ...ExtractConfig) (*Result, error)
html.ExtractTextFromFile(filePath string, configs ...ExtractConfig) (string, error)

// Format conversion (from bytes)
html.ExtractToMarkdown(htmlBytes []byte, configs ...ExtractConfig) (string, error)
html.ExtractToJSON(htmlBytes []byte, configs ...ExtractConfig) ([]byte, error)

// Format conversion (from file)
html.ExtractToMarkdownFromFile(filePath string, configs ...ExtractConfig) (string, error)
html.ExtractToJSONFromFile(filePath string, configs ...ExtractConfig) ([]byte, error)

// Links (from bytes)
html.ExtractAllLinks(htmlBytes []byte, configs ...LinkExtractionConfig) ([]LinkResource, error)

// Links (from file)
html.ExtractAllLinksFromFile(filePath string, configs ...LinkExtractionConfig) ([]LinkResource, error)
html.GroupLinksByType(links []LinkResource) map[string][]LinkResource

// Batch processing
html.ExtractBatch(htmlContents [][]byte, configs ...ExtractConfig) ([]*Result, error)
html.ExtractBatchFiles(filePaths []string, configs ...ExtractConfig) ([]*Result, error)
html.ExtractBatchWithContext(ctx context.Context, htmlContents [][]byte, configs ...ExtractConfig) *BatchResult
html.ExtractBatchFilesWithContext(ctx context.Context, filePaths []string, configs ...ExtractConfig) *BatchResult
```

### Processor Methods
```go
// Creation
processor, err := html.New()
processor, err := html.New(config)                    // Using Config struct
processor, err := html.New(html.HighSecurityConfig()) // Using preset configuration
processor, err := html.New(myScorer)                  // Using custom Scorer
defer processor.Close()

// Extract (from bytes)
processor.Extract(htmlBytes []byte, configs ...ExtractConfig) (*Result, error)
processor.ExtractText(htmlBytes []byte, configs ...ExtractConfig) (string, error)

// Extract (from file)
processor.ExtractFromFile(filePath string, configs ...ExtractConfig) (*Result, error)
processor.ExtractTextFromFile(filePath string, configs ...ExtractConfig) (string, error)

// Format conversion (from bytes)
processor.ExtractToMarkdown(htmlBytes []byte, configs ...ExtractConfig) (string, error)
processor.ExtractToJSON(htmlBytes []byte, configs ...ExtractConfig) ([]byte, error)

// Format conversion (from file)
processor.ExtractToMarkdownFromFile(filePath string, configs ...ExtractConfig) (string, error)
processor.ExtractToJSONFromFile(filePath string, configs ...ExtractConfig) ([]byte, error)

// Links (from bytes)
processor.ExtractAllLinks(htmlBytes []byte, configs ...LinkExtractionConfig) ([]LinkResource, error)

// Links (from file)
processor.ExtractAllLinksFromFile(filePath string, configs ...LinkExtractionConfig) ([]LinkResource, error)

// Batch processing
processor.ExtractBatch(contents [][]byte, configs ...ExtractConfig) ([]*Result, error)
processor.ExtractBatchFiles(paths []string, configs ...ExtractConfig) ([]*Result, error)
processor.ExtractBatchWithContext(ctx context.Context, contents [][]byte, configs ...ExtractConfig) *BatchResult
processor.ExtractBatchFilesWithContext(ctx context.Context, paths []string, configs ...ExtractConfig) *BatchResult

// Monitoring
processor.GetStatistics() Statistics
processor.ClearCache()
processor.ResetStatistics()
processor.GetAuditLog() []AuditEntry
processor.ClearAuditLog()
```

### Configuration Functions

```go
// Processor configuration
html.DefaultConfig() Config        // Standard configuration
html.HighSecurityConfig() Config   // Security-optimized configuration

// Extraction configuration
html.DefaultExtractConfig() ExtractConfig

// Link extraction configuration
html.DefaultLinkExtractionConfig() LinkExtractionConfig

// Audit configuration
html.DefaultAuditConfig() AuditConfig
html.HighSecurityAuditConfig() AuditConfig
```

### Configuration Functions

```go
// Processor configuration
html.DefaultConfig() Config        // Standard configuration
html.HighSecurityConfig() Config   // Security-optimized configuration

// Extraction configuration
html.DefaultExtractConfig() ExtractConfig
html.TextOnlyExtractConfig() ExtractConfig  // Text-only preset
// For full content with markdown images:
// cfg := html.DefaultExtractConfig()
// cfg.InlineImageFormat = "markdown"

// Link extraction configuration
html.DefaultLinkExtractionConfig() LinkExtractionConfig

// Audit configuration
html.DefaultAuditConfig() AuditConfig
html.HighSecurityAuditConfig() AuditConfig
```

### Default Configuration Values

**DefaultConfig():**
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

**HighSecurityConfig():**
```go
Config{
    MaxInputSize:       10 * 1024 * 1024, // 10MB - reduced for security
    MaxCacheEntries:    500,              // Reduced cache size
    CacheTTL:           30 * time.Minute, // Shorter TTL
    WorkerPoolSize:     2,                // Fewer workers
    EnableSanitization: true,
    MaxDepth:           100,              // Reduced depth limit
    ProcessingTimeout:  10 * time.Second, // Shorter timeout
}
```

**DefaultExtractConfig():**
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

**DefaultLinkExtractionConfig():**
```go
LinkExtractionConfig{
    ResolveRelativeURLs:  true,
    BaseURL:              "",    // Auto-detect
    IncludeImages:        true,
    IncludeVideos:        true,
    IncludeAudios:        true,
    IncludeCSS:           true,
    IncludeJS:            true,
    IncludeContentLinks:  true,
    IncludeExternalLinks: true,
    IncludeIcons:         true,
}
```

---

## Result Structures

```go
type Result struct {
    Text           string        `json:"text"`
    Title          string        `json:"title"`
    Images         []ImageInfo   `json:"images,omitempty"`
    Links          []LinkInfo    `json:"links,omitempty"`
    Videos         []VideoInfo   `json:"videos,omitempty"`
    Audios         []AudioInfo   `json:"audios,omitempty"`
    WordCount      int           `json:"word_count"`
    ReadingTime    time.Duration `json:"reading_time_ms"`    // JSON: milliseconds
    ProcessingTime time.Duration `json:"processing_time_ms"` // JSON: milliseconds
}

type ImageInfo struct {
    URL          string `json:"url"`
    Alt          string `json:"alt"`
    Title        string `json:"title"`
    Width        string `json:"width"`
    Height       string `json:"height"`
    IsDecorative bool   `json:"is_decorative"`
    Position     int    `json:"position"`
}

type LinkInfo struct {
    URL        string `json:"url"`
    Text       string `json:"text"`
    Title      string `json:"title"`
    IsExternal bool   `json:"is_external"`
    IsNoFollow bool   `json:"is_nofollow"`
}

type VideoInfo struct {
    URL      string `json:"url"`
    Type     string `json:"type"`
    Poster   string `json:"poster"`
    Width    string `json:"width"`
    Height   string `json:"height"`
    Duration string `json:"duration"`
}

type AudioInfo struct {
    URL      string `json:"url"`
    Type     string `json:"type"`
    Duration string `json:"duration"`
}

type LinkResource struct {
    URL   string
    Title string
    Type  string // "css", "js", "image", "video", "audio", "icon", "link"
}

type BatchResult struct {
    Results    []*Result
    Errors     []error
    Success    int
    Failed     int
    Cancelled  int
}

type Statistics struct {
    TotalProcessed     int64
    CacheHits          int64
    CacheMisses        int64
    ErrorCount         int64
    AverageProcessTime time.Duration
}
```

---

## 🔒 Security Features

This library has built-in security protections:

### HTML Sanitization
- **Dangerous Tag Removal**: `<script>`, `<style>`, `<noscript>`, `<iframe>`, `<embed>`, `<object>`, `<form>`, `<input>`, `<button>`
- **Event Handler Removal**: All `on*` attributes (onclick, onerror, onload, etc.)
- **Dangerous Protocol Blocking**: `javascript:`, `vbscript:`, `data:` (except safe media types)
- **XSS Protection**: Comprehensive sanitization to prevent cross-site scripting attacks

### Input Validation
- **Size Limits**: Configurable `MaxInputSize` prevents memory exhaustion
- **Depth Limits**: `MaxDepth` prevents stack overflow from deeply nested HTML
- **Timeout Protection**: `ProcessingTimeout` prevents hanging on malformed input
- **Path Traversal Protection**: `ExtractFromFile` validates file paths

### Data URL Security
Only safe media type data URLs are allowed:
- **Allowed**: `data:image/*`, `data:font/*`, `data:application/pdf`
- **Blocked**: `data:text/html`, `data:text/javascript`, `data:text/plain`

### High-Security Preset
For applications requiring tighter security, use `html.HighSecurityConfig()`:
- Smaller input size limit (10MB vs 50MB)
- Lower depth limit (100 vs 500)
- Shorter timeout (10s vs 30s)
- Audit logging enabled by default

---

## 🔍 Audit Logging

The library provides comprehensive audit logging capabilities for security compliance:

### Enable Audit Logging

```go
// Method 1: Use HighSecurityConfig (audit enabled by default)
processor, _ := html.New(html.HighSecurityConfig())

// Method 2: Custom configuration with audit enabled
config := html.DefaultConfig()
config.Audit = html.AuditConfig{
    Enabled:            true,
    LogBlockedTags:     true,
    LogBlockedAttrs:    true,
    LogBlockedURLs:     true,
    LogInputViolations: true,
    LogDepthViolations: true,
    LogTimeouts:        true,
    LogEncodingIssues:  true,
    LogPathTraversal:   true,
}
processor, _ := html.New(config)
```

### Get Audit Logs

```go
processor, _ := html.New(html.HighSecurityConfig())
defer processor.Close()

// Process content
processor.Extract(htmlBytes, html.DefaultExtractConfig())

// Get audit entries
entries := processor.GetAuditLog()
for _, entry := range entries {
    fmt.Printf("[%s] %s: %s\n", entry.Level, entry.EventType, entry.Message)
}

// Clear audit log
processor.ClearAuditLog()
```

### Audit Entry Structure

```go
type AuditEntry struct {
    Timestamp time.Time      `json:"timestamp"`
    EventType AuditEventType `json:"event_type"`
    Level     AuditLevel     `json:"level"`
    Message   string         `json:"message"`
    Tag       string         `json:"tag,omitempty"`
    Attribute string         `json:"attribute,omitempty"`
    URL       string         `json:"url,omitempty"`
    InputSize int            `json:"input_size,omitempty"`
    MaxSize   int            `json:"max_size,omitempty"`
    Depth     int            `json:"depth,omitempty"`
    MaxDepth  int            `json:"max_depth,omitempty"`
    Path      string         `json:"path,omitempty"`
    RawValue  string         `json:"raw_value,omitempty"`
}
```

### Custom Audit Sinks

```go
import "github.com/cybergodev/html"

// Write audit logs to file
file, _ := os.Create("audit.log")
fileSink := html.NewWriterAuditSink(file)

// Filter to critical events only
filteredSink := html.NewLevelFilteredSink(fileSink, html.AuditLevelCritical)

// Use custom sink in configuration
config := html.DefaultConfig()
config.Audit = html.AuditConfig{
    Enabled: true,
    Sink:    filteredSink,
}
processor, _ := html.New(config)
```

### Built-in Audit Sinks

| Sink | Description |
|------|-------------|
| `NewLoggerAuditSink()` | Writes to stderr with `[AUDIT]` prefix |
| `NewLoggerAuditSinkWithWriter(w)` | Writes to custom io.Writer |
| `NewWriterAuditSink(w)` | Writes to io.Writer as JSON lines |
| `NewChannelAuditSink(bufferSize)` | Sends to channel for external processing |
| `NewMultiSink(sinks...)` | Combines multiple sinks |
| `NewFilteredSink(sink, filter)` | Filters entries before writing |
| `NewLevelFilteredSink(sink, level)` | Only writes entries at or above specified level |

---

## Example Code

For complete runnable examples, see the [examples/](examples) directory:

| Example | Description |
|---------|-------------|
| [01_quick_start.go](examples/01_quick_start.go) | Quick start (get started in 3 steps) |
| [02_content_extraction.go](examples/02_content_extraction.go) | Content extraction configuration (preset options, custom configuration) |
| [03_links_media.go](examples/03_links_media.go) | Link and media extraction (URL resolution, images/video/audio) |
| [04_output_formats.go](examples/04_output_formats.go) | Output formats (JSON, Markdown, table formats) |
| [05_configuration.go](examples/05_configuration.go) | Configuration and performance (caching, batch processing, concurrency) |
| [06_http_integration.go](examples/06_http_integration.go) | HTTP integration (fetching web pages, concurrent processing) |
| [07_advanced_usage.go](examples/07_advanced_usage.go) | Advanced features (custom scorers, audit system, security configuration) |
| [08_error_handling.go](examples/08_error_handling.go) | Error handling patterns |
| [09_real_world.go](examples/09_real_world.go) | Real-world use cases (blogs, RSS, documentation) |

---

## Compatibility

This library is a **drop-in replacement** for `golang.org/x/net/html`:

```go
// Just change the import
- import "golang.org/x/net/html"
+ import "github.com/cybergodev/html"

// All existing code continues to work
doc, err := html.Parse(reader)
html.Render(writer, doc)
escaped := html.EscapeString("<script>")
```

This library re-exports commonly used types, constants, and functions:
- **Types**: `Node`, `NodeType`, `Token`, `Attribute`, `Tokenizer`, `ParseOption`
- **Constants**: All `NodeType` and `TokenType` constants (`ErrorNode`, `TextNode`, `DocumentNode`, `ElementNode`, etc.)
- **Functions**: `Parse`, `ParseFragment`, `Render`, `EscapeString`, `UnescapeString`, `NewTokenizer`, `NewTokenizerFragment`

---

## Thread Safety

`Processor` is safe for concurrent use:

```go
processor, _ := html.New()
defer processor.Close()

var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        processor.Extract(htmlBytes, html.DefaultExtractConfig())
    }()
}
wg.Wait()
```

---

## 🤝 Contributing

Contributions are welcome! Please read the contributing guidelines before submitting a PR.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

**Built with ❤️ for the Go community**

If this project helps you, please give it a Star!
