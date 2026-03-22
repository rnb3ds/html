# HTML Library

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://golang.org)
[![GoDoc](https://pkg.go.dev/badge/github.com/cybergodev/html.svg)](https://pkg.go.dev/github.com/cybergodev/html)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/cybergodev/html)](https://goreportcard.com/report/github.com/cybergodev/html)

**A high-performance Go library for intelligent HTML content extraction.** Drop-in replacement for `golang.org/x/net/html` with enhanced content extraction capabilities.

**[📖 中文文档](README_zh-CN.md)**

---

## 🎯 Why This Library?

| Feature | Description |
|---------|-------------|
| 🚀 **One-Line Extraction** | Extract clean text from HTML in a single function call |
| 🔍 **Smart Article Detection** | Identifies main content using scoring algorithms |
| 🌐 **Auto Encoding Detection** | Handles UTF-8, Windows-1252, GBK, Shift_JIS, etc. |
| 🔄 **Batch Processing** | Parallel extraction with Worker Pool and Context support |
| 📦 **Multiple Output Formats** | Text, Markdown, JSON |
| 🛡️ **Security First** | HTML sanitization, XSS protection, audit logging |
| 🧵 **Thread-Safe** | Concurrent use without external synchronization |
| 🔗 **golang.org/x/net/html Compatible** | Drop-in replacement with zero code changes |

---

## 📦 Installation

```bash
go get github.com/cybergodev/html
```

**Requirements**: Go 1.24+

---

## ⚡ 30-Second Quick Start

```go
package main

import (
    "fmt"
    "github.com/cybergodev/html"
)

func main() {
    // One-liner: extract clean text from HTML
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
    fmt.Println(text)
    // Output: "Hello World\nContent here..."
}
```

**What happens automatically:**
- ✅ Removes navigation, footers, ads, scripts
- ✅ Detects main content using scoring algorithms
- ✅ Handles character encoding (UTF-8, Windows-1252, GBK, etc.)
- ✅ Cleans up whitespace

---

## 🚀 Usage Guide

### 1️⃣ Package-Level Functions (Simplest)

For one-off extractions, use package-level functions:

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

    // Extract all content with metadata
    result, _ := html.Extract(htmlBytes)
    fmt.Println(result.Title)     // "Title"
    fmt.Println(result.Text)      // "Content here..."
    fmt.Println(result.WordCount) // 2

    // Extract all resource links
    links, _ := html.ExtractAllLinks(htmlBytes)

    // Format conversion
    markdown, _ := html.ExtractToMarkdown(htmlBytes)
    jsonData, _ := html.ExtractToJSON(htmlBytes)
}
```

---

### 2️⃣ Processor Usage (Recommended for Multiple Extractions)

For multiple extractions, create a Processor to leverage caching and connection pooling:

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
    result, _ := processor.Extract(htmlBytes)

    // Extract from file
    result, _ = processor.ExtractFromFile("page.html")

    // Batch processing
    htmlContents := [][]byte{htmlBytes, htmlBytes, htmlBytes}
    results, _ := processor.ExtractBatch(htmlContents)

    fmt.Printf("Processed %d documents\n", len(results))
}
```

---

### 3️⃣ Custom Configuration

```go
package main

import (
    "fmt"
    "github.com/cybergodev/html"
)

func main() {
    htmlBytes := []byte(`<html><body><h1>Title</h1><img src="img.jpg"><p>Content</p></body></html>`)

    // Start from DefaultConfig and customize
    config := html.DefaultConfig()
    config.PreserveVideos = false       // Skip videos
    config.PreserveAudios = false       // Skip audio
    config.InlineImageFormat = "none"   // Options: "none", "markdown", "html", "placeholder"
    config.InlineLinkFormat = "none"    // Options: "none", "markdown", "html"
    config.TableFormat = "markdown"     // Options: "markdown", "html"

    processor, _ := html.New(config)
    defer processor.Close()

    result, _ := processor.Extract(htmlBytes)
    fmt.Printf("Found %d images\n", len(result.Images))
}
```

---

### 4️⃣ Preset Configurations

```go
// Text only - no media preservation
processor, _ := html.New(html.TextOnlyConfig())

// Markdown output - images formatted as markdown
processor, _ := html.New(html.MarkdownConfig())

// Default - all features enabled
processor, _ := html.New(html.DefaultConfig())

// High security - stricter limits for untrusted input
processor, _ := html.New(html.HighSecurityConfig())
```

---

### 5️⃣ Advanced Configuration

```go
package main

import (
    "time"
    "github.com/cybergodev/html"
)

func main() {
    config := html.Config{
        MaxInputSize:       10 * 1024 * 1024, // 10MB limit
        ProcessingTimeout:  30 * time.Second,
        MaxCacheEntries:    500,
        CacheTTL:           30 * time.Minute,
        CacheCleanup:       5 * time.Minute,  // Background cleanup interval
        WorkerPoolSize:     8,
        EnableSanitization: true,  // Remove <script>, <style> tags
        MaxDepth:           50,    // Prevent deeply nested attacks
    }
    processor, _ := html.New(config)
    defer processor.Close()
}
```

---

## 📖 Common Examples

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

### Extract All Links

```go
processor, _ := html.New()
defer processor.Close()

links, _ := processor.ExtractAllLinks(htmlBytes)
for _, link := range links {
    fmt.Printf("%s: %s\n", link.Type, link.URL)
}

// Group by type
byType := html.GroupLinksByType(links)
cssLinks := byType["css"]
jsLinks := byType["js"]
images := byType["image"]
```

### Get Reading Time

```go
result, _ := html.Extract(htmlBytes)
minutes := result.ReadingTime.Minutes()
fmt.Printf("Reading time: %.1f minutes", minutes)
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
    result := processor.ExtractBatchFilesWithContext(ctx, files)

    fmt.Printf("Success: %d, Failed: %d, Cancelled: %d\n",
        result.Success, result.Failed, result.Cancelled)
}
```

### Caching & Statistics

```go
processor, _ := html.New()
defer processor.Close()

htmlBytes := []byte(`<html><body><p>Content</p></body></html>`)

// Caching is automatically enabled
processor.Extract(htmlBytes)
processor.Extract(htmlBytes) // Cache hit!

// View performance statistics
stats := processor.GetStatistics()
fmt.Printf("Cache hits: %d/%d\n", stats.CacheHits, stats.TotalProcessed)

// Clear cache (preserves statistics)
processor.ClearCache()

// Reset statistics (preserves cache entries)
processor.ResetStatistics()
```

---

## 🔧 API Quick Reference

### Package-Level Functions

```go
// Extract (from bytes)
html.Extract(htmlBytes []byte, cfg ...Config) (*Result, error)
html.ExtractText(htmlBytes []byte, cfg ...Config) (string, error)

// Extract (from file)
html.ExtractFromFile(filePath string, cfg ...Config) (*Result, error)
html.ExtractTextFromFile(filePath string, cfg ...Config) (string, error)

// Format conversion (from bytes)
html.ExtractToMarkdown(htmlBytes []byte, cfg ...Config) (string, error)
html.ExtractToJSON(htmlBytes []byte, cfg ...Config) ([]byte, error)

// Format conversion (from file)
html.ExtractToMarkdownFromFile(filePath string, cfg ...Config) (string, error)
html.ExtractToJSONFromFile(filePath string, cfg ...Config) ([]byte, error)

// Links
html.ExtractAllLinks(htmlBytes []byte, cfg ...Config) ([]LinkResource, error)
html.ExtractAllLinksFromFile(filePath string, cfg ...Config) ([]LinkResource, error)
html.GroupLinksByType(links []LinkResource) map[string][]LinkResource
```

### Processor Methods

```go
// Creation
processor, err := html.New()
processor, err := html.New(config)
processor, err := html.New(html.HighSecurityConfig())
processor, err := html.New(html.TextOnlyConfig())
processor, err := html.New(html.MarkdownConfig())
defer processor.Close()

// Extract (from bytes)
processor.Extract(htmlBytes []byte) (*Result, error)
processor.ExtractText(htmlBytes []byte) (string, error)
processor.ExtractWithContext(ctx context.Context, htmlBytes []byte) (*Result, error)

// Extract (from file)
processor.ExtractFromFile(filePath string) (*Result, error)
processor.ExtractTextFromFile(filePath string) (string, error)
processor.ExtractFromFileWithContext(ctx context.Context, filePath string) (*Result, error)

// Format conversion
processor.ExtractToMarkdown(htmlBytes []byte) (string, error)
processor.ExtractToJSON(htmlBytes []byte) ([]byte, error)
processor.ExtractToMarkdownFromFile(filePath string) (string, error)
processor.ExtractToJSONFromFile(filePath string) ([]byte, error)

// Links
processor.ExtractAllLinks(htmlBytes []byte) ([]LinkResource, error)
processor.ExtractAllLinksFromFile(filePath string) ([]LinkResource, error)
processor.ExtractAllLinksWithContext(ctx context.Context, htmlBytes []byte) ([]LinkResource, error)

// Batch processing
processor.ExtractBatch(contents [][]byte) ([]*Result, error)
processor.ExtractBatchFiles(paths []string) ([]*Result, error)
processor.ExtractBatchWithContext(ctx context.Context, contents [][]byte) *BatchResult
processor.ExtractBatchFilesWithContext(ctx context.Context, paths []string) *BatchResult

// Monitoring
processor.GetStatistics() Statistics
processor.ClearCache()
processor.ResetStatistics()
processor.GetAuditLog() []AuditEntry
processor.ClearAuditLog()
```

### Configuration Presets

```go
html.DefaultConfig() Config        // Standard configuration
html.HighSecurityConfig() Config   // Security-optimized configuration
html.TextOnlyConfig() Config       // Text-only (no media)
html.MarkdownConfig() Config       // Markdown image format
```

---

## 📋 Result Structures

```go
type Result struct {
    Text           string        `json:"text"`
    Title          string        `json:"title"`
    Images         []ImageInfo   `json:"images,omitempty"`
    Links          []LinkInfo    `json:"links,omitempty"`
    Videos         []VideoInfo   `json:"videos,omitempty"`
    Audios         []AudioInfo   `json:"audios,omitempty"`
    WordCount      int           `json:"word_count"`
    ReadingTime    time.Duration `json:"reading_time_ms"`
    ProcessingTime time.Duration `json:"processing_time_ms"`
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
    Position   int    `json:"position"`
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

## ⚙️ Configuration Reference

### Config Struct

```go
type Config struct {
    // === Resource Management ===
    MaxInputSize      int           // Maximum HTML input size (default: 50MB)
    MaxCacheEntries   int           // Maximum cache entries (default: 2000, 0=disabled)
    CacheTTL          time.Duration // Cache time-to-live (default: 1 hour)
    CacheCleanup      time.Duration // Background cleanup interval (default: 5 min)
    WorkerPoolSize    int           // Concurrent workers for batch (default: 4)
    ProcessingTimeout time.Duration // Max processing time (default: 30s, 0=no timeout)

    // === Security ===
    EnableSanitization bool        // HTML sanitization (default: true)
    MaxDepth           int         // Max HTML nesting depth (default: 500)
    Audit              AuditConfig // Security audit logging

    // === Content Extraction ===
    ExtractArticle bool // Enable article detection (default: true)
    PreserveImages bool // Extract images (default: true)
    PreserveLinks  bool // Extract links (default: true)
    PreserveVideos bool // Extract videos (default: true)
    PreserveAudios bool // Extract audios (default: true)

    // === Output Formats ===
    InlineImageFormat string // "none", "markdown", "html", "placeholder"
    InlineLinkFormat  string // "none", "markdown", "html"
    TableFormat       string // "markdown", "html"
    Encoding          string // Input encoding (empty=auto-detect)

    // === Link Extraction ===
    ResolveRelativeURLs  bool   // Resolve relative URLs (default: true)
    BaseURL              string // Base URL for resolution
    IncludeImages        bool   // Include image URLs (default: true)
    IncludeVideos        bool   // Include video URLs (default: true)
    IncludeAudios        bool   // Include audio URLs (default: true)
    IncludeCSS           bool   // Include CSS URLs (default: true)
    IncludeJS            bool   // Include JS URLs (default: true)
    IncludeContentLinks  bool   // Include anchor links (default: true)
    IncludeExternalLinks bool   // Include external links (default: true)
    IncludeIcons         bool   // Include favicon URLs (default: true)
}
```

### Default Configuration Values

| Setting | Default | High Security |
|---------|---------|---------------|
| MaxInputSize | 50 MB | 10 MB |
| MaxCacheEntries | 2000 | 500 |
| CacheTTL | 1 hour | 30 min |
| CacheCleanup | 5 min | 1 min |
| WorkerPoolSize | 4 | 2 |
| ProcessingTimeout | 30s | 10s |
| MaxDepth | 500 | 100 |
| Audit | Disabled | Enabled |

---

## 🔒 Security Features

### HTML Sanitization
- **Dangerous Tag Removal**: `<script>`, `<style>`, `<noscript>`, `<iframe>`, `<embed>`, `<object>`, `<form>`, `<input>`, `<button>`
- **Event Handler Removal**: All `on*` attributes (onclick, onerror, onload, etc.)
- **Dangerous Protocol Blocking**: `javascript:`, `vbscript:`, `data:` (except safe media types)
- **XSS Protection**: Comprehensive sanitization

### Input Validation
- **Size Limits**: Configurable `MaxInputSize` prevents memory exhaustion
- **Depth Limits**: `MaxDepth` prevents stack overflow from deeply nested HTML
- **Timeout Protection**: `ProcessingTimeout` prevents hanging
- **Path Traversal Protection**: `ExtractFromFile` validates file paths

### Data URL Security
- **Allowed**: `data:image/*`, `data:font/*`, `data:application/pdf`
- **Blocked**: `data:text/html`, `data:text/javascript`, `data:text/plain`

---

## 🔍 Audit Logging

Enable audit logging for security compliance:

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

// Get audit logs
entries := processor.GetAuditLog()
for _, entry := range entries {
    fmt.Printf("[%s] %s: %s\n", entry.Level, entry.EventType, entry.Message)
}
```

### Custom Audit Sinks

```go
// Write audit logs to file
file, _ := os.Create("audit.log")
fileSink := html.NewWriterAuditSink(file)

// Filter to critical events only
filteredSink := html.NewLevelFilteredSink(fileSink, html.AuditLevelCritical)

// Use in configuration
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

## 📁 Example Code

For complete runnable examples, see the [examples/](examples) directory:

| Example | Description |
|---------|-------------|
| [01_quick_start.go](examples/01_quick_start.go) | Quick start guide |
| [02_content_extraction.go](examples/02_content_extraction.go) | Content extraction options and output formats |
| [03_links_media.go](examples/03_links_media.go) | Link and media extraction |
| [04_performance.go](examples/04_performance.go) | Performance optimization and batch processing |
| [05_http_integration.go](examples/05_http_integration.go) | HTTP integration patterns |
| [06_advanced_usage.go](examples/06_advanced_usage.go) | Custom scorers, audit logging, security |
| [07_error_handling.go](examples/07_error_handling.go) | Error handling patterns |
| [08_real_world.go](examples/08_real_world.go) | Real-world use cases |

---

## 🔄 Compatibility

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

Re-exported types, constants, and functions:
- **Types**: `Node`, `NodeType`, `Token`, `Attribute`, `Tokenizer`, `ParseOption`
- **Constants**: All `NodeType` and `TokenType` constants (`ErrorNode`, `TextNode`, `DocumentNode`, `ElementNode`, etc.)
- **Functions**: `Parse`, `ParseFragment`, `Render`, `EscapeString`, `UnescapeString`, `NewTokenizer`, `NewTokenizerFragment`

---

## 🧵 Thread Safety

`Processor` is safe for concurrent use:

```go
processor, _ := html.New()
defer processor.Close()

var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        processor.Extract(htmlBytes)
    }()
}
wg.Wait()
```

---

## 🎯 Use Cases

- **News Aggregators**: Extract article content from news websites
- **Web Crawlers**: Fetch structured data from HTML pages
- **Content Management**: Convert HTML to Markdown or other formats
- **Search Engines**: Index main content, excluding navigation and ads
- **Data Analysis**: Extract and analyze web content at scale
- **RSS Feed Generators**: Extract content for feed creation
- **Archive Tools**: Preserve web page content

---

## 📄 License

MIT License - See [LICENSE](LICENSE) file for details.

---

If this project helps you, please give it a Star! ⭐
